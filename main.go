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
	"embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fortio.org/cli"
	"fortio.org/log"
	"fortio.org/terminal/ansipixels"
	"fortio.org/terminal/ansipixels/tcolor"
	"github.com/ansipixels/trophy/math3d"
	"github.com/ansipixels/trophy/models"
	"github.com/ansipixels/trophy/render"
	"github.com/charmbracelet/harmonica"
)

var (
	texturePath string
	targetFPS   float64
	// Embed default model files (GLB and STL only from docs/)
	//go:embed docs/*.glb docs/*.stl
	docsEmbedFS embed.FS
	// docsFS is the docs directory exposed as the root of the embedded filesystem.
	docsFS fs.FS
)

const embeddedPrefix = "res:"

func init() {
	var err error
	docsFS, err = fs.Sub(docsEmbedFS, "docs")
	if err != nil {
		panic(fmt.Sprintf("failed to create embedded filesystem: %v", err))
	}
}

func main() {
	flag.StringVar(&texturePath, "texture", "", "Path to texture image (PNG/JPG)")
	flag.Float64Var(&targetFPS, "fps", 60, "Target FPS")
	listEmbedded := flag.Bool("ls", false, "List embedded model options (res: files) and exit")
	cli.ArgsHelp = "<model.obj|model.glb|model.stl> (default: " + embeddedPrefix + "trophy.glb)"
	cli.MinArgs = 0
	cli.MaxArgs = 1
	cli.Main()
	// Handle -ls flag
	if *listEmbedded {
		entries, err := fs.ReadDir(docsFS, ".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading embedded files: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Embedded model options:")
		for _, entry := range entries {
			if !entry.IsDir() {
				fmt.Printf("  %s%s\n", embeddedPrefix, entry.Name())
			}
		}
		os.Exit(0)
	}

	// At this point, cli.Main has validated arguments
	var modelPath string
	if flag.NArg() > 0 {
		modelPath = flag.Arg(0)
	} else {
		modelPath = embeddedPrefix + "trophy.glb" // Use embedded model
	}
	os.Exit(run(modelPath))
}

// RotationAxis tracks position and velocity for one rotation axis with spring decay.
type RotationAxis struct {
	Position  float64
	Velocity  float64
	velSpring harmonica.Spring
	velAccel  float64 // internal spring velocity (for animating Velocity toward 0)
}

// NewRotationAxis creates an axis with harmonica spring for smooth velocity decay.
func NewRotationAxis(fps int) RotationAxis {
	return RotationAxis{
		// Frequency 4.0 = moderate speed, damping 1.0 = critically damped (no overshoot)
		velSpring: harmonica.NewSpring(harmonica.FPS(fps), 4.0, 1.0),
	}
}

// Update applies velocity to position and decays velocity toward 0 using spring.
func (a *RotationAxis) Update(damping bool) {
	// Apply velocity to position
	a.Position += a.Velocity

	// Use spring to animate velocity toward 0 (smooth deceleration)
	if damping {
		a.Velocity, a.velAccel = a.velSpring.Update(a.Velocity, a.velAccel, 0)
	}
}

// RotationState holds rotation with harmonica spring physics.
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

// RenderMode controls how the mesh is drawn.
type RenderMode int

const (
	RenderModeTextured  RenderMode = iota // Textured with Gouraud shading
	RenderModeFlat                        // Flat shading (no texture)
	RenderModeWireframe                   // Wireframe only
)

// ViewState holds all view-related settings (UI state, not library code).
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

// NewViewState creates default view state.
func NewViewState() *ViewState {
	return &ViewState{
		TextureEnabled: true,
		RenderMode:     RenderModeTextured,
		LightMode:      false,
		LightDir:       math3d.V3(0.5, 1, 0.3).Normalize(),
		BackfaceCull:   false, // Default OFF - most STL files are single-sided shells
	}
}

// HUD renders an overlay with model info and controls.
type HUD struct {
	filename  string
	polyCount int
	fps       float64
	fpsFrames int
	fpsTime   time.Time
	state     *ViewState
}

// NewHUD creates a new HUD.
func NewHUD(filename string, polyCount int, state *ViewState) *HUD {
	return &HUD{
		filename:  filename,
		polyCount: polyCount,
		fpsTime:   time.Now(),
		state:     state,
	}
}

// UpdateFPS updates the FPS counter (call once per frame).
func (h *HUD) UpdateFPS() {
	h.fpsFrames++
	elapsed := time.Since(h.fpsTime)
	if elapsed >= time.Second {
		h.fps = float64(h.fpsFrames) / elapsed.Seconds()
		h.fpsFrames = 0
		h.fpsTime = time.Now()
	}
}

// Draw renders the HUD overlay to the terminal using ansipixels.
func (h *HUD) Draw(ap *ansipixels.AnsiPixels) {
	if h.state.LightMode {
		// Light mode indicator
		ap.WriteCentered(ap.H-1, "%s◉ LIGHT MODE - Move mouse to position, click to set, Esc to cancel%s",
			tcolor.BrightYellow.Foreground(), tcolor.Reset)
		return
	}

	if !h.state.ShowHUD {
		return
	}

	// Top left: FPS
	ap.WriteAt(0, 0, tcolor.Green.Foreground()+"%.0f FPS "+tcolor.Reset, h.fps)

	// Top middle: filename
	ap.WriteCentered(0, "%s", h.filename)

	// Top right: polygon count
	ap.WriteRight(0, tcolor.Cyan.Foreground()+"%d polys"+tcolor.Reset, h.polyCount)

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
	ap.WriteRight(ap.H-1, "%sL: position light%s", tcolor.Yellow.Foreground(), tcolor.Reset)
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
		length := math.Sqrt(lenSq)
		nx /= length
		ny /= length
		lenSq = 1
	}

	// Z component (hemisphere projection)
	nz := math.Sqrt(1 - lenSq)

	// Return as light direction (pointing toward the object)
	return math3d.V3(nx, -ny, nz).Normalize()
}

// selectFilesystem resolves a file path to the appropriate filesystem.
// Supports "res:" URI prefix for explicit embedded files,
// or searches embedded FS first, then falls back to local FS.
// Returns (filesystem, path, isEmbedded, error).
// isEmbedded indicates whether the file comes from the embedded FS (vs local disk).
func selectFilesystem(modelPath string) (fs.FS, string, bool, error) {
	// Check for explicit res: prefix
	if strings.HasPrefix(modelPath, embeddedPrefix) {
		cleanPath := modelPath[len(embeddedPrefix):]
		// Verify it exists in embedded FS
		if _, err := fs.Stat(docsFS, cleanPath); err != nil {
			return nil, "", false, fmt.Errorf("file not found in embedded filesystem: %s", cleanPath)
		}
		return docsFS, cleanPath, true, nil
	}

	// Try embedded FS first
	if _, err := fs.Stat(docsFS, modelPath); err == nil {
		return docsFS, modelPath, true, nil
	}

	// Fall back to local FS
	if _, err := os.Stat(modelPath); err == nil {
		// Clean the path for os.DirFS compatibility (remove ./ prefix, etc.)
		cleanPath := filepath.Clean(modelPath)
		return os.DirFS("."), cleanPath, false, nil
	}

	return nil, "", false, fmt.Errorf("file not found in embedded or local filesystem: %s", modelPath)
}

// LoadModelFromFS loads a model from a filesystem interface (embed.FS or os.DirFS).
// If copyGLBToTemp is true, GLB files are read from fsys and written to a temp file
// since the gltf library requires a real file path.
// If copyGLBToTemp is false, the modelPath is used directly (for local files).
func LoadModelFromFS(fsys fs.FS, modelPath string, copyGLBToTemp bool) (*models.Mesh, image.Image, error) {
	ext := strings.ToLower(filepath.Ext(modelPath))

	switch ext {
	case ".glb", ".gltf":
		// For GLTF, the gltf library requires file path access, not abstract fs.FS
		if copyGLBToTemp {
			// Read from virtual FS and write to temp file
			data, err := fs.ReadFile(fsys, modelPath)
			if err != nil {
				return nil, nil, fmt.Errorf("read glb file: %w", err)
			}
			tempFile, err := os.CreateTemp("", "model-*.glb")
			if err != nil {
				return nil, nil, err
			}
			defer os.Remove(tempFile.Name())
			if _, err := tempFile.Write(data); err != nil {
				tempFile.Close()
				return nil, nil, err
			}
			tempFile.Close()
			return models.LoadGLBWithTexture(tempFile.Name())
		}
		// Local file - use the path directly with the gltf loader
		mesh, img, err := models.LoadGLBWithTexture(modelPath)
		return mesh, img, err

	case ".obj":
		mesh, err := models.LoadOBJFromFS(fsys, modelPath)
		return mesh, nil, err
	case ".stl":
		mesh, err := models.LoadSTLFromFS(fsys, modelPath)
		return mesh, nil, err
	default:
		return nil, nil, fmt.Errorf("unsupported format: %s (use .obj, .glb, or .stl)", ext)
	}
}

//nolint:gocognit,gocyclo,funlen,maintidx // yeah it's kinda long.
func run(modelPath string) int {
	// Resolve the filesystem based on the model path
	// Supports "res:" URI prefix or searches embedded first with fallback to local
	modelFS, resolvedPath, isEmbedded, err := selectFilesystem(modelPath)
	if err != nil {
		return log.FErrf("resolve model path: %v", err)
	}
	// Initialize ansipixels for terminal rendering
	ap := ansipixels.NewAnsiPixels(float64(targetFPS))
	if err = ap.Open(); err != nil {
		return log.FErrf("open ansipixels: %v", err)
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

	// Create renderer with framebuffer sized for terminal
	// Using 2x height for half-block characters
	fb := render.NewFramebuffer(ap.W, ap.H*2)
	fb.BG = color.RGBA{ap.Background.R, ap.Background.G, ap.Background.B, 255}

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
			return log.FErrf("Could not load texture: %v", err)
		}
	}

	// Load model
	var mesh *models.Mesh
	var embeddedImg image.Image

	mesh, embeddedImg, err = LoadModelFromFS(modelFS, resolvedPath, isEmbedded)
	if err != nil {
		return log.FErrf("load model: %v", err)
	}

	// Use embedded texture if no explicit texture and one exists
	if texture == nil && embeddedImg != nil {
		texture = render.TextureFromImage(embeddedImg)
		log.Infof("Using embedded texture: %dx%d", embeddedImg.Bounds().Dx(), embeddedImg.Bounds().Dy())
	}

	// Generate fallback texture if none
	if texture == nil {
		texture = render.NewCheckerTexture(64, 64, 8, render.RGB(200, 200, 200), render.RGB(100, 100, 100))
	}

	fmt.Printf("Loaded: %s (%d vertices, %d triangles)\n", filepath.Base(modelPath), mesh.VertexCount(), mesh.TriangleCount())

	// Initialize rotation and view state
	rotation := NewRotationState(int(math.Round(targetFPS)))
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
	lastFrame := time.Now()

	cameraZ := 5.0
	lastMouseX, lastMouseY := 0, 0

	ap.OnMouse = func() {
		switch {
		case ap.MouseWheelUp():
			cameraZ -= 0.5
			if cameraZ < 1 {
				cameraZ = 1
			}
		case ap.MouseWheelDown():
			cameraZ += 0.5
			if cameraZ > 20 {
				cameraZ = 20
			}
		case ap.LeftClick():
		case ap.LeftDrag():
			dx := ap.Mx - lastMouseX
			dy := ap.My - lastMouseY
			rotation.ApplyImpulse(float64(dy)*0.03, float64(dx)*0.03, 0)
		}
		camera.SetPosition(math3d.V3(0, 0, cameraZ))
		if viewState.LightMode {
			// Convert screen coordinates to light direction
			viewState.PendingLight = viewState.ScreenToLightDir(ap.Mx, ap.My, ap.W, ap.H)

			// Check for mouse click to confirm light position
			if ap.MouseRelease() {
				viewState.LightDir = viewState.PendingLight
				viewState.LightMode = false
			}
		}
		lastMouseX, lastMouseY = ap.Mx, ap.My
	}
	// Update framebuffer and camera aspect ratio on terminal resize
	ap.OnResize = func() error {
		fb.Resize(ap.W, ap.H*2)
		camera.SetAspectRatio(float64(fb.Width) / float64(fb.Height))
		return nil
	}

	now := time.Now()
	err = ap.FPSTicks(func() bool {
		dt := now.Sub(lastFrame).Seconds()
		lastFrame = now
		if dt > 0.1 {
			dt = 0.1
		}
		// Process keyboard input from ap.Data
		if len(ap.Data) > 0 { //nolint:nestif // it's just a big switch
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
					cameraZ = 5.0
					camera.SetPosition(math3d.V3(0, 0, cameraZ))
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
					cameraZ = max(1., cameraZ-0.5)
					camera.SetPosition(math3d.V3(0, 0, cameraZ))
				case '-', '_':
					// Zoom out
					cameraZ = min(20., cameraZ+0.5)
					camera.SetPosition(math3d.V3(0, 0, cameraZ))
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
						return false
					}
				case 3, 4: // Ctrl-C, Ctrl-D
					return false
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
		ap.ClearScreen()
		if err = ap.ShowScaledImage(img); err != nil {
			log.Errf("show image: %v", err)
			return false
		}
		// HUD overlay
		hud.UpdateFPS()
		hud.Draw(ap)
		return true // continue running
	})
	if err != nil {
		return log.FErrf("main loop: %v", err)
	}
	return 0
}
