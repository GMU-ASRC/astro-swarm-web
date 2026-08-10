package bench

import "astroswarm/worker/internal/godot"

const (
	PhysicsTicksPerSecond = 60

	ArenaWidth   = 3840.0
	ArenaHeight  = 2160.0
	PlanetX      = 1920.0
	PlanetY      = 1080.0
	PlanetRadius = 120.0
	GoalMargin   = 16.0

	MaxDefenders = 6
	EnemySpeed   = 105.0

	DefaultTrials          = 100
	DefaultSweepMax        = 100
	DefaultSweepTrials     = 100
	DefaultSeed            = 987654321
	DefaultMatchSeconds    = 240.0
	DefaultGoalTailSeconds = 3.0

	SweepRadius      = 300.0
	PlaceMin         = PlanetRadius + 50.0
	ScatterMax       = 250.0
	ScatterSpacing   = 110.0
	ScatterAttempts  = 40
	RingCount        = 5
	EnemySpawnRadius = 1000.0

	ReplaySweepNMax   = 50
	ReplaySweepTrials = 10

	SweepSeedOffset      = 100000
	SweepSpawnSeedOffset = 300000
	SweepSeedStride      = 1000000
	SweepMatchOffset     = 500000
	PlacementSeedOffset  = 700000

	OutcomeWin     = "win"
	OutcomeLose    = "lose"
	OutcomeTimeout = "timeout"
)

var (
	ArenaSize    = godot.Vec{X: ArenaWidth, Y: ArenaHeight}
	PlanetCenter = godot.Vec{X: PlanetX, Y: PlanetY}
)

type Placement struct {
	Position godot.Vec
	Rotation float64
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
