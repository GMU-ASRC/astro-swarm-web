package stress

import (
	"math"

	"astroswarm/worker/internal/godot"
)

type placement struct {
	position godot.Vec
	rotation float64
}

func planetCenter(options Options) godot.Vec {
	return godot.Vec{X: options.ArenaWidth * 0.5, Y: options.ArenaHeight * 0.5}
}

func defenderPlacements(options Options, rng *godot.RNG) []placement {
	switch options.Spawn {
	case SpawnRing:
		return ringPlacements(options, rng)
	case SpawnArena:
		return arenaPlacements(options, rng)
	}
	return areaPlacements(options, rng)
}

func areaPlacements(options Options, rng *godot.RNG) []placement {
	center := planetCenter(options)
	placements := make([]placement, 0, options.Ships)
	for index := 0; index < options.Ships; index++ {
		position := areaPoint(options, center, rng)
		for attempt := 0; attempt < SpawnAttempts; attempt++ {
			if isClearOf(position, placements, options.SpawnSpacing) {
				break
			}
			position = areaPoint(options, center, rng)
		}
		placements = append(placements, placement{position: position, rotation: rng.Randf() * godot.Tau})
	}
	return placements
}

func areaPoint(options Options, center godot.Vec, rng *godot.RNG) godot.Vec {
	angle := rng.Randf() * godot.Tau
	radius := rng.RandfRange(options.SpawnInner, options.SpawnOuter)
	return center.Add(godot.Vec{X: radius}.Rotated(angle))
}

func isClearOf(position godot.Vec, placements []placement, spacing float64) bool {
	if spacing <= 0 {
		return true
	}
	spacingSquared := spacing * spacing
	for _, existing := range placements {
		if position.DistanceSquaredTo(existing.position) < spacingSquared {
			return false
		}
	}
	return true
}

func arenaPlacements(options Options, rng *godot.RNG) []placement {
	center := planetCenter(options)
	keepOut := math.Max(options.SpawnInner, options.PlanetRadius)
	placements := make([]placement, 0, options.Ships)
	for index := 0; index < options.Ships; index++ {
		position := randomArenaPoint(options, rng)
		for attempt := 0; attempt < SpawnAttempts && position.DistanceTo(center) < keepOut; attempt++ {
			position = randomArenaPoint(options, rng)
		}
		placements = append(placements, placement{position: position, rotation: rng.Randf() * godot.Tau})
	}
	return placements
}

func randomArenaPoint(options Options, rng *godot.RNG) godot.Vec {
	return godot.Vec{
		X: rng.RandfRange(EdgeMargin, options.ArenaWidth-EdgeMargin),
		Y: rng.RandfRange(EdgeMargin, options.ArenaHeight-EdgeMargin),
	}
}

func ringPlacements(options Options, rng *godot.RNG) []placement {
	center := planetCenter(options)
	offset := rng.Randf() * godot.Tau
	placements := make([]placement, 0, options.Ships)
	for index := 0; index < options.Ships; index++ {
		angle := offset + godot.Tau*float64(index)/float64(options.Ships)
		placements = append(placements, placement{
			position: center.Add(godot.Vec{X: options.RingRadius}.Rotated(angle)),
			rotation: rng.Randf() * godot.Tau,
		})
	}
	return placements
}

func evaderSpawnPoint(options Options, angle float64) godot.Vec {
	center := planetCenter(options)
	if options.EvaderSpawn == EvaderSpawnRing {
		return center.Add(godot.Vec{X: options.EvaderRingRadius}.Rotated(angle))
	}
	direction := godot.FromAngle(angle)
	halfWidth := options.ArenaWidth*0.5 - EdgeSpawnInset
	halfHeight := options.ArenaHeight*0.5 - EdgeSpawnInset
	reach := math.Inf(1)
	if math.Abs(direction.X) > 0.0001 {
		reach = math.Min(reach, halfWidth/math.Abs(direction.X))
	}
	if math.Abs(direction.Y) > 0.0001 {
		reach = math.Min(reach, halfHeight/math.Abs(direction.Y))
	}
	return center.Add(direction.Scale(reach))
}
