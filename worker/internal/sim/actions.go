package sim

import (
	"math"
	"strings"

	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
)

const (
	levyShortestStep = 0.2
	levyLongestStep  = 3.0
)

func (s *Ship) ExecAction(blockType string, params map[string]any, delta float64, state *blocks.ActionState) bool {
	if blockType == "set_var_random" {
		low := int64(blocks.ParamFloat(params, "min", 0.0))
		high := int64(blocks.ParamFloat(params, "max", 0.0))
		s.world.RNG.RandiRange(low, high)
		return blocks.Done
	}
	if strings.HasPrefix(blockType, "set_") {
		return blocks.Done
	}
	if len(blockType) < 3 {
		return blocks.Done
	}

	switch blockType[3:] {
	case "forward":
		s.forwardInput = s.throttleMult
		return s.advanceStep(state, delta)
	case "backward":
		s.forwardInput = -s.throttleMult
		return s.advanceStep(state, delta)
	case "stop":
		s.forwardInput = 0.0
		s.turnInput = 0.0
		s.turnCommand = 0.0
		return blocks.Done
	case "random_walk", "wander":
		s.turnCommand = 0.0
		if !state.HasHeading {
			state.HasHeading = true
			state.Heading = s.world.RNG.Randf() * godot.Tau
			state.Remaining = s.levyStepTime()
		}
		s.Rotation = state.Heading
		s.narrowRotation()
		s.forwardInput = s.throttleMult
		state.Remaining -= delta
		return state.Remaining <= 0.0
	case "turn_left":
		s.turnCommand = -godot.DegToRad(blocks.ParamFloat(params, "value", 90.0))
		return blocks.Done
	case "turn_right":
		s.turnCommand = godot.DegToRad(blocks.ParamFloat(params, "value", 90.0))
		return blocks.Done
	case "turn_left_by":
		s.turnCommand = 0.0
		return s.turnBy(state, delta, -1.0, blocks.ParamFloat(params, "value", 180.0))
	case "turn_right_by":
		s.turnCommand = 0.0
		return s.turnBy(state, delta, 1.0, blocks.ParamFloat(params, "value", 180.0))
	case "face":
		s.turnCommand = 0.0
		s.rotateToward(0.0, delta)
		return s.advanceStep(state, delta)
	case "flee":
		s.turnCommand = 0.0
		s.rotateToward(math.Pi, delta)
		return s.advanceStep(state, delta)
	case "throttle":
		s.throttleMult = blocks.ParamFloat(params, "value", 1.0)
		return blocks.Done
	}
	return blocks.Done
}

func (s *Ship) levyStepTime() float64 {
	sample := s.world.RNG.Randf()
	step := levyShortestStep / math.Max(0.001, 1.0-sample)
	return math.Min(step, levyLongestStep)
}

func (s *Ship) advanceStep(state *blocks.ActionState, delta float64) bool {
	if !state.HasStepTime {
		state.HasStepTime = true
		state.StepTime = StepTime
	}
	state.StepTime -= delta
	return state.StepTime <= 0.0
}

func (s *Ship) turnBy(state *blocks.ActionState, delta float64, direction float64, degrees float64) bool {
	if !state.HasTurnAmount {
		state.HasTurnAmount = true
		state.TurnAmount = godot.DegToRad(degrees)
	}
	step := s.TurnRate * delta
	if step >= state.TurnAmount {
		s.Rotation += direction * state.TurnAmount
		s.narrowRotation()
		return blocks.Done
	}
	s.Rotation += direction * step
	s.narrowRotation()
	state.TurnAmount -= step
	return blocks.Running
}

func (s *Ship) rotateToward(offset float64, delta float64) {
	target := s.nearestEnemy()
	if target == nil {
		return
	}
	angle := s.Position.AngleToPoint(target.Position) + offset
	s.Rotation = godot.LerpAngle(s.Rotation, angle, godot.Clamp(s.TurnRate*delta, 0.0, 1.0))
	s.narrowRotation()
}
