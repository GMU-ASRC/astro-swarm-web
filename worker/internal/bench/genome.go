package bench

import (
	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/sim"
)

type GenomeStage struct {
	Fraction  float64
	Algorithm []blocks.Script
}

type genomeSwitcher struct {
	stages    []GenomeStage
	startLine int
	next      int
	switches  int
	frames    []int
}

func newGenomeSwitcher(stages []GenomeStage, startLine int) *genomeSwitcher {
	return &genomeSwitcher{stages: stages, startLine: startLine}
}

func (g *genomeSwitcher) update(live []*sim.Ship, recordedFrames int) {
	if g == nil || g.startLine < 1 || len(live) >= g.startLine {
		return
	}
	share := float64(len(live)) / float64(g.startLine)
	applied := -1
	for g.next < len(g.stages) && share <= g.stages[g.next].Fraction {
		applied = g.next
		g.next++
	}
	if applied < 0 {
		return
	}
	program := g.stages[applied].Algorithm
	config := blocks.ConfigFromScripts(program, defaultShipConfig())
	for _, ship := range live {
		ship.SetProgram(program)
		ship.ApplyConfig(config)
	}
	g.switches++
	g.frames = append(g.frames, recordedFrames)
}

func defaultShipConfig() blocks.ShipConfig {
	return blocks.ShipConfig{
		ViewDistance: sim.DefaultViewDistance,
		FovDegrees:   sim.DefaultFovDegrees,
		Speed:        sim.DefaultSpeed,
		TurnSpeed:    sim.DefaultTurnRate,
		HullRadius:   sim.DefaultHullRadius,
	}
}
