package blocks

import (
	"math"

	"astroswarm/worker/internal/godot"
)

// PixelsPerMeter converts between the meters block authors type in the editor
// and the pixels the simulation runs in.
const PixelsPerMeter = 40.0

type ShipConfig struct {
	ViewDistance float64 // pixels
	FovDegrees   float64 // degrees, full width of the vision cone
	Speed        float64 // pixels/second
	TurnSpeed    float64 // radians/second
	HullRadius   float64 // pixels
}

func ConfigFromScripts(scripts []Script, base ShipConfig) ShipConfig {
	config := base
	for _, script := range scripts {
		collectConfig(script.Blocks, &config)
	}
	return config
}

func collectConfig(list []Block, config *ShipConfig) {
	for _, block := range list {
		if block.Type == "when_start" {
			collectConfig(block.Children, config)
			continue
		}
		ApplyConfigBlock(block.Type, block.Params, config)
	}
}

func ApplyConfigBlock(blockType string, params map[string]any, config *ShipConfig) bool {
	switch blockType {
	case "set_speed":
		config.Speed = ParamFloat(params, "value", config.Speed/PixelsPerMeter) * PixelsPerMeter
	case "set_turn":
		config.TurnSpeed = godot.DegToRad(ParamFloat(params, "value", 120.0))
	case "set_view":
		config.ViewDistance = ParamFloat(params, "value", config.ViewDistance/PixelsPerMeter) * PixelsPerMeter
	case "set_fov":
		config.FovDegrees = math.Min(180.0, ParamFloat(params, "value", config.FovDegrees))
	case "set_size":
		config.HullRadius = ParamFloat(params, "value", 6.0)
	default:
		return false
	}
	return true
}
