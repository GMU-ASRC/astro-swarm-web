package sim

import (
	"math"

	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
)

func (s *Ship) EvalCondition(condition string, params map[string]any) bool {
	switch condition {
	case "always":
		return true
	case "sees", "sees_species", "sees_enemy":
		return len(s.enemies) > 0
	case "alone":
		return len(s.enemies) == 0 && len(s.allies) == 0
	case "no_sees_species":
		return len(s.enemies) == 0
	case "sees_ally":
		return len(s.allies) > 0
	case "near_wall":
		return s.nearWall()
	case "sees_wall", "sees_rim":
		return s.seesRim()
	case "sees_object":
		return s.seesObject()
	case "within":
		distance := s.nearestEnemyDistance()
		return distance >= 0.0 && distance <= blocks.ParamFloat(params, "value", 3.0)*blocks.PixelsPerMeter
	case "beyond":
		distance := s.nearestEnemyDistance()
		return distance >= 0.0 && distance > blocks.ParamFloat(params, "value", 3.0)*blocks.PixelsPerMeter
	case "see":
		switch blocks.ParamString(params, "target", "anyone") {
		case "object":
			return s.seesObject()
		case "wall":
			return s.nearWall()
		case "ally":
			return len(s.allies) > 0
		default:
			return len(s.enemies) > 0
		}
	case "compare":
		return compareVariable(params)
	}
	return false
}

func compareVariable(params map[string]any) bool {
	left := 0.0
	right := blocks.ParamFloat(params, "value", 0.0)
	switch blocks.ParamString(params, "op", "=") {
	case "=":
		return left == right
	case "!=":
		return left != right
	case "<":
		return left < right
	case ">":
		return left > right
	case "<=":
		return left <= right
	case ">=":
		return left >= right
	}
	return false
}

func (s *Ship) nearestEnemyDistance() float64 {
	best := -1.0
	for _, enemy := range s.enemies {
		distance := s.Position.DistanceTo(enemy.Position)
		if best < 0.0 || distance < best {
			best = distance
		}
	}
	return best
}

func (s *Ship) nearestEnemy() *Ship {
	var best *Ship
	bestDistance := math.Inf(1)
	for _, enemy := range s.enemies {
		distance := s.Position.DistanceSquaredTo(enemy.Position)
		if distance < bestDistance {
			bestDistance = distance
			best = enemy
		}
	}
	return best
}

func (s *Ship) objectInView(center godot.Vec, radius float64) bool {
	offset := center.Sub(s.Position)
	if offset.Length()-radius > s.ViewDistance {
		return false
	}
	difference := godot.AngleDifference(s.Rotation, offset.Angle())
	if difference < 0 {
		difference = -difference
	}
	return difference <= godot.DegToRad(s.FovDegrees)*0.5
}

func (s *Ship) seesObject() bool {
	if s.StarRadius > 0.0 && s.objectInView(s.StarCenter, s.StarRadius) {
		return true
	}
	if s.PlanetRadius > 0.0 && s.objectInView(s.PlanetCenter, s.PlanetRadius) {
		return true
	}
	return false
}

func (s *Ship) seesRim() bool {
	direction := godot.FromAngle(s.Rotation)
	distance := math.Inf(1)
	if direction.X > 0.0001 {
		distance = math.Min(distance, (s.ArenaSize.X-s.Position.X)/direction.X)
	} else if direction.X < -0.0001 {
		distance = math.Min(distance, -s.Position.X/direction.X)
	}
	if direction.Y > 0.0001 {
		distance = math.Min(distance, (s.ArenaSize.Y-s.Position.Y)/direction.Y)
	} else if direction.Y < -0.0001 {
		distance = math.Min(distance, -s.Position.Y/direction.Y)
	}
	return distance <= s.ViewDistance
}

func (s *Ship) nearWall() bool {
	return s.Position.X <= WallMargin || s.Position.Y <= WallMargin ||
		s.Position.X >= s.ArenaSize.X-WallMargin || s.Position.Y >= s.ArenaSize.Y-WallMargin
}
