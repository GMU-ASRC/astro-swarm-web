package bench

import "astroswarm/worker/internal/godot"

func SpawnPoints(seed int64, trials int) []godot.Vec {
	return spawnRing(seed, trials)
}

func SweepSpawnPoints(seed int64, sweepTrials int) []godot.Vec {
	return spawnRing(seed+SweepSpawnSeedOffset, sweepTrials)
}

func spawnRing(seed int64, trials int) []godot.Vec {
	rng := godot.NewRNGFromInt(seed)
	count := trials
	if count < 1 {
		count = 1
	}
	points := make([]godot.Vec, 0, count)
	for index := 0; index < count; index++ {
		angle := (float64(index) + 0.2 + rng.Randf()*0.6) / float64(count) * godot.Tau
		points = append(points, PlanetCenter.Add(godot.Vec{X: EnemySpawnRadius}.Rotated(angle)))
	}
	return points
}

func SweepTrialSeed(seed int64, trial int, sweepTrials int) int64 {
	if sweepTrials < 1 {
		return seed + SweepSeedOffset
	}
	return seed + SweepSeedOffset + int64(trial%sweepTrials)*SweepSeedStride
}

// Every defender owns an equal sector of the ring and sits at a random angle
// inside it, so coverage stays roughly uniform while no two trials repeat a
// layout. The inset keeps neighbours apart at high defender counts.
func RingPlacements(seed int64, trial int, sweepTrials int, count int) []Placement {
	rng := godot.NewRNGFromInt(SweepTrialSeed(seed, trial, sweepTrials) + int64(count))
	ringOffset := rng.Randf() * godot.Tau
	span := 1.0 - 2.0*RingSectorInset
	placements := make([]Placement, 0, count)
	for index := 0; index < count; index++ {
		sector := float64(index) + RingSectorInset + rng.Randf()*span
		angle := ringOffset + godot.Tau*sector/float64(count)
		placements = append(placements, Placement{
			Position: PlanetCenter.Add(godot.Vec{X: SweepRadius}.Rotated(angle)),
			Rotation: rng.Randf() * godot.Tau,
		})
	}
	return placements
}

func ScatterLayout(seed int64, count int) []Placement {
	rng := godot.NewRNGFromInt(seed + PlacementSeedOffset)
	placements := make([]Placement, 0, count)
	for index := 0; index < count; index++ {
		position := scatterPoint(rng)
		for attempt := 0; attempt < ScatterAttempts; attempt++ {
			if isClearOf(position, placements) {
				break
			}
			position = scatterPoint(rng)
		}
		placements = append(placements, Placement{Position: position, Rotation: rng.Randf() * godot.Tau})
	}
	return placements
}

func scatterPoint(rng *godot.RNG) godot.Vec {
	angle := rng.Randf() * godot.Tau
	radius := rng.RandfRange(PlaceMin, ScatterMax)
	return PlanetCenter.Add(godot.Vec{X: radius}.Rotated(angle))
}

func isClearOf(position godot.Vec, placements []Placement) bool {
	for _, placement := range placements {
		if position.DistanceTo(placement.Position) < ScatterSpacing {
			return false
		}
	}
	return true
}
