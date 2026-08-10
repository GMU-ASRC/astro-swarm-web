package godot

import "math/bits"

const (
	pcgMultiplier = 6364136223846793005
	pcgDefaultInc = 1442695040888963407
	pcgRandomMax  = float64(0xFFFFFFFF)
)

type RNG struct {
	state uint64
	inc   uint64
}

func NewRNG(seed uint64) *RNG {
	return &RNG{state: seed, inc: pcgDefaultInc}
}

func NewRNGFromInt(seed int64) *RNG {
	return NewRNG(uint64(seed))
}

func (r *RNG) Seed(seed uint64) {
	r.state = seed
}

func (r *RNG) SeedInt(seed int64) {
	r.state = uint64(seed)
}

func (r *RNG) Randi() uint32 {
	previous := r.state
	r.state = previous*pcgMultiplier + r.inc
	xorshifted := uint32(((previous >> 18) ^ previous) >> 27)
	rotation := uint32(previous >> 59)
	return bits.RotateLeft32(xorshifted, -int(rotation))
}

func (r *RNG) Randf() float64 {
	return float64(r.Randi()) / pcgRandomMax
}

func (r *RNG) RandfRange(from, to float64) float64 {
	return r.Randf()*(to-from) + from
}

func (r *RNG) RandiRange(from, to int64) int64 {
	if from == to {
		return from
	}
	span := to - from
	if span < 0 {
		span = -span
	}
	lowest := from
	if to < from {
		lowest = to
	}
	return int64(r.Randi()%uint32(span+1)) + lowest
}
