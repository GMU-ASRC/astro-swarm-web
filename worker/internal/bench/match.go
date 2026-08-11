package bench

import (
	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
	"astroswarm/worker/internal/sim"
)

var enemyProgram = []blocks.Script{{
	Blocks: []blocks.Block{{
		Type:   "when_always",
		Params: map[string]any{},
		Children: []blocks.Block{
			{Type: "do_forward", Params: map[string]any{}},
		},
	}},
}}

type MatchInput struct {
	Algorithm       []blocks.Script
	Placements      []Placement
	EnemyStart      godot.Vec
	Seed            int64
	MatchFrames     int
	GoalTailFrames  int
	Collisions      bool
	SinglePrecision bool
	Record          bool
}

type MatchOutput struct {
	Outcome       string
	DetectionTime float64
	CaptureTime   float64
	GoalTime      float64
	Frames        [][]int
	ViewDistance  float64
	FovDegrees    float64
	Speed         float64
	HullRadius    float64
}

func RunMatch(input MatchInput) MatchOutput {
	world := sim.NewWorld(godot.NewRNGFromInt(input.Seed), input.SinglePrecision)

	planetCenter := godot.Vec{}
	planetRadius := 0.0
	if input.Collisions {
		planetCenter = PlanetCenter
		planetRadius = PlanetRadius
	}

	config := blocks.ConfigFromScripts(input.Algorithm, blocks.ShipConfig{
		ViewDistance: sim.DefaultViewDistance,
		FovDegrees:   sim.DefaultFovDegrees,
		Speed:        sim.DefaultSpeed,
		TurnSpeed:    sim.DefaultTurnRate,
		HullRadius:   sim.DefaultHullRadius,
	})

	defenders := make([]*sim.Ship, 0, len(input.Placements))
	for _, placement := range input.Placements {
		ship := sim.NewShip(sim.TeamDefender, input.Algorithm)
		ship.ArenaSize = ArenaSize
		ship.PlanetCenter = planetCenter
		ship.PlanetRadius = planetRadius
		ship.CollisionsEnabled = input.Collisions
		ship.ApplyConfig(config)
		ship.Position = placement.Position
		ship.Rotation = placement.Rotation
		world.Add(ship)
		defenders = append(defenders, ship)
	}

	evader := sim.NewShip(sim.TeamEvader, enemyProgram)
	evader.ArenaSize = ArenaSize
	evader.SpeedMult = EnemySpeed / sim.DefaultSpeed
	evader.CollisionsEnabled = false
	evader.IsEvader = true
	evader.Position = input.EnemyStart
	evader.Rotation = PlanetCenter.Sub(input.EnemyStart).Angle()
	world.Add(evader)

	output := MatchOutput{
		ViewDistance: config.ViewDistance,
		FovDegrees:   config.FovDegrees,
		Speed:        config.Speed,
		HullRadius:   config.HullRadius,
	}

	delta := 1.0 / float64(PhysicsTicksPerSecond)
	frame := 0
	detectFrame := -1
	captureFrame := -1
	goalFrame := -1
	endFrame := -1

	if input.Record {
		output.Frames = append(output.Frames, snapshot(defenders, evader))
	}

	for {
		frame++
		if input.Record && (endFrame < 0 || frame <= endFrame) {
			output.Frames = append(output.Frames, snapshot(defenders, evader))
		}

		if detectFrame < 0 && anyDefenderSees(defenders, evader) {
			detectFrame = frame
		}
		if captureFrame < 0 && anyDefenderTouches(defenders, evader) {
			captureFrame = frame
		}
		if goalFrame < 0 && evaderAtGoal(evader) {
			goalFrame = frame
			endFrame = frame + input.GoalTailFrames
		}

		finished := (endFrame >= 0 && frame >= endFrame) ||
			evaderReachedFarEdge(evader, frame) ||
			frame >= input.MatchFrames
		if finished {
			break
		}

		world.Step(delta)
	}

	output.Outcome = classify(captureFrame, goalFrame)
	output.DetectionTime = frameToTime(detectFrame)
	output.CaptureTime = frameToTime(captureFrame)
	output.GoalTime = frameToTime(goalFrame)
	return output
}

func classify(captureFrame, goalFrame int) string {
	capturedBeforeGoal := captureFrame >= 0 && (goalFrame < 0 || captureFrame <= goalFrame)
	if capturedBeforeGoal {
		return OutcomeWin
	}
	if goalFrame >= 0 {
		return OutcomeLose
	}
	return OutcomeTimeout
}

func frameToTime(frame int) float64 {
	if frame < 0 {
		return -1.0
	}
	return godot.Snapped(float64(frame)/float64(PhysicsTicksPerSecond), 0.01)
}

func anyDefenderSees(defenders []*sim.Ship, evader *sim.Ship) bool {
	for _, defender := range defenders {
		if defender.SeesPoint(evader.Position) {
			return true
		}
	}
	return false
}

func anyDefenderTouches(defenders []*sim.Ship, evader *sim.Ship) bool {
	for _, defender := range defenders {
		if defender.Touches(evader) {
			return true
		}
	}
	return false
}

func evaderAtGoal(evader *sim.Ship) bool {
	return evader.Position.DistanceTo(PlanetCenter) <= PlanetRadius+GoalMargin
}

func evaderReachedFarEdge(evader *sim.Ship, frame int) bool {
	if frame < 30 {
		return false
	}
	position := evader.Position
	return position.X <= 16.0 || position.Y <= 16.0 ||
		position.X >= ArenaWidth-16.0 || position.Y >= ArenaHeight-16.0
}

func snapshot(defenders []*sim.Ship, evader *sim.Ship) []int {
	frame := make([]int, 0, len(defenders)*3+3)
	for _, defender := range defenders {
		frame = append(frame, int(defender.Position.X), int(defender.Position.Y), int(godot.RadToDeg(defender.Rotation)))
	}
	frame = append(frame, int(evader.Position.X), int(evader.Position.Y), int(godot.RadToDeg(evader.Rotation)))
	return frame
}
