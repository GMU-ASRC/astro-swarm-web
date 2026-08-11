package bench

import "astroswarm/worker/internal/godot"

// Distances are pixels, durations are seconds and angles are radians, matching
// the sim package.
const (
	PhysicsTicksPerSecond = 60 // ticks/second the match is stepped at

	ArenaWidth   = 3840.0 // pixels
	ArenaHeight  = 2160.0 // pixels
	PlanetX      = 1920.0 // pixels
	PlanetY      = 1080.0 // pixels
	PlanetRadius = 120.0  // pixels
	GoalMargin   = 16.0   // pixels past PlanetRadius that still counts as reaching the planet

	MaxDefenders = 6     // count, largest hand placed defender layout
	EnemySpeed   = 105.0 // pixels/second (2.625 m/s)

	DefaultTrials          = 100       // count, trials per benchmark run
	DefaultSweepMax        = 100       // count, largest defender count in the sweep
	DefaultSweepTrials     = 100       // count, trials per defender count in the sweep
	DefaultSeed            = 987654321 // rng seed
	DefaultMatchSeconds    = 240.0     // seconds before a match times out
	DefaultGoalTailSeconds = 3.0       // seconds simulated after the evader reaches the planet

	SweepRadius      = 300.0               // pixels, radius of the defender ring around the planet
	PlaceMin         = PlanetRadius + 50.0 // pixels, closest a scattered defender may spawn to the planet center
	ScatterMax       = 250.0               // pixels, farthest a scattered defender may spawn from the planet center
	ScatterSpacing   = 110.0               // pixels, minimum gap between two scattered defenders
	ScatterAttempts  = 40                  // count, retries before a scattered position is accepted anyway
	RingCount        = 5                   // count, defenders in the default level 2 layout
	RingSectorInset  = 0.2                 // fraction of a ring sector kept clear at each end when a defender is jittered
	EnemySpawnRadius = 1000.0              // pixels, radius of the ring the evader spawns on

	ReplaySweepNMax   = 50 // count, largest defender count that still records a replay
	ReplaySweepTrials = 10 // count, trials per defender count that still record a replay

	SweepSeedOffset      = 100000  // rng seed offset
	SweepSpawnSeedOffset = 300000  // rng seed offset
	SweepSeedStride      = 1000000 // rng seed step between sweep trials
	SweepMatchOffset     = 500000  // rng seed offset
	PlacementSeedOffset  = 700000  // rng seed offset

	OutcomeWin     = "win"
	OutcomeLose    = "lose"
	OutcomeTimeout = "timeout"
)

var (
	ArenaSize    = godot.Vec{X: ArenaWidth, Y: ArenaHeight} // pixels
	PlanetCenter = godot.Vec{X: PlanetX, Y: PlanetY}        // pixels
)

type Placement struct {
	Position godot.Vec // pixels, arena space
	Rotation float64   // radians
}

func LevelNumber(levelID string) int {
	digits := ""
	for _, character := range levelID {
		if character >= '0' && character <= '9' {
			digits += string(character)
		}
	}
	if digits == "" {
		return 1
	}
	number := 0
	for _, character := range digits {
		number = number*10 + int(character-'0')
	}
	return number
}
