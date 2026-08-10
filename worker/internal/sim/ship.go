package sim

import (
	"math"

	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/geom"
	"astroswarm/worker/internal/godot"
)

const (
	TeamDefender = 0
	TeamEvader   = 1

	DefaultSpeed        = 150.0
	DefaultTurnRate     = 3.2
	DefaultViewDistance = 300.0
	DefaultFovDegrees   = 70.0
	DefaultHullRadius   = 9.0

	VisionHullRadius   = 9.0
	VisionConeSegments = 16
	StepTime           = 0.4
	ArenaEdgeMargin    = 14.0
	WallMargin         = 24.0
)

type Ship struct {
	Position godot.Vec
	Rotation float64
	Team     int

	ViewDistance float64
	FovDegrees   float64
	MaxSpeed     float64
	TurnRate     float64
	HullRadius   float64
	SpeedMult    float64

	CollisionsEnabled bool
	IsEvader          bool

	ArenaSize    godot.Vec
	StarCenter   godot.Vec
	StarRadius   float64
	PlanetCenter godot.Vec
	PlanetRadius float64

	forwardInput     float64
	turnInput        float64
	turnCommand      float64
	throttleMult     float64
	visionMaxDist    float64
	visionHalfAngle  float64
	visionInnerRange float64

	cone    []godot.Vec
	visible []*Ship
	enemies []*Ship
	allies  []*Ship

	executor *blocks.Executor
	world    *World
}

func NewShip(team int, program []blocks.Script) *Ship {
	ship := &Ship{
		Team:         team,
		ViewDistance: DefaultViewDistance,
		FovDegrees:   DefaultFovDegrees,
		MaxSpeed:     DefaultSpeed,
		TurnRate:     DefaultTurnRate,
		HullRadius:   DefaultHullRadius,
		SpeedMult:    1.0,
		throttleMult: 1.0,
	}
	ship.executor = blocks.NewExecutor(ship, program)
	ship.RefreshCone()
	return ship
}

func (s *Ship) ApplyConfig(config blocks.ShipConfig) {
	s.ViewDistance = config.ViewDistance
	s.FovDegrees = config.FovDegrees
	s.MaxSpeed = config.Speed
	s.TurnRate = config.TurnSpeed
	s.HullRadius = config.HullRadius
	s.RefreshCone()
}

func (s *Ship) RefreshCone() {
	fov := godot.DegToRad(s.FovDegrees)
	points := make([]godot.Vec, 0, VisionConeSegments+2)
	points = append(points, godot.Vec{})
	start := -fov / 2.0
	for index := 0; index <= VisionConeSegments; index++ {
		angle := start + fov*float64(index)/float64(VisionConeSegments)
		points = append(points, godot.FromAngle(angle).Scale(s.ViewDistance))
	}
	s.cone = points
	s.visionMaxDist = s.ViewDistance + VisionHullRadius
	s.visionHalfAngle = fov / 2.0
	s.visionInnerRange = s.ViewDistance * math.Cos(fov/(2.0*VisionConeSegments))
}

func (s *Ship) updateVisible() {
	s.visible = s.visible[:0]
	sin, cos := math.Sincos(-s.Rotation)
	for _, other := range s.world.Ships {
		if other == s {
			continue
		}
		offset := other.Position.Sub(s.Position)
		if offset.LengthSquared() > s.visionMaxDist*s.visionMaxDist {
			continue
		}
		local := godot.Vec{X: offset.X*cos - offset.Y*sin, Y: offset.X*sin + offset.Y*cos}
		if s.coneSees(local) {
			s.visible = append(s.visible, other)
		}
	}

	s.enemies = s.enemies[:0]
	s.allies = s.allies[:0]
	for _, other := range s.visible {
		if other.Team != s.Team {
			s.enemies = append(s.enemies, other)
		} else {
			s.allies = append(s.allies, other)
		}
	}
}

func (s *Ship) coneSees(local godot.Vec) bool {
	distanceSquared := local.LengthSquared()
	if distanceSquared <= VisionHullRadius*VisionHullRadius {
		return true
	}
	distance := math.Sqrt(distanceSquared)
	angle := math.Abs(math.Atan2(local.Y, local.X))
	slack := math.Asin(math.Min(1.0, VisionHullRadius/distance))
	if angle > s.visionHalfAngle+slack {
		return false
	}
	if angle+slack <= s.visionHalfAngle && distance+VisionHullRadius <= s.visionInnerRange {
		return true
	}
	return geom.PolygonCircleOverlap(s.cone, local, VisionHullRadius)
}

func (s *Ship) step(delta float64) {
	s.executor.Process(delta)
	s.applyMovement(delta)
}

func (s *Ship) ResetInputs() {
	s.forwardInput = 0.0
	s.turnInput = 0.0
}

func (s *Ship) OnDeactivate() {
	s.turnCommand = 0.0
}

func (s *Ship) applyMovement(delta float64) {
	s.Rotation += (s.turnInput*s.TurnRate + s.turnCommand) * delta
	s.narrowRotation()

	velocity := godot.FromAngle(s.Rotation).Scale(s.MaxSpeed * s.SpeedMult * s.forwardInput)
	s.Position = s.Position.Add(velocity.Scale(delta))
	s.Position.X = godot.Clamp(s.Position.X, ArenaEdgeMargin, s.ArenaSize.X-ArenaEdgeMargin)
	s.Position.Y = godot.Clamp(s.Position.Y, ArenaEdgeMargin, s.ArenaSize.Y-ArenaEdgeMargin)
	s.narrowPosition()

	s.resolveObstacles()
}

func (s *Ship) resolveObstacles() {
	if !s.CollisionsEnabled || s.IsEvader {
		return
	}
	if s.PlanetRadius > 0.0 {
		minimum := s.PlanetRadius + s.HullRadius
		offset := s.Position.Sub(s.PlanetCenter)
		distance := offset.Length()
		if distance < minimum {
			if distance < 0.001 {
				offset = godot.Vec{X: 1.0}
				distance = 1.0
			}
			s.Position = s.PlanetCenter.Add(offset.Scale(minimum / distance))
			s.narrowPosition()
		}
	}
	s.resolveShipCollisions()
}

func (s *Ship) resolveShipCollisions() {
	for _, other := range s.world.Ships {
		if other == s || other.IsEvader {
			continue
		}
		minimum := s.HullRadius + other.HullRadius
		offset := s.Position.Sub(other.Position)
		distance := offset.Length()
		if distance < minimum && distance > 0.001 {
			s.Position = other.Position.Add(offset.Scale(minimum / distance))
			s.narrowPosition()
		}
	}
}

func (s *Ship) narrowPosition() {
	if s.world != nil && s.world.SinglePrecision {
		s.Position = s.Position.Narrowed()
	}
}

func (s *Ship) narrowRotation() {
	if s.world != nil && s.world.SinglePrecision {
		s.Rotation = godot.Narrow(s.Rotation)
	}
}

func (s *Ship) SeesPoint(target godot.Vec) bool {
	offset := target.Sub(s.Position)
	if offset.Length() > s.ViewDistance {
		return false
	}
	limit := godot.DegToRad(s.FovDegrees * 0.5)
	difference := godot.AngleDifference(s.Rotation, offset.Angle())
	if difference < 0 {
		difference = -difference
	}
	return difference <= limit
}

func (s *Ship) Touches(other *Ship) bool {
	return s.Position.DistanceTo(other.Position) <= s.HullRadius+other.HullRadius
}
