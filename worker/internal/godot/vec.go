package godot

import "math"

type Vec struct {
	X float64
	Y float64
}

func FromAngle(angle float64) Vec {
	sin, cos := math.Sincos(angle)
	return Vec{X: cos, Y: sin}
}

func (v Vec) Add(other Vec) Vec {
	return Vec{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vec) Sub(other Vec) Vec {
	return Vec{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vec) Scale(factor float64) Vec {
	return Vec{X: v.X * factor, Y: v.Y * factor}
}

func (v Vec) Dot(other Vec) float64 {
	return v.X*other.X + v.Y*other.Y
}

func (v Vec) LengthSquared() float64 {
	return v.X*v.X + v.Y*v.Y
}

func (v Vec) Length() float64 {
	return math.Hypot(v.X, v.Y)
}

func (v Vec) DistanceTo(other Vec) float64 {
	return other.Sub(v).Length()
}

func (v Vec) DistanceSquaredTo(other Vec) float64 {
	return other.Sub(v).LengthSquared()
}

func (v Vec) Angle() float64 {
	return math.Atan2(v.Y, v.X)
}

func (v Vec) AngleToPoint(other Vec) float64 {
	return other.Sub(v).Angle()
}

func (v Vec) Rotated(angle float64) Vec {
	sin, cos := math.Sincos(angle)
	return Vec{X: v.X*cos - v.Y*sin, Y: v.X*sin + v.Y*cos}
}

func (v Vec) Narrowed() Vec {
	return Vec{X: Narrow(v.X), Y: Narrow(v.Y)}
}
