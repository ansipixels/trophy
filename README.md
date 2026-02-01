# Trophy 🏆

Terminal 3D Model Viewer - View OBJ and GLB files directly in your terminal.

![Trophy Demo](docs/demo.png)

## Features

- **OBJ & GLB Support** - Load standard 3D model formats
- **Embedded Textures** - Automatically extracts and applies GLB textures
- **Interactive Controls** - Rotate, zoom, and spin models with mouse/keyboard
- **Software Rendering** - No GPU required, works over SSH
- **Springy Physics** - Smooth, satisfying rotation with momentum

## Installation

```bash
go install github.com/taigrr/trophy/cmd/trophy@latest
```

## Usage

```bash
trophy model.glb              # View a GLB model
trophy model.obj              # View an OBJ model
trophy -texture tex.png model.obj  # Apply custom texture
trophy -bg 0,0,0 model.glb    # Black background
trophy -fps 60 model.glb      # Higher framerate
```

## Controls

| Input | Action |
|-------|--------|
| Mouse drag | Rotate model |
| Scroll wheel | Zoom in/out |
| W/S | Pitch up/down |
| A/D | Yaw left/right |
| Q/E | Roll |
| Space | Random spin |
| +/- | Zoom |
| R | Reset view |
| Esc | Quit |

## Library Usage

Trophy's rendering packages can be used as a library:

```go
import (
    "github.com/taigrr/trophy/pkg/models"
    "github.com/taigrr/trophy/pkg/render"
    "github.com/taigrr/trophy/pkg/math3d"
)

// Load a model
mesh, texture, _ := models.LoadGLBWithTexture("model.glb")

// Create renderer
fb := render.NewFramebuffer(320, 200)
camera := render.NewCamera()
rasterizer := render.NewRasterizer(camera, fb)

// Render
rasterizer.DrawMeshTexturedGouraud(mesh, transform, texture, lightDir)
```

## Packages

- `pkg/math3d` - 3D math (Vec2, Vec3, Vec4, Mat4)
- `pkg/models` - Model loaders (OBJ, GLB/GLTF)
- `pkg/render` - Software rasterizer, camera, textures

## License

MIT

## Credits

Built with [Ultraviolet](https://github.com/charmbracelet/ultraviolet) for terminal rendering.
