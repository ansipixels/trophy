[![Go Reference](https://pkg.go.dev/badge/github.com/ansipixels/trophy.svg)](https://pkg.go.dev/github.com/ansipixels/trophy)
[![Go Report Card](https://goreportcard.com/badge/github.com/ansipixels/trophy)](https://goreportcard.com/report/github.com/ansipixels/trophy)
[![GitHub Release](https://img.shields.io/github/release/ansipixels/trophy.svg?style=flat)](https://github.com/ansipixels/trophy/releases/)
[![CI Checks](https://github.com/ansipixels/trophy/actions/workflows/include.yml/badge.svg)](https://github.com/ansipixels/trophy/actions/workflows/include.yml)
[![codecov](https://codecov.io/github/ansipixels/trophy/graph/badge.svg?token=CODECOV_TOKEN)](https://codecov.io/github/ansipixels/trophy)


# Trophy 🏆

Terminal 3D Model Viewer - View OBJ, GLB, and STL files directly in your terminal.

![Trophy Demo](docs/trophy.gif)

## Features

- **OBJ, GLB & STL Support** - Load standard 3D model formats
- **Embedded Textures** - Automatically extracts and applies GLB textures
- **Interactive Controls** - Rotate, zoom, and spin models with mouse/keyboard
- **Software Rendering** - No GPU required, works over SSH
- **Springy Physics** - Smooth, satisfying rotation with momentum

## Installation

See the [binary releases](releases/) or

From source/with go:
```bash
go install github.com/ansipixels/trophy@latest
```

Or on mac
```
brew install ansipixels/tap/trophy
```

Or even with docker (slower than native though)
```sh
docker run -ti ghcr.io/ansipixels/trophy # default Trophy model demo
# Or
docker run -v `pwd`:/data -ti ghcr.io/ansipixels/trophy ./yourmodel.glb
```

## Usage

```bash
trophy help                   # See all the options and flags
trophy -ls                    # list the res:* embedded models
trophy res:teapot.stl         # Run the viewer for the Teapot embedded model
trophy                        # Run the viewer for the Trophy embedded model
trophy model.glb              # View a GLB model
trophy model.obj              # View an OBJ model
trophy model.stl              # View an STL model
trophy -texture tex.png model.obj  # Apply custom texture
trophy -fps 60 model.glb      # Higher framerate
```

## Controls

| Input        | Action                |
| ------------ | --------------------- |
| Mouse drag   | Rotate model          |
| Scroll wheel | Zoom in/out           |
| W/S          | Pitch up/down         |
| A/D          | Yaw left/right        |
| Q/E          | Roll                  |
| Space        | Toggle spin mode      |
| +/-          | Zoom                  |
| R            | Reset view            |
| T            | Toggle texture        |
| X            | Toggle wireframe      |
| B            | Toggle backface cull  |
| L            | Position light        |
| ?            | Toggle HUD overlay    |
| Esc          | Quit                  |

## Lighting

Press `L` to enter lighting mode and drag to reposition the light source in real-time:

![Lighting Demo](docs/lighting-demo.gif)

## Library Usage

Trophy's rendering packages can be used as a library:

```go
import (
    "github.com/ansipixels/trophy/models"
    "github.com/ansipixels/trophy/render"
    "github.com/ansipixels/trophy/math3d"
)

// Load a model
mesh, texture, _ := models.LoadGLBWithTexture("model.glb")

// Create renderer
fb := render.NewFramebuffer(320, 200)
camera := render.NewCamera()
rasterizer := render.NewRasterizer(camera, fb)

// Render (uses optimized edge-function rasterizer)
rasterizer.DrawMeshTexturedOpt(mesh, transform, texture, lightDir)
```

## Packages

- `math3d` - 3D math (Vec2, Vec3, Vec4, Mat4)
- `models` - Model loaders (OBJ, GLB/GLTF, STL)
- `render` - Software rasterizer, camera, textures

## Benchmarks

Run with `go test -bench=. -benchmem ./...`

### Math (math3d)

| Benchmark      | ns/op | B/op | allocs/op |
| -------------- | ----: | ---: | --------: |
| Mat4Mul        | 74.86 |    0 |         0 |
| Mat4MulVec4    |  6.84 |    0 |         0 |
| Mat4MulVec3    |  8.24 |    0 |         0 |
| Mat4Inverse    | 62.13 |    0 |         0 |
| Vec3Normalize  |  7.41 |    0 |         0 |
| Vec3Cross      |  2.47 |    0 |         0 |
| Vec3Dot        |  2.48 |    0 |         0 |
| Perspective    | 24.15 |    0 |         0 |
| LookAt         | 33.66 |    0 |         0 |
| ViewProjection | 71.32 |    0 |         0 |

### Rendering (render)

| Benchmark                  |  ns/op | B/op | allocs/op |
| -------------------------- | -----: | ---: | --------: |
| FrustumExtract             |  37.03 |    0 |         0 |
| AABBIntersection (visible) |   9.96 |    0 |         0 |
| AABBIntersection (culled)  |   7.76 |    0 |         0 |
| TransformAABB              |  125.8 |    0 |         0 |
| FrustumIntersectAABB       |   7.31 |    0 |         0 |
| FrustumIntersectsSphere    |   4.91 |    0 |         0 |
| DrawTriangleGouraud        |   4858 |    0 |         0 |
| DrawTriangleGouraudOpt     |   3981 |    0 |         0 |
| DrawMeshGouraud            |  55741 |    0 |         0 |
| DrawMeshGouraudOpt         |  27868 |    0 |         0 |

### Culling Performance

| Benchmark                       |  ns/op | B/op | allocs/op |
| ------------------------------- | -----: | ---: | --------: |
| MeshRendering (with culling)    | 115793 |    0 |         0 |
| MeshRendering (without culling) | 141766 |    0 |         0 |

The optimized rasterizer (`*Opt` variants) uses incremental edge functions instead of per-pixel barycentric recomputation, yielding **~18% speedup** on triangles and **~50% speedup** on full mesh rendering.

_Benchmarks run on AMD EPYC 7642 48-Core, linux/amd64_

## Credits

The  (Awesome!) original [github.com/taigrr/trophy](https://github.com/taigrr/trophy) was built with [ultraviolet](https://github.com/charmbracelet/ultraviolet) for terminal rendering.

This is the [Ansipixels](https://github.com/fortio/terminal/#fortioorgterminalansipixels) TUI version which 30% smaller binary and has faster/more stable FPS and I believe clearer/simpler main loop code.
