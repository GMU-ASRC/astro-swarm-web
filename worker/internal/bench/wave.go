package bench

import (
	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
	"astroswarm/worker/internal/sim"
)

const (
	WaveSpreadRadians  = 0.7 // radians between two evader spawn angles in a wave
	WaveSequentialGap  = 0.0 // seconds held between one evader dying and the next launching
	WaveConsecutiveMax = 3   // count of consecutive clean defender counts that ends the sweep
)

type WaveInput struct {
	Algorithm         []blocks.Script
	Placements        []Placement
	SpawnAngles       []float64
	Simultaneous      bool
	DestroysDefenders bool
	Seed              int64
	MatchFrames       int
	SinglePrecision   bool
	Record            bool
}

type WaveOutput struct {
	Outcome       string
	EvaderCount   int
	Destroyed     int
	DefendersLost int
	DetectionTime float64
	CaptureTime   float64
	GoalTime      float64
	ClearTime     float64
	Frames        [][]int
	ViewDistance  float64
	FovDegrees    float64
	Speed         float64
	HullRadius    float64
}

type waveEvader struct {
	ship  *sim.Ship
	alive bool
}

func RunWaveMatch(input WaveInput) WaveOutput {
	world := sim.NewWorld(godot.NewRNGFromInt(input.Seed), input.SinglePrecision)

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
		ship.CollisionsEnabled = false
		ship.ApplyConfig(config)
		ship.Position = placement.Position
		ship.Rotation = placement.Rotation
		world.Add(ship)
		defenders = append(defenders, ship)
	}

	output := WaveOutput{
		EvaderCount:   len(input.SpawnAngles),
		DetectionTime: -1.0,
		CaptureTime:   -1.0,
		GoalTime:      -1.0,
		ClearTime:     -1.0,
		ViewDistance:  config.ViewDistance,
		FovDegrees:    config.FovDegrees,
		Speed:         config.Speed,
		HullRadius:    config.HullRadius,
	}

	evaders := make([]*waveEvader, 0, len(input.SpawnAngles))
	launched := 0
	launch := func() {
		if launched >= len(input.SpawnAngles) {
			return
		}
		spawn := PlanetCenter.Add(godot.Vec{X: EnemySpawnRadius}.Rotated(input.SpawnAngles[launched]))
		ship := sim.NewShip(sim.TeamEvader, enemyProgram)
		ship.ArenaSize = ArenaSize
		ship.SpeedMult = EnemySpeed / sim.DefaultSpeed
		ship.CollisionsEnabled = false
		ship.IsEvader = true
		ship.Position = spawn
		ship.Rotation = PlanetCenter.Sub(spawn).Angle()
		world.Add(ship)
		evaders = append(evaders, &waveEvader{ship: ship, alive: true})
		launched++
	}

	if input.Simultaneous {
		for range input.SpawnAngles {
			launch()
		}
	} else {
		launch()
	}

	delta := 1.0 / float64(PhysicsTicksPerSecond)
	frame := 0
	destroyed := 0
	breached := false

	if input.Record {
		output.Frames = append(output.Frames, waveSnapshot(defenders, evaders))
	}

	for {
		frame++
		if input.Record {
			output.Frames = append(output.Frames, waveSnapshot(defenders, evaders))
		}

		for _, entry := range evaders {
			if !entry.alive {
				continue
			}
			if output.DetectionTime < 0.0 && anyDefenderSees(defenders, entry.ship) {
				output.DetectionTime = frameToTime(frame)
			}
			if entry.ship.Position.DistanceTo(PlanetCenter) <= PlanetRadius+GoalMargin {
				output.GoalTime = frameToTime(frame)
				breached = true
				break
			}
			catcher := defenderTouching(defenders, entry.ship)
			if catcher == nil {
				continue
			}
			entry.alive = false
			world.Remove(entry.ship)
			destroyed++
			if output.CaptureTime < 0.0 {
				output.CaptureTime = frameToTime(frame)
			}
			if input.DestroysDefenders {
				defenders = removeShip(defenders, catcher)
				world.Remove(catcher)
				output.DefendersLost++
			}
		}

		if breached {
			break
		}

		if destroyed >= len(input.SpawnAngles) {
			output.ClearTime = frameToTime(frame)
			break
		}
		if !input.Simultaneous && launched < len(input.SpawnAngles) && !anyAlive(evaders) {
			launch()
		}
		if len(defenders) == 0 {
			break
		}
		if frame >= input.MatchFrames {
			break
		}

		world.Step(delta)
	}

	output.Destroyed = destroyed
	switch {
	case destroyed >= len(input.SpawnAngles) && !breached:
		output.Outcome = OutcomeWin
	case breached:
		output.Outcome = OutcomeLose
	default:
		output.Outcome = OutcomeTimeout
	}
	return output
}

func WaveSpawnAngles(seed int64, trial int, count int) []float64 {
	rng := godot.NewRNGFromInt(seed + int64(trial)*SweepSeedStride)
	base := rng.Randf() * godot.Tau
	angles := make([]float64, 0, count)
	for index := 0; index < count; index++ {
		angles = append(angles, base+WaveSpreadRadians*float64(index)+rng.RandfRange(-0.15, 0.15))
	}
	return angles
}

func WaveEvaderCount(trial int, defenders int) int {
	if defenders < 1 {
		return 1
	}
	return 1 + trial%defenders
}

func defenderTouching(defenders []*sim.Ship, evader *sim.Ship) *sim.Ship {
	for _, defender := range defenders {
		if defender.Touches(evader) {
			return defender
		}
	}
	return nil
}

func removeShip(ships []*sim.Ship, target *sim.Ship) []*sim.Ship {
	out := ships[:0]
	for _, ship := range ships {
		if ship != target {
			out = append(out, ship)
		}
	}
	return out
}

func anyAlive(evaders []*waveEvader) bool {
	for _, entry := range evaders {
		if entry.alive {
			return true
		}
	}
	return false
}

func waveSnapshot(defenders []*sim.Ship, evaders []*waveEvader) []int {
	frame := make([]int, 0, (len(defenders)+len(evaders))*3)
	for _, defender := range defenders {
		frame = append(frame, int(defender.Position.X), int(defender.Position.Y), int(godot.RadToDeg(defender.Rotation)))
	}
	for _, entry := range evaders {
		if !entry.alive {
			frame = append(frame, -1, -1, 0)
			continue
		}
		frame = append(frame, int(entry.ship.Position.X), int(entry.ship.Position.Y), int(godot.RadToDeg(entry.ship.Rotation)))
	}
	return frame
}
