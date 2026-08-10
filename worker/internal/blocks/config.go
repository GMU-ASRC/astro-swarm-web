package blocks

import (
	"math"

	"astroswarm/worker/internal/godot"
)

const PixelsPerMeter = 40.0

type ShipConfig struct {
	ViewDistance float64
	FovDegrees   float64
	Speed        float64
	TurnSpeed    float64
	HullRadius   float64
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
		switch block.Type {
		case "set_speed":
			config.Speed = ParamFloat(block.Params, "value", config.Speed/PixelsPerMeter) * PixelsPerMeter
		case "set_turn":
			config.TurnSpeed = godot.DegToRad(ParamFloat(block.Params, "value", 120.0))
		case "set_view":
			config.ViewDistance = ParamFloat(block.Params, "value", config.ViewDistance/PixelsPerMeter) * PixelsPerMeter
		case "set_fov":
			config.FovDegrees = math.Min(180.0, ParamFloat(block.Params, "value", config.FovDegrees))
		case "set_size":
			config.HullRadius = ParamFloat(block.Params, "value", 6.0)
		case "when_start":
			collectConfig(block.Children, config)
		}
	}
}
