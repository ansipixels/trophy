package render

import "github.com/ansipixels/trophy/math3d"

func cubeVertices(center math3d.Vec3, half float64) [8]math3d.Vec3 {
	return [8]math3d.Vec3{
		{X: center.X - half, Y: center.Y - half, Z: center.Z - half},
		{X: center.X + half, Y: center.Y - half, Z: center.Z - half},
		{X: center.X + half, Y: center.Y + half, Z: center.Z - half},
		{X: center.X - half, Y: center.Y + half, Z: center.Z - half},
		{X: center.X - half, Y: center.Y - half, Z: center.Z + half},
		{X: center.X + half, Y: center.Y - half, Z: center.Z + half},
		{X: center.X + half, Y: center.Y + half, Z: center.Z + half},
		{X: center.X - half, Y: center.Y + half, Z: center.Z + half},
	}
}

func cubeLocalVertices(half float64) [8]math3d.Vec3 {
	return [8]math3d.Vec3{
		{X: -half, Y: -half, Z: -half},
		{X: half, Y: -half, Z: -half},
		{X: half, Y: half, Z: -half},
		{X: -half, Y: half, Z: -half},
		{X: -half, Y: -half, Z: half},
		{X: half, Y: -half, Z: half},
		{X: half, Y: half, Z: half},
		{X: -half, Y: half, Z: half},
	}
}
