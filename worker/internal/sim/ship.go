package sim

import (
	"math"

	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/geom"
	"astroswarm/worker/internal/godot"
)

// World units are pixels and seconds. Angles are radians unless a name ends in
// Degrees. blocks.PixelsPerMeter (40) converts pixels to the meters shown in the
// block editor.
const (
	TeamDefender = 0 // team id
	TeamEvader   = 1 // team id

	DefaultSpeed        = 150.0 // pixels/second (3.75 m/s)
	DefaultTurnRate     = 3.2   // radians/second
	DefaultViewDistance = 300.0 // pixels (7.5 m)
	DefaultFovDegrees   = 70.0  // degrees, full width of the vision cone
	DefaultHullRadius   = 9.0   // pixels (0.225 m)

	VisionHullRadius   = 9.0  // pixels, hull radius assumed for every ship being looked at
	VisionConeSegments = 16   // count, segments approximating the cone arc
	StepTime           = 0.4  // seconds a movement action runs before it reports done
	ArenaEdgeMargin    = 14.0 // pixels, clamp distance from the arena edge
	WallMargin         = 24.0 // pixels, distance from a wall counted as near_wall
)

type Ship struct {
	Position godot.Vec // pixels, arena space with the origin at the top left corner
	Rotation float64   // radians, 0 points along +X and grows clockwise
	Team     int       // TeamDefender or TeamEvader

	ViewDistance float64 // pixels
	FovDegrees   float64 // degrees, full width of the vision cone
	MaxSpeed     float64 // pixels/second at full throttle
	TurnRate     float64 // radians/second
	HullRadius   float64 // pixels, collision radius
	SpeedMult    float64 // multiplier on MaxSpeed, 1.0 is unscaled

	CollisionsEnabled bool
	IsEvader          bool

	ArenaSize    godot.Vec // pixels, arena width and height
	StarCenter   godot.Vec // pixels
	StarRadius   float64   // pixels, 0 means there is no star
	PlanetCenter godot.Vec // pixels
	PlanetRadius float64   // pixels, 0 means there is no planet

	forwardInput     float64 // throttle in -1..1, negative drives backward
	turnInput        float64 // turn in -1..1, scaled by TurnRate
	turnCommand      float64 // radians/second added on top of turnInput
	throttleMult     float64 // multiplier applied to forwardInput, 1.0 is full throttle
	visionMaxDist    float64 // pixels, ViewDistance plus VisionHullRadius
	visionHalfAngle  float64 // radians, half of FovDegrees
	visionInnerRange float64 // pixels, radius fully inside the segmented cone

	cone    []godot.Vec // pixels, cone outline in ship local space
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
