package stress

import (
	"fmt"
	"image/color"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/sim"
)

const (
	SpawnArea  = "area"
	SpawnArena = "arena"
	SpawnRing  = "ring"

	EvaderSpawnRing = "ring"
	EvaderSpawnEdge = "edge"

	DefaultShips            = 1000
	DefaultArenaWidth       = bench.ArenaWidth
	DefaultArenaHeight      = bench.ArenaHeight
	DefaultPlanetRadius     = bench.PlanetRadius
	DefaultSpawnInner       = bench.PlaceMin
	DefaultSpawnOuter       = bench.ScatterMax
	DefaultSpawnSpacing     = bench.ScatterSpacing
	DefaultRingRadius       = bench.SweepRadius
	DefaultEvaderRingRadius = bench.EnemySpawnRadius
	DefaultDurationSeconds  = 120.0
	DefaultTickRate         = bench.PhysicsTicksPerSecond
	DefaultSeed             = bench.DefaultSeed
	DefaultWaveInterval     = 2.0
	DefaultEvadersPerWave   = 10
	DefaultMaxEvaders       = 100
	DefaultEvaderSpeed      = bench.EnemySpeed
	DefaultSampleSeconds    = 0.5
	DefaultVideoFPS         = 30

	GoalMargin     = bench.GoalMargin
	EdgeSpawnInset = bench.SiegeEdgeInset
	EdgeMargin     = sim.ArenaEdgeMargin
	SpawnAttempts  = bench.ScatterAttempts
)

type Squad struct {
	Name      string
	Algorithm []blocks.Script
	Color     color.RGBA
}

type Options struct {
	Squads           []Squad
	Ships            int
	ArenaWidth       float64
	ArenaHeight      float64
	PlanetRadius     float64
	Spawn            string
	SpawnInner       float64
	SpawnOuter       float64
	SpawnSpacing     float64
	RingRadius       float64
	DurationSeconds  float64
	TickRate         int
	Seed             int64
	WaveInterval     float64
	EvadersPerWave   int
	MaxEvaders       int
	EvaderSpeed      float64
	EvaderSpawn      string
	EvaderRingRadius float64
	Attrition        bool
	Collisions       bool
	KeepInArena      bool
	SampleSeconds    float64
	VideoFPS         int
}

func (options Options) Validate() error {
	switch {
	case len(options.Squads) == 0:
		return fmt.Errorf("at least one squad is required")
	case options.Ships < 1:
		return fmt.Errorf("ships must be at least 1")
	case options.ArenaWidth < 200 || options.ArenaHeight < 200:
		return fmt.Errorf("arena must be at least 200 x 200 pixels")
	case options.PlanetRadius <= 0:
		return fmt.Errorf("planet radius must be positive")
	case options.PlanetRadius*2 >= options.ArenaWidth || options.PlanetRadius*2 >= options.ArenaHeight:
		return fmt.Errorf("planet radius %.0f does not fit in a %.0f x %.0f arena", options.PlanetRadius, options.ArenaWidth, options.ArenaHeight)
	case options.Spawn != SpawnArea && options.Spawn != SpawnArena && options.Spawn != SpawnRing:
		return fmt.Errorf("spawn must be %q, %q or %q", SpawnArea, SpawnArena, SpawnRing)
	case options.SpawnInner < 0 || options.SpawnSpacing < 0:
		return fmt.Errorf("spawn inner radius and spacing cannot be negative")
	case options.Spawn == SpawnArea && options.SpawnOuter <= options.SpawnInner:
		return fmt.Errorf("spawn outer radius (%.0f) must be larger than the inner radius (%.0f)", options.SpawnOuter, options.SpawnInner)
	case options.Spawn == SpawnRing && options.RingRadius <= 0:
		return fmt.Errorf("ring radius must be positive")
	case options.DurationSeconds <= 0:
		return fmt.Errorf("seconds must be positive")
	case options.TickRate < 1:
		return fmt.Errorf("tick rate must be at least 1")
	case options.WaveInterval <= 0:
		return fmt.Errorf("wave interval must be positive")
	case options.EvadersPerWave < 0 || options.MaxEvaders < 0:
		return fmt.Errorf("evader counts cannot be negative")
	case options.EvaderSpeed <= 0:
		return fmt.Errorf("evader speed must be positive")
	case options.EvaderSpawn != EvaderSpawnRing && options.EvaderSpawn != EvaderSpawnEdge:
		return fmt.Errorf("evader spawn must be %q or %q", EvaderSpawnRing, EvaderSpawnEdge)
	case options.EvaderSpawn == EvaderSpawnRing && options.EvaderRingRadius <= options.PlanetRadius:
		return fmt.Errorf("evader ring radius must be larger than the planet radius")
	case options.SampleSeconds <= 0:
		return fmt.Errorf("sample seconds must be positive")
	case options.VideoFPS < 0 || options.VideoFPS > options.TickRate:
		return fmt.Errorf("video fps must be between 0 and the tick rate (%d)", options.TickRate)
	}
	return nil
}

func SquadConfig(squad Squad) blocks.ShipConfig {
	return blocks.ConfigFromScripts(squad.Algorithm, blocks.ShipConfig{
		ViewDistance: sim.DefaultViewDistance,
		FovDegrees:   sim.DefaultFovDegrees,
		Speed:        sim.DefaultSpeed,
		TurnSpeed:    sim.DefaultTurnRate,
		HullRadius:   sim.DefaultHullRadius,
	})
}
