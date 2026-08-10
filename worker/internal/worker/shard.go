package worker

import (
	"context"
	"errors"
	"fmt"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
)

const (
	defaultReplayFPS  = 30
	defaultReplayView = 300
	defaultReplayFov  = 70
)

var ErrCancelled = errors.New("shard cancelled")

type progressReporter func(done int, stage string)

func (w *Worker) executeShard(ctx context.Context, shard Shard, report progressReporter) (bench.ShardResult, error) {
	if shard.Run != nil {
		report(1, "Rendering piloted run")
		return renderPilotRun(*shard.Run), nil
	}

	algorithm := blocks.NormalizeToScripts(shard.Algorithm)
	if len(algorithm) == 0 {
		return bench.ShardResult{}, fmt.Errorf("shard %s carries no algorithm blocks", shard.ShardID)
	}

	config := shard.Config
	levelID := config.LevelID
	if levelID == "" {
		levelID = "farp1"
	}

	options := bench.Options{
		LevelID:          levelID,
		Algorithm:        algorithm,
		Placements:       shard.benchPlacements(),
		Trials:           shard.Trials,
		TrialStart:       shard.TrialStart,
		TrialCount:       shard.TrialCount,
		Seed:             config.Seed,
		SweepMax:         config.SweepMax,
		SweepTrials:      config.SweepTrials,
		NStart:           shard.NStart,
		NCount:           shard.NCount,
		MatchSeconds:     config.MatchSeconds,
		GoalTailSeconds:  config.GoalTailSeconds,
		EnemyStart:       godot.Vec{X: config.EnemyX, Y: config.EnemyY},
		HasEnemyStart:    true,
		Collisions:       config.Collisions,
		VariedSweepSpawn: w.settings.VariedSweepSpawn,
		Record:           true,
		Workers:          w.settings.SimWorkers,
		Context:          ctx,
		Progress:         shardProgress(shard, report),
	}

	_, result := bench.RunShard(options)
	if ctx.Err() != nil {
		return bench.ShardResult{}, ErrCancelled
	}
	return result, nil
}

func shardProgress(shard Shard, report progressReporter) func(done, total int) {
	placementUnits := shard.TrialCount
	return func(done, total int) {
		stage := ""
		if done <= placementUnits {
			stage = fmt.Sprintf("Placement: trial %d/%d", done, placementUnits)
		} else {
			stage = fmt.Sprintf("Ring sweep: %d/%d runs over n=%d..%d",
				done-placementUnits, total-placementUnits, shard.NStart, shard.NStart+shard.NCount-1)
		}
		report(done, stage)
	}
}

func renderPilotRun(run PilotRun) bench.ShardResult {
	fps := run.FPS
	if fps <= 0 {
		fps = defaultReplayFPS
	}
	view := run.View
	if view <= 0 {
		view = defaultReplayView
	}
	fov := run.Fov
	if fov <= 0 {
		fov = defaultReplayFov
	}
	planet := run.Planet
	if len(planet) != 3 {
		planet = []int{int(bench.PlanetX), int(bench.PlanetY), int(bench.PlanetRadius)}
	}
	arena := run.Arena
	if len(arena) != 2 {
		arena = []int{int(bench.ArenaWidth), int(bench.ArenaHeight)}
	}
	outcome := run.Outcome
	if outcome == "" {
		outcome = bench.OutcomeTimeout
	}

	return bench.ShardResult{
		Runs: []bench.ReplayRun{{
			Trial:         0,
			Outcome:       outcome,
			DetectionTime: run.DetectionTime,
			CaptureTime:   run.CaptureTime,
			GoalTime:      run.GoalTime,
			FramesPacked:  bench.PackFrames(run.Frames),
		}},
		SweepRuns: []bench.ReplaySweepRun{},
		Meta: bench.ShardMeta{
			FPS:       fps,
			Defenders: run.Defenders,
			View:      view,
			Fov:       fov,
			Speed:     run.Speed,
			Planet:    planet,
			Arena:     arena,
		},
	}
}
