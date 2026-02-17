// trophy - Terminal 3D Model Viewer
// View OBJ and GLB files in your terminal with full 3D rendering.
//
// Controls:
//
//	Mouse drag  - Rotate model (yaw/pitch)
//	Scroll      - Zoom in/out
//	W/S         - Pitch up/down
//	A/D         - Yaw left/right
//	Q/E         - Roll left/right (Q rolls left, E rolls right)
//	Space       - Apply random impulse
//	R           - Reset rotation
//	T           - Toggle texture on/off
//	X           - Toggle wireframe mode (x-ray)
//	L           - Light positioning mode (move mouse, click to set, Esc to cancel)
//	?           - Toggle HUD overlay (FPS, filename, poly count, mode status)
//	+/-         - Adjust zoom
//	Esc         - Quit (or cancel light mode)
package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fortio.org/terminal/ansipixels"
	"github.com/charmbracelet/harmonica"
	"github.com/spf13/cobra"
	"github.com/taigrr/trophy/pkg/math3d"
	"github.com/taigrr/trophy/pkg/models"
	"github.com/taigrr/trophy/pkg/render"
)

var (
	texturePath string
	targetFPS   int
	bgColor     string
)

func main() {
	cmd := &cobra.Command{
		Use:   "trophy <model.obj|model.glb|model.stl>",
		Short: "Terminal 3D Model Viewer",
		Long: `trophy - Terminal 3D Model Viewer

View OBJ and GLB files in your terminal with full 3D rendering.

Controls:
  Mouse drag  - Rotate model
  Scroll      - Zoom in/out
  W/S/A/D     - Pitch and yaw
  Q/E         - Roll left/right
  Space       - Random spin
  R           - Reset view
  T           - Toggle texture
  X           - Toggle wireframe
  L           - Position light (mouse to aim, click to set)
  ?           - Toggle HUD overlay
  Esc         - Quit`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args[0])
		},
	}

	cmd.Flags().StringVar(&texturePath, "texture", "", "Path to texture image (PNG/JPG)")
	cmd.Flags().IntVar(&targetFPS, "fps", 60, "Target FPS")
	cmd.Flags().StringVar(&bgColor, "bg", "", "Background color (R,G,B)")

	// Add info subcommand
	infoCmd := &cobra.Command{
		Use:   "info <model.obj|model.glb|model.stl>",
		Short: "Display model information",
		Long:  "Display detailed information about a 3D model file including format, polygon count, vertex count, and bounding box.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(args[0])
		},
	}
	cmd.AddCommand(infoCmd)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runInfo(modelPath string) error {
	ext := strings.ToLower(filepath.Ext(modelPath))

	// Check file exists
	info, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}

	var mesh *models.Mesh
	var hasEmbeddedTexture bool
	var textureSize string

	switch ext {
	case ".glb", ".gltf":
		var img image.Image
		mesh, img, err = models.LoadGLBWithTexture(modelPath)
		if err != nil {
			return fmt.Errorf("load model: %w", err)
		}
		if img != nil {
			hasEmbeddedTexture = true
			bounds := img.Bounds()
			textureSize = fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())
		}
	case ".obj":
		mesh, err = models.LoadOBJ(modelPath)
		if err != nil {
			return fmt.Errorf("load model: %w", err)
		}
	case ".stl":
		mesh, err = models.LoadSTL(modelPath)
		if err != nil {
			return fmt.Errorf("load model: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s (use .obj, .glb, or .stl)", ext)
	}

	mesh.CalculateBounds()
	size := mesh.Size()
	center := mesh.Center()

	// Format output
	fmt.Printf("File:       %s\n", filepath.Base(modelPath))
	fmt.Printf("Format:     %s\n", strings.ToUpper(strings.TrimPrefix(ext, ".")))
	fmt.Printf("Size:       %.2f KB\n", float64(info.Size())/1024)
	fmt.Println()
	fmt.Printf("Vertices:   %d\n", mesh.VertexCount())
	fmt.Printf("Triangles:  %d\n", mesh.TriangleCount())
	fmt.Println()
	fmt.Printf("Bounds Min: (%.3f, %.3f, %.3f)\n", mesh.BoundsMin.X, mesh.BoundsMin.Y, mesh.BoundsMin.Z)
	fmt.Printf("Bounds Max: (%.3f, %.3f, %.3f)\n", mesh.BoundsMax.X, mesh.BoundsMax.Y, mesh.BoundsMax.Z)
	fmt.Printf("Dimensions: %.3f x %.3f x %.3f\n", size.X, size.Y, size.Z)
	fmt.Printf("Center:     (%.3f, %.3f, %.3f)\n", center.X, center.Y, center.Z)

	if hasEmbeddedTexture {
		fmt.Println()
		fmt.Printf("Texture:    embedded (%s)\n", textureSize)
	}

	return nil
}

// RotationAxis tracks position and velocity for one rotation axis with spring decay
type RotationAxis struct {
	Position  float64
	Velocity  float64
	velSpring harmonica.Spring
	velAccel  float64 // internal spring velocity (for animating Velocity toward 0)
}

// NewRotationAxis creates an axis with harmonica spring for smooth velocity decay
func NewRotationAxis(fps int) RotationAxis {
	return RotationAxis{
		// Frequency 4.0 = moderate speed, damping 1.0 = critically damped (no overshoot)
		velSpring: harmonica.NewSpring(harmonica.FPS(fps), 4.0, 1.0),
	}
}

// Update applies velocity to position and decays velocity toward 0 using spring
func (a *RotationAxis) Update(damping bool) {
	// Apply velocity to position
	a.Position += a.Velocity

	// Use spring to animate velocity toward 0 (smooth deceleration)
	if damping {
		a.Velocity, a.velAccel = a.velSpring.Update(a.Velocity, a.velAccel, 0)
	}
}

// RotationState holds rotation with harmonica spring physics
type RotationState struct {
	Pitch, Yaw, Roll RotationAxis
	fps              int
}

func NewRotationState(fps int) *RotationState {
	return &RotationState{
		Pitch: NewRotationAxis(fps),
		Yaw:   NewRotationAxis(fps),
		Roll:  NewRotationAxis(fps),
		fps:   fps,
	}
}

func (r *RotationState) Update(damping bool) {
	r.Pitch.Update(damping)
	r.Yaw.Update(damping)
	r.Roll.Update(damping)
}

func (r *RotationState) ApplyImpulse(pitch, yaw, roll float64) {
	r.Pitch.Velocity += pitch
	r.Yaw.Velocity += yaw
	r.Roll.Velocity += roll
}

func (r *RotationState) Reset() {
	r.Pitch = NewRotationAxis(r.fps)
	r.Yaw = NewRotationAxis(r.fps)
	r.Roll = NewRotationAxis(r.fps)
}

// RenderMode controls how the mesh is drawn
type RenderMode int

const (
	RenderModeTextured  RenderMode = iota // Textured with Gouraud shading
	RenderModeFlat                        // Flat shading (no texture)
	RenderModeWireframe                   // Wireframe only
)

// ViewState holds all view-related settings (UI state, not library code)
type ViewState struct {
	TextureEnabled bool        // Whether to show textures
	RenderMode     RenderMode  // Current render mode
	LightMode      bool        // Whether in light positioning mode
	LightDir       math3d.Vec3 // Current light direction
	PendingLight   math3d.Vec3 // Light direction while positioning
	ShowHUD        bool        // Whether to show the HUD overlay
	SpinMode       bool        // Whether auto-spin is enabled
	BackfaceCull   bool        // Whether to cull backfaces (true = cull, false = show both sides)
}

// NewViewState creates default view state
func NewViewState() *ViewState {
	return &ViewState{
		TextureEnabled: true,
		RenderMode:     RenderModeTextured,
		LightMode:      false,
		LightDir:       math3d.V3(0.5, 1, 0.3).Normalize(),
		BackfaceCull:   false, // Default OFF - most STL files are single-sided shells
	}
}

// HUD renders an overlay with model info and controls
type HUD struct {
	filename  string
	polyCount int
	fps       float64
	fpsFrames int
	fpsTime   time.Time
	state     *ViewState
}

// NewHUD creates a new HUD
func NewHUD(filename string, polyCount int, state *ViewState) *HUD {
	return &HUD{
		filename:  filename,
		polyCount: polyCount,
		fpsTime:   time.Now(),
		state:     state,
	}
}

// UpdateFPS updates the FPS counter (call once per frame)
func (h *HUD) UpdateFPS() {
	h.fpsFrames++
	elapsed := time.Since(h.fpsTime)
	if elapsed >= time.Second {
		h.fps = float64(h.fpsFrames) / elapsed.Seconds()
		h.fpsFrames = 0
		h.fpsTime = time.Now()
	}
}

// Draw renders the HUD overlay to the terminal using ansipixels
func (h *HUD) Draw(ap *ansipixels.AnsiPixels) {
	if h.state.LightMode {
		// Light mode indicator
		ap.WriteAtStr(0, ap.H-1, "◉ LIGHT MODE - Move mouse to position, click to set, Esc to cancel")
		return
	}

	if !h.state.ShowHUD {
		return
	}

	// Top left: FPS
	ap.WriteAt(0, 0, "%.0f FPS ", h.fps)

	// Top middle: filename
	midX := (ap.W - len(h.filename)) / 2
	if midX > 0 {
		ap.WriteAtStr(midX, 0, h.filename)
	}

	// Top right: polygon count
	rightX := ap.W - 10 // Reserve space for "NNN polys"
	if rightX > 0 {
		ap.WriteAt(rightX, 0, "%d polys", h.polyCount)
	}

	// Bottom: mode indicators
	checkTex := "[ ]"
	if h.state.TextureEnabled && h.state.RenderMode != RenderModeWireframe {
		checkTex = "[✓]"
	}
	checkWire := "[ ]"
	if h.state.RenderMode == RenderModeWireframe {
		checkWire = "[✓]"
	}

	ap.WriteAt(0, ap.H-1, "%s Texture  %s X-Ray (wireframe)", checkTex, checkWire)

	// Bottom right: light hint
	lightStr := "L: position light"
	lightX := ap.W - len(lightStr)
	if lightX > 0 {
		ap.WriteAtStr(lightX, ap.H-1, lightStr)
	}
}

// ScreenToLightDir converts a screen position to a light direction.
// Maps screen coords to a hemisphere above the object.
func (v *ViewState) ScreenToLightDir(screenX, screenY, width, height int) math3d.Vec3 {
	// Normalize to [-1, 1]
	nx := (float64(screenX)/float64(width))*2 - 1
	ny := (float64(screenY)/float64(height))*2 - 1

	// Clamp to unit circle
	lenSq := nx*nx + ny*ny
	if lenSq > 1 {
		len := math.Sqrt(lenSq)
		nx /= len
		ny /= len
		lenSq = 1
	}

	// Z component (hemisphere projection)
	nz := math.Sqrt(1 - lenSq)

	// Return as light direction (pointing toward the object)
	return math3d.V3(nx, -ny, nz).Normalize()
}

func run(modelPath string) (err error) {
	// Parse background color
	var bg color.RGBA
	if bgColor != "" {
		_, err := fmt.Sscanf(bgColor, "%d,%d,%d", &bg.R, &bg.G, &bg.B)
		if err == nil {
			bg.A = 255
		}
	}

	// Initialize ansipixels for terminal rendering
	ap := ansipixels.NewAnsiPixels(float64(targetFPS))
	if err := ap.Open(); err != nil {
		return fmt.Errorf("open ansipixels: %w", err)
	}
	defer func() {
		ap.ShowCursor()
		ap.MouseTrackingOff()
		ap.Out.Flush()
		ap.Restore()
	}()
	ap.SyncBackgroundColor()
	ap.MouseTrackingOn()
	ap.HideCursor()

	// Get terminal dimensions
	termWidth := ap.W
	termHeight := ap.H
	if termWidth <= 0 || termHeight <= 0 {
		return fmt.Errorf("invalid terminal size: %dx%d", termWidth, termHeight)
	}

	// Create renderer with framebuffer sized for terminal
	// Using 2x height for half-block characters
	fb := render.NewFramebuffer(termWidth, termHeight*2)
	fb.BG = bg

	// Create camera
	camera := render.NewCamera()
	camera.SetAspectRatio(float64(fb.Width) / float64(fb.Height))
	camera.SetFOV(math.Pi / 3)
	camera.SetClipPlanes(0.1, 100)
	camera.SetPosition(math3d.V3(0, 0, 5))
	camera.LookAt(math3d.V3(0, 0, 0))

	rasterizer := render.NewRasterizer(camera, fb)

	// Load texture if specified
	var texture *render.Texture
	if texturePath != "" {
		texture, err = render.LoadTexture(texturePath)
		if err != nil {
			fmt.Printf("Warning: could not load texture: %v\n", err)
		}
	}

	// Load model
	ext := strings.ToLower(filepath.Ext(modelPath))
	var mesh *models.Mesh

	switch ext {
	case ".glb", ".gltf":
		var embeddedImg image.Image
		mesh, embeddedImg, err = models.LoadGLBWithTexture(modelPath)
		if err != nil {
			return fmt.Errorf("load model: %w", err)
		}
		// Use embedded texture if no explicit texture and one exists
		if texture == nil && embeddedImg != nil {
			texture = render.TextureFromImage(embeddedImg)
			fmt.Printf("Using embedded texture: %dx%d\n", embeddedImg.Bounds().Dx(), embeddedImg.Bounds().Dy())
		}
	case ".obj":
		mesh, err = models.LoadOBJ(modelPath)
		if err != nil {
			return fmt.Errorf("load model: %w", err)
		}
	case ".stl":
		mesh, err = models.LoadSTL(modelPath)
		if err != nil {
			return fmt.Errorf("load model: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s (use .obj, .glb, or .stl)", ext)
	}

	// Generate fallback texture if none
	if texture == nil {
		texture = render.NewCheckerTexture(64, 64, 8, render.RGB(200, 200, 200), render.RGB(100, 100, 100))
	}

	fmt.Printf("Loaded: %s (%d vertices, %d triangles)\n", filepath.Base(modelPath), mesh.VertexCount(), mesh.TriangleCount())

	// Initialize rotation and view state
	rotation := NewRotationState(targetFPS)
	viewState := NewViewState()

	// Create HUD
	hud := NewHUD(filepath.Base(modelPath), mesh.TriangleCount(), viewState)

	// Center and scale model
	mesh.CalculateBounds()
	center := mesh.Center()
	size := mesh.Size()
	maxDim := math.Max(size.X, math.Max(size.Y, size.Z))
	if maxDim > 0 {
		scale := 2.0 / maxDim
		transform := math3d.Scale(math3d.V3(scale, scale, scale)).Mul(math3d.Translate(center.Scale(-1)))
		mesh.Transform(transform)
	}
	// Input state
	inputTorque := struct{ pitch, yaw, roll float64 }{}
	const torqueStrength = 3.0

	// Main loop
	targetDuration := time.Second / time.Duration(targetFPS)
	lastFrame := time.Now()

	ap.OnMouse = func() {
		if !viewState.LightMode {
			return
		}
		// Convert screen coordinates to light direction
		viewState.PendingLight = viewState.ScreenToLightDir(ap.Mx, ap.My, termWidth, termHeight)

		// Check for mouse click to confirm light position
		if ap.MouseRelease() {
			viewState.LightDir = viewState.PendingLight
			viewState.LightMode = false
		}
	}

	for {
		now := time.Now()
		dt := now.Sub(lastFrame).Seconds()
		lastFrame = now

		if dt > 0.1 {
			dt = 0.1
		}

		// Read input
		_, err := ap.ReadOrResizeOrSignalOnce()
		if err != nil {
			return fmt.Errorf("input error: %w", err)
		}
		// Process keyboard input from ap.Data
		if len(ap.Data) > 0 {
			for _, b := range ap.Data {
				switch b {
				case 'q', 'Q':
					inputTorque.roll = -torqueStrength
				case 'e', 'E':
					inputTorque.roll = torqueStrength
				case 'w', 'W':
					inputTorque.pitch = -torqueStrength
				case 's', 'S':
					inputTorque.pitch = torqueStrength
				case 'a', 'A':
					inputTorque.yaw = -torqueStrength
				case 'd', 'D':
					inputTorque.yaw = torqueStrength
				case 'r', 'R':
					rotation.Reset()
					camera.SetPosition(math3d.V3(0, 0, 5))
				case 't', 'T':
					// Toggle texture
					viewState.TextureEnabled = !viewState.TextureEnabled
				case 'x', 'X':
					// Toggle wireframe mode
					if viewState.RenderMode == RenderModeWireframe {
						viewState.RenderMode = RenderModeTextured
					} else {
						viewState.RenderMode = RenderModeWireframe
					}
				case 'l', 'L':
					// Enter light positioning mode
					viewState.LightMode = true
					viewState.PendingLight = viewState.LightDir
				case 'b', 'B':
					// Toggle backface culling
					viewState.BackfaceCull = !viewState.BackfaceCull
				case '?':
					// Toggle HUD
					viewState.ShowHUD = !viewState.ShowHUD
				case '+', '=':
					// Zoom in
					camera.SetPosition(math3d.V3(0, 0, math.Max(1, camera.Position.Z-0.5)))
				case '-', '_':
					// Zoom out
					camera.SetPosition(math3d.V3(0, 0, math.Min(20, camera.Position.Z+0.5)))
				case ' ':
					// Toggle spin mode
					viewState.SpinMode = !viewState.SpinMode
					if viewState.SpinMode {
						rotation.Yaw.Velocity = 0.02
					}
				case 27: // Escape
					if viewState.LightMode {
						viewState.LightMode = false
					} else {
						return nil
					}
				}
			}
		}

		// Apply input torque and decay it
		rotation.ApplyImpulse(
			inputTorque.pitch*dt,
			inputTorque.yaw*dt,
			inputTorque.roll*dt,
		)
		inputTorque.pitch *= 0.9
		inputTorque.yaw *= 0.9
		inputTorque.roll *= 0.9

		// Update springs (harmonica handles timing internally)
		rotation.Update(!viewState.SpinMode)

		// Build transform
		transform := math3d.RotateX(rotation.Pitch.Position).
			Mul(math3d.RotateY(rotation.Yaw.Position)).
			Mul(math3d.RotateZ(rotation.Roll.Position))

		// Render
		fb.Clear()
		rasterizer.ClearDepth()

		// Choose light direction (pending if in light mode, otherwise current)
		lightDir := viewState.LightDir
		if viewState.LightMode {
			lightDir = viewState.PendingLight
		}

		// Set backface culling mode
		rasterizer.DisableBackfaceCulling = !viewState.BackfaceCull

		// Draw mesh based on render mode
		switch viewState.RenderMode {
		case RenderModeWireframe:
			// X-ray wireframe mode
			rasterizer.DrawMeshWireframe(mesh, transform, render.RGB(0, 255, 128))
		case RenderModeFlat:
			// Flat shading (no texture)
			rasterizer.DrawMeshGouraudOpt(mesh, transform, render.RGB(200, 200, 200), lightDir)
		default:
			// Textured mode
			if viewState.TextureEnabled {
				rasterizer.DrawMeshTexturedOpt(mesh, transform, texture, lightDir)
			} else {
				rasterizer.DrawMeshGouraudOpt(mesh, transform, render.RGB(200, 200, 200), lightDir)
			}
		}

		// Convert framebuffer to image for ansipixels
		img := fb.ToImage()

		// Display using ansipixels
		ap.StartSyncMode()
		ap.ClearScreen()
		if err := ap.ShowScaledImage(img); err != nil {
			return fmt.Errorf("show image: %w", err)
		}

		// HUD overlay
		hud.UpdateFPS()
		hud.Draw(ap)
		ap.EndSyncMode()

		// Frame timing
		elapsed := time.Since(now)
		if elapsed < targetDuration {
			time.Sleep(targetDuration - elapsed)
		}
	}
}
