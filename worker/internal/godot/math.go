package godot

import "math"

const Tau = 2 * math.Pi

func DegToRad(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

func RadToDeg(radians float64) float64 {
	return radians * 180.0 / math.Pi
}

func AngleDifference(from, to float64) float64 {
	difference := math.Mod(to-from, Tau)
	return math.Mod(2.0*difference, Tau) - difference
}

func LerpAngle(from, to, weight float64) float64 {
	return from + AngleDifference(from, to)*weight
}

func Snapped(value, step float64) float64 {
	if step == 0.0 {
		return value
	}
	return math.Floor(value/step+0.5) * step
}

func Clamp(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func Narrow(value float64) float64 {
	return float64(float32(value))
}
