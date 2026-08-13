package sim

import "astroswarm/worker/internal/godot"

type World struct {
	Ships           []*Ship
	RNG             *godot.RNG
	SinglePrecision bool // round positions and rotations to float32 to match Godot
}

func NewWorld(rng *godot.RNG, singlePrecision bool) *World {
	return &World{RNG: rng, SinglePrecision: singlePrecision}
}

func (w *World) Add(ship *Ship) {
	ship.world = w
	w.Ships = append(w.Ships, ship)
}

func (w *World) Remove(target *Ship) {
	kept := w.Ships[:0]
	for _, ship := range w.Ships {
		if ship != target {
			kept = append(kept, ship)
		}
	}
	w.Ships = kept
}

// delta is the tick length in seconds.
func (w *World) Step(delta float64) {
	for _, ship := range w.Ships {
		ship.updateVisible()
	}
	for _, ship := range w.Ships {
		ship.step(delta)
	}
}
