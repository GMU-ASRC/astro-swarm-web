package bench

import (
	"math"

	"astroswarm/worker/internal/godot"
	"astroswarm/worker/internal/sim"
)

const (
	MillLinkDistance       = 240.0
	MillMinimumSize        = 2
	CirclinessSampleStride = 3
)

type CirclinessTracker struct {
	total   float64
	samples int
}

func (t *CirclinessTracker) Sample(ships []*sim.Ship) {
	average, mills := MillCircliness(ships)
	if mills == 0 {
		return
	}
	t.total += average
	t.samples++
}

func (t *CirclinessTracker) Average() float64 {
	if t.samples == 0 {
		return -1.0
	}
	return t.total / float64(t.samples)
}

func MillCircliness(ships []*sim.Ship) (float64, int) {
	positions := make([]godot.Vec, len(ships))
	headings := make([]float64, len(ships))
	for index, ship := range ships {
		positions[index] = ship.Position
		headings[index] = ship.Rotation
	}
	total := 0.0
	mills := 0
	for _, members := range Mills(positions, MillLinkDistance) {
		if len(members) < MillMinimumSize {
			continue
		}
		millPositions := make([]godot.Vec, len(members))
		millHeadings := make([]float64, len(members))
		for slot, index := range members {
			millPositions[slot] = positions[index]
			millHeadings[slot] = headings[index]
		}
		total += Circliness(millPositions, millHeadings)
		mills++
	}
	if mills == 0 {
		return -1.0, 0
	}
	return total / float64(mills), mills
}

func Mills(positions []godot.Vec, linkDistance float64) [][]int {
	assigned := make([]bool, len(positions))
	groups := [][]int{}
	for start := range positions {
		if assigned[start] {
			continue
		}
		assigned[start] = true
		members := []int{start}
		frontier := []int{start}
		for len(frontier) > 0 {
			index := frontier[len(frontier)-1]
			frontier = frontier[:len(frontier)-1]
			for other := range positions {
				if assigned[other] || positions[index].DistanceTo(positions[other]) > linkDistance {
					continue
				}
				assigned[other] = true
				members = append(members, other)
				frontier = append(frontier, other)
			}
		}
		groups = append(groups, members)
	}
	return groups
}

func Circliness(positions []godot.Vec, headings []float64) float64 {
	if len(positions) < 2 || len(headings) != len(positions) {
		return 0.0
	}
	center := godot.Vec{}
	for _, position := range positions {
		center = center.Add(position)
	}
	center = center.Scale(1.0 / float64(len(positions)))

	smallest := math.Inf(1)
	largest := 0.0
	motionError := 0.0
	for index, position := range positions {
		offset := position.Sub(center)
		radius := offset.Length()
		smallest = math.Min(smallest, radius)
		largest = math.Max(largest, radius)
		motionError += math.Abs(math.Cos(godot.AngleDifference(headings[index], offset.Angle())))
	}
	if largest <= 0.001 {
		return 0.0
	}
	motionError /= float64(len(positions))
	shapeError := 1.0 - (smallest*smallest)/(largest*largest)
	return godot.Clamp(1.0-math.Max(shapeError, motionError), 0.0, 1.0)
}
