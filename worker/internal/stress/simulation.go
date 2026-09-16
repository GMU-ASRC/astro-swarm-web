package stress

import (
	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
	"astroswarm/worker/internal/sim"
)

const worldSeedShift = 1

var evaderProgram = []blocks.Script{{
	Blocks: []blocks.Block{{
		Type:   "when_always",
		Params: map[string]any{},
		Children: []blocks.Block{
			{Type: "do_forward", Params: map[string]any{}},
		},
	}},
}}

type ShipView struct {
	X        float64
	Y        float64
	Rotation float64
	Squad    int
}

type Totals struct {
	Launched        int   `json:"launched"`
	Breaches        int   `json:"breaches"`
	EvadersInFlight int   `json:"evaders_in_flight"`
	Captures        []int `json:"captures"`
	DefendersLost   []int `json:"defenders_lost"`
	DefendersAlive  []int `json:"defenders_alive"`
}

type Snapshot struct {
	Tick       int
	SimSeconds float64
	Defenders  []ShipView
	Evaders    []ShipView
	Totals     Totals
}

type defender struct {
	ship  *sim.Ship
	squad int
}

type swarmSimulation struct {
	options   Options
	world     *sim.World
	rng       *godot.RNG
	center    godot.Vec
	arenaSize godot.Vec
	defenders []*defender
	evaders   []*sim.Ship
	totals    Totals
}

func newSimulation(options Options) *swarmSimulation {
	squadCount := len(options.Squads)
	simulation := &swarmSimulation{
		options:   options,
		world:     sim.NewWorld(godot.NewRNGFromInt(options.Seed+worldSeedShift), false),
		rng:       godot.NewRNGFromInt(options.Seed),
		center:    planetCenter(options),
		arenaSize: godot.Vec{X: options.ArenaWidth, Y: options.ArenaHeight},
		totals: Totals{
			Captures:       make([]int, squadCount),
			DefendersLost:  make([]int, squadCount),
			DefendersAlive: make([]int, squadCount),
		},
	}

	configs := make([]blocks.ShipConfig, squadCount)
	for index, squad := range options.Squads {
		configs[index] = SquadConfig(squad)
	}

	for index, spot := range defenderPlacements(options, simulation.rng) {
		squadIndex := index % squadCount
		ship := sim.NewShip(sim.TeamDefender, options.Squads[squadIndex].Algorithm)
		ship.ArenaSize = simulation.arenaSize
		if options.Collisions {
			ship.PlanetCenter = simulation.center
			ship.PlanetRadius = options.PlanetRadius
		}
		ship.CollisionsEnabled = options.Collisions
		ship.Unbounded = !options.KeepInArena
		ship.ApplyConfig(configs[squadIndex])
		ship.Position = spot.position
		ship.Rotation = spot.rotation
		simulation.world.Add(ship)
		simulation.defenders = append(simulation.defenders, &defender{ship: ship, squad: squadIndex})
		simulation.totals.DefendersAlive[squadIndex]++
	}
	return simulation
}

func (s *swarmSimulation) launchWave() {
	for count := 0; count < s.options.EvadersPerWave && len(s.evaders) < s.options.MaxEvaders; count++ {
		spawn := evaderSpawnPoint(s.options, s.rng.Randf()*godot.Tau)
		ship := sim.NewShip(sim.TeamEvader, evaderProgram)
		ship.ArenaSize = s.arenaSize
		ship.SpeedMult = s.options.EvaderSpeed / sim.DefaultSpeed
		ship.IsEvader = true
		ship.Unbounded = !s.options.KeepInArena
		ship.Position = spawn
		ship.Rotation = s.center.Sub(spawn).Angle()
		s.world.Add(ship)
		s.evaders = append(s.evaders, ship)
		s.totals.Launched++
	}
	s.totals.EvadersInFlight = len(s.evaders)
}

func (s *swarmSimulation) resolveContacts() {
	goalDistance := s.options.PlanetRadius + GoalMargin
	remaining := s.evaders[:0]
	for _, evader := range s.evaders {
		if evader.Position.DistanceTo(s.center) <= goalDistance {
			s.totals.Breaches++
			s.world.Remove(evader)
			continue
		}
		catcherIndex := s.catcherOf(evader)
		if catcherIndex < 0 {
			remaining = append(remaining, evader)
			continue
		}
		catcher := s.defenders[catcherIndex]
		s.totals.Captures[catcher.squad]++
		s.world.Remove(evader)
		if s.options.Attrition {
			s.world.Remove(catcher.ship)
			s.defenders = append(s.defenders[:catcherIndex], s.defenders[catcherIndex+1:]...)
			s.totals.DefendersLost[catcher.squad]++
			s.totals.DefendersAlive[catcher.squad]--
		}
	}
	s.evaders = remaining
	s.totals.EvadersInFlight = len(s.evaders)
}

func (s *swarmSimulation) catcherOf(evader *sim.Ship) int {
	for index, candidate := range s.defenders {
		if candidate.ship.Touches(evader) {
			return index
		}
	}
	return -1
}

func (s *swarmSimulation) snapshot(tick int) Snapshot {
	snapshot := Snapshot{
		Tick:       tick,
		SimSeconds: float64(tick) / float64(s.options.TickRate),
		Defenders:  make([]ShipView, 0, len(s.defenders)),
		Evaders:    make([]ShipView, 0, len(s.evaders)),
		Totals:     s.totals.clone(),
	}
	for _, entry := range s.defenders {
		snapshot.Defenders = append(snapshot.Defenders, ShipView{
			X:        entry.ship.Position.X,
			Y:        entry.ship.Position.Y,
			Rotation: entry.ship.Rotation,
			Squad:    entry.squad,
		})
	}
	for _, evader := range s.evaders {
		snapshot.Evaders = append(snapshot.Evaders, ShipView{
			X:        evader.Position.X,
			Y:        evader.Position.Y,
			Rotation: evader.Rotation,
			Squad:    -1,
		})
	}
	return snapshot
}

func (totals Totals) clone() Totals {
	copied := totals
	copied.Captures = append([]int(nil), totals.Captures...)
	copied.DefendersLost = append([]int(nil), totals.DefendersLost...)
	copied.DefendersAlive = append([]int(nil), totals.DefendersAlive...)
	return copied
}

func (totals Totals) AliveDefenders() int {
	return sum(totals.DefendersAlive)
}

func (totals Totals) TotalCaptures() int {
	return sum(totals.Captures)
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
