package bench

import (
	"math"

	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
	"astroswarm/worker/internal/sim"
)

// Levels 3 to 5 all pit one scattered line against a stream of evaders. Waves
// send them one at a time for as long as the line lasts; a siege sends the
// whole group at once off the arena edges.
const (
	AssaultModeWaves = "waves"
	AssaultModeSiege = "siege"

	WaveGapFrames      = 90      // frames between one evader resolving and the next wave launching
	WaveSpawnSeedShift = 4400000 // rng seed offset for the spawn bearings of a run
	WaveConsecutiveMax = 3       // count of consecutive clean defender counts that ends the sweep

	SiegeEvaders     = 5    // count, evaders that arrive together in a siege
	SiegeEdgeInset   = 60.0 // pixels the siege spawn band sits inside the arena border
	SiegeAngleJitter = 0.35 // radians a siege spawn bearing may wander inside its slice

	AssaultTailFrames   = 180 // frames recorded after the last evader of a siege is resolved
	AssaultRecordStride = 3   // physics frames per recorded frame, so a long run stays replayable
)

type AssaultInput struct {
	Algorithm         []blocks.Script
	Placements        []Placement
	Mode              string
	DestroysDefenders bool
	Seed              int64
	MatchFrames       int
	SinglePrecision   bool
	Record            bool
}

// One launched evader, tagged with how many defenders were still standing when
// it left the ring. That is what turns a run into a risk-against-line-size
// curve once every trial is pooled.
type AttritionSample struct {
	Defenders int // line size when this evader launched
	Remaining int // line size once it was resolved, after any trade
	Destroyed bool
}

type AssaultOutput struct {
	Outcome       string
	Mode          string
	Defenders     int
	Launched      int
	Resolved      int
	Destroyed     int
	Breaches      int
	DefendersLost int
	DetectionTime float64
	CaptureTime   float64
	GoalTime      float64
	EndTime       float64
	Samples       []AttritionSample
	Frames        [][]int
	ViewDistance  float64
	FovDegrees    float64
	Speed         float64
	HullRadius    float64
}

type assaultDefender struct {
	ship  *sim.Ship
	alive bool
}

type assaultEvader struct {
	ship     *sim.Ship
	slot     int
	breached bool
	atLaunch int
}

func RunAssaultMatch(input AssaultInput) AssaultOutput {
	config := blocks.ConfigFromScripts(input.Algorithm, blocks.ShipConfig{
		ViewDistance: sim.DefaultViewDistance,
		FovDegrees:   sim.DefaultFovDegrees,
		Speed:        sim.DefaultSpeed,
		TurnSpeed:    sim.DefaultTurnRate,
		HullRadius:   sim.DefaultHullRadius,
	})

	siege := input.Mode == AssaultModeSiege
	world := sim.NewWorld(godot.NewRNGFromInt(input.Seed), input.SinglePrecision)
	spawnRNG := godot.NewRNGFromInt(input.Seed + WaveSpawnSeedShift)

	output := AssaultOutput{
		Mode:          input.Mode,
		Defenders:     len(input.Placements),
		DetectionTime: -1.0,
		CaptureTime:   -1.0,
		GoalTime:      -1.0,
		EndTime:       -1.0,
		ViewDistance:  config.ViewDistance,
		FovDegrees:    config.FovDegrees,
		Speed:         config.Speed,
		HullRadius:    config.HullRadius,
	}

	defenders := make([]*assaultDefender, 0, len(input.Placements))
	live := make([]*sim.Ship, 0, len(input.Placements))
	for _, placement := range input.Placements {
		ship := sim.NewShip(sim.TeamDefender, input.Algorithm)
		ship.ArenaSize = ArenaSize
		ship.CollisionsEnabled = false
		ship.ApplyConfig(config)
		ship.Position = placement.Position
		ship.Rotation = placement.Rotation
		world.Add(ship)
		defenders = append(defenders, &assaultDefender{ship: ship, alive: true})
		live = append(live, ship)
	}

	slots := 1
	if siege {
		slots = SiegeEvaders
	}
	inFlight := make([]*assaultEvader, slots)

	launch := func(spawn godot.Vec) {
		slot := freeSlot(inFlight)
		if slot < 0 {
			return
		}
		ship := sim.NewShip(sim.TeamEvader, enemyProgram)
		ship.ArenaSize = ArenaSize
		ship.SpeedMult = EnemySpeed / sim.DefaultSpeed
		ship.CollisionsEnabled = false
		ship.IsEvader = true
		ship.Position = spawn
		ship.Rotation = PlanetCenter.Sub(spawn).Angle()
		world.Add(ship)
		inFlight[slot] = &assaultEvader{ship: ship, slot: slot, atLaunch: len(live)}
		output.Launched++
	}

	if siege {
		base := spawnRNG.Randf() * godot.Tau
		for index := 0; index < SiegeEvaders; index++ {
			slice := godot.Tau * float64(index) / float64(SiegeEvaders)
			angle := base + slice + spawnRNG.RandfRange(-SiegeAngleJitter, SiegeAngleJitter)
			launch(SiegeEdgePoint(angle))
		}
	} else {
		launch(waveSpawn(spawnRNG))
	}

	delta := 1.0 / float64(PhysicsTicksPerSecond)
	gap := WaveGapFrames
	endFrame := -1
	frame := 0

	for frame < input.MatchFrames {
		frame++
		if input.Record && frame%AssaultRecordStride == 0 {
			output.Frames = append(output.Frames, assaultSnapshot(defenders, inFlight))
		}

		for _, entry := range inFlight {
			if entry == nil {
				continue
			}
			if output.DetectionTime < 0.0 && anyDefenderSees(live, entry.ship) {
				output.DetectionTime = frameToTime(frame)
			}
			if !entry.breached && entry.ship.Position.DistanceTo(PlanetCenter) <= PlanetRadius+GoalMargin {
				if output.GoalTime < 0.0 {
					output.GoalTime = frameToTime(frame)
				}
				entry.breached = true
				output.Breaches++
				output.Resolved++
				output.Samples = append(output.Samples, AttritionSample{Defenders: entry.atLaunch, Remaining: len(live)})
				world.Remove(entry.ship)
				inFlight[entry.slot] = nil
				continue
			}
			catcher := defenderTouching(live, entry.ship)
			if catcher == nil {
				continue
			}
			if output.CaptureTime < 0.0 {
				output.CaptureTime = frameToTime(frame)
			}
			output.Destroyed++
			output.Resolved++
			world.Remove(entry.ship)
			inFlight[entry.slot] = nil
			if input.DestroysDefenders {
				live = removeShip(live, catcher)
				world.Remove(catcher)
				killDefender(defenders, catcher)
				output.DefendersLost++
			}
			output.Samples = append(output.Samples, AttritionSample{Defenders: entry.atLaunch, Remaining: len(live), Destroyed: true})
		}

		if !siege && len(live) < 1 {
			break
		}

		if siege {
			if allSlotsFree(inFlight) {
				if endFrame < 0 {
					endFrame = frame + AssaultTailFrames
				}
				if frame >= endFrame {
					break
				}
			}
		} else if freeSlot(inFlight) < 0 {
			gap = WaveGapFrames
		} else {
			gap--
			if gap <= 0 {
				gap = WaveGapFrames
				launch(waveSpawn(spawnRNG))
			}
		}

		world.Step(delta)
	}

	output.EndTime = frameToTime(frame)
	output.Outcome = classifyAssault(output, siege)
	return output
}

// A run is clean only if nothing touched the planet. A siege that ran out of
// clock with evaders still inbound decided nothing, so it times out instead.
func classifyAssault(output AssaultOutput, siege bool) string {
	if output.Breaches > 0 {
		return OutcomeLose
	}
	if siege && output.Resolved < output.Launched {
		return OutcomeTimeout
	}
	return OutcomeWin
}

func waveSpawn(rng *godot.RNG) godot.Vec {
	return PlanetCenter.Add(godot.Vec{X: EnemySpawnRadius}.Rotated(rng.Randf() * godot.Tau))
}

// Walks out from the planet on a bearing until it meets the arena border, so a
// siege spawn lands on the edge of the screen rather than on a ring.
func SiegeEdgePoint(angle float64) godot.Vec {
	direction := godot.Vec{X: 1.0}.Rotated(angle)
	halfWidth := ArenaWidth*0.5 - SiegeEdgeInset
	halfHeight := ArenaHeight*0.5 - SiegeEdgeInset
	reach := math.Inf(1)
	if math.Abs(direction.X) > 0.0001 {
		reach = math.Min(reach, halfWidth/math.Abs(direction.X))
	}
	if math.Abs(direction.Y) > 0.0001 {
		reach = math.Min(reach, halfHeight/math.Abs(direction.Y))
	}
	return PlanetCenter.Add(direction.Scale(reach))
}

func allSlotsFree(inFlight []*assaultEvader) bool {
	for _, entry := range inFlight {
		if entry != nil {
			return false
		}
	}
	return true
}

func freeSlot(inFlight []*assaultEvader) int {
	for index, entry := range inFlight {
		if entry == nil {
			return index
		}
	}
	return -1
}

func killDefender(defenders []*assaultDefender, target *sim.Ship) {
	for _, entry := range defenders {
		if entry.ship == target {
			entry.alive = false
			return
		}
	}
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

// Every frame carries the same slots in the same order, dead ships included as
// -1, so the delta packing and the player never have to guess what moved.
func assaultSnapshot(defenders []*assaultDefender, inFlight []*assaultEvader) []int {
	frame := make([]int, 0, (len(defenders)+len(inFlight))*3)
	for _, entry := range defenders {
		if !entry.alive {
			frame = append(frame, -1, -1, 0)
			continue
		}
		frame = append(frame, int(entry.ship.Position.X), int(entry.ship.Position.Y), int(godot.RadToDeg(entry.ship.Rotation)))
	}
	for _, entry := range inFlight {
		if entry == nil {
			frame = append(frame, -1, -1, 0)
			continue
		}
		frame = append(frame, int(entry.ship.Position.X), int(entry.ship.Position.Y), int(godot.RadToDeg(entry.ship.Rotation)))
	}
	return frame
}
