package geom

import (
	"math"

	"astroswarm/worker/internal/godot"
)

func PointInPolygon(polygon []godot.Vec, point godot.Vec) bool {
	inside := false
	previous := len(polygon) - 1
	for current := range polygon {
		a := polygon[current]
		b := polygon[previous]
		if (a.Y > point.Y) != (b.Y > point.Y) {
			crossing := a.X + (point.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X)
			if point.X < crossing {
				inside = !inside
			}
		}
		previous = current
	}
	return inside
}

func SegmentDistance(start, end, point godot.Vec) float64 {
	segment := end.Sub(start)
	lengthSquared := segment.LengthSquared()
	if lengthSquared <= 0.0 {
		return point.DistanceTo(start)
	}
	position := point.Sub(start).Dot(segment) / lengthSquared
	position = math.Max(0.0, math.Min(1.0, position))
	return point.DistanceTo(start.Add(segment.Scale(position)))
}

func PolygonCircleOverlap(polygon []godot.Vec, center godot.Vec, radius float64) bool {
	if len(polygon) < 3 {
		return false
	}
	if PointInPolygon(polygon, center) {
		return true
	}
	previous := len(polygon) - 1
	for current := range polygon {
		if SegmentDistance(polygon[current], polygon[previous], center) <= radius {
			return true
		}
		previous = current
	}
	return false
}
