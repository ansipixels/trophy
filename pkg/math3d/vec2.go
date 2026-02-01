package math3d

import "math"

// Vec2 represents a 2D vector.
type Vec2 struct {
	X, Y float64
}

// V2 creates a new Vec2.
func V2(x, y float64) Vec2 {
	return Vec2{x, y}
}

// Zero2 returns the zero vector.
func Zero2() Vec2 {
	return Vec2{}
}

// Add returns the vector sum a + b.
func (a Vec2) Add(b Vec2) Vec2 {
	return Vec2{a.X + b.X, a.Y + b.Y}
}

// Sub returns the vector difference a - b.
func (a Vec2) Sub(b Vec2) Vec2 {
	return Vec2{a.X - b.X, a.Y - b.Y}
}

// Scale returns the scalar product a * s.
func (a Vec2) Scale(s float64) Vec2 {
	return Vec2{a.X * s, a.Y * s}
}

// Mul returns the component-wise product a * b.
func (a Vec2) Mul(b Vec2) Vec2 {
	return Vec2{a.X * b.X, a.Y * b.Y}
}

// Dot returns the dot product a · b.
func (a Vec2) Dot(b Vec2) float64 {
	return a.X*b.X + a.Y*b.Y
}

// Len returns the length of the vector.
func (a Vec2) Len() float64 {
	return math.Sqrt(a.X*a.X + a.Y*a.Y)
}

// LenSq returns the squared length (faster, no sqrt).
func (a Vec2) LenSq() float64 {
	return a.X*a.X + a.Y*a.Y
}

// Normalize returns the unit vector.
func (a Vec2) Normalize() Vec2 {
	l := a.Len()
	if l == 0 {
		return Vec2{}
	}
	return Vec2{a.X / l, a.Y / l}
}

// Negate returns the negated vector.
func (a Vec2) Negate() Vec2 {
	return Vec2{-a.X, -a.Y}
}

// Lerp returns linear interpolation between a and b.
func (a Vec2) Lerp(b Vec2, t float64) Vec2 {
	return Vec2{
		a.X + (b.X-a.X)*t,
		a.Y + (b.Y-a.Y)*t,
	}
}

// Rotate rotates the vector by angle (radians).
func (a Vec2) Rotate(angle float64) Vec2 {
	cos, sin := math.Cos(angle), math.Sin(angle)
	return Vec2{
		a.X*cos - a.Y*sin,
		a.X*sin + a.Y*cos,
	}
}

// Perpendicular returns a perpendicular vector (90° counter-clockwise).
func (a Vec2) Perpendicular() Vec2 {
	return Vec2{-a.Y, a.X}
}

// Angle returns the angle of the vector in radians.
func (a Vec2) Angle() float64 {
	return math.Atan2(a.Y, a.X)
}

// AngleTo returns the angle between two vectors.
func (a Vec2) AngleTo(b Vec2) float64 {
	return math.Atan2(b.Y-a.Y, b.X-a.X)
}

// Distance returns the distance between two points.
func (a Vec2) Distance(b Vec2) float64 {
	return a.Sub(b).Len()
}
