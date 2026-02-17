package math3d

import "math"

// Vec4 represents a 4D vector (or homogeneous 3D point).
type Vec4 struct {
	X, Y, Z, W float64
}

// V4 creates a new Vec4.
func V4(x, y, z, w float64) Vec4 {
	return Vec4{x, y, z, w}
}

// V4FromV3 creates a Vec4 from Vec3 with specified W.
func V4FromV3(v Vec3, w float64) Vec4 {
	return Vec4{v.X, v.Y, v.Z, w}
}

// Vec3 returns the Vec3 portion (ignoring W).
func (v Vec4) Vec3() Vec3 {
	return Vec3{v.X, v.Y, v.Z}
}

// PerspectiveDivide returns Vec3 after dividing by W.
func (v Vec4) PerspectiveDivide() Vec3 {
	if v.W == 0 {
		return Vec3{v.X, v.Y, v.Z}
	}
	return Vec3{v.X / v.W, v.Y / v.W, v.Z / v.W}
}

// Add returns the vector sum.
//

func (v Vec4) Add(b Vec4) Vec4 {
	return Vec4{v.X + b.X, v.Y + b.Y, v.Z + b.Z, v.W + b.W}
}

// Sub returns the vector difference.
//

func (v Vec4) Sub(b Vec4) Vec4 {
	return Vec4{v.X - b.X, v.Y - b.Y, v.Z - b.Z, v.W - b.W}
}

// Scale returns the scalar product.
func (v Vec4) Scale(s float64) Vec4 {
	return Vec4{v.X * s, v.Y * s, v.Z * s, v.W * s}
}

// Dot returns the dot product.
//

func (v Vec4) Dot(b Vec4) float64 {
	return v.X*b.X + v.Y*b.Y + v.Z*b.Z + v.W*b.W
}

// Len returns the length.
func (v Vec4) Len() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z + v.W*v.W)
}

// Normalize returns the unit vector.
func (v Vec4) Normalize() Vec4 {
	l := v.Len()
	if l == 0 {
		return Vec4{}
	}
	return Vec4{v.X / l, v.Y / l, v.Z / l, v.W / l}
}

// Lerp returns linear interpolation.
//

func (v Vec4) Lerp(b Vec4, t float64) Vec4 {
	return Vec4{
		v.X + (b.X-v.X)*t,
		v.Y + (b.Y-v.Y)*t,
		v.Z + (b.Z-v.Z)*t,
		v.W + (b.W-v.W)*t,
	}
}
