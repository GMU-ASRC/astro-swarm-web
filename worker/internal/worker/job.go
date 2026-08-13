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

var ErrCancelled = errors.New("job cancelled")

type progressReporter func(done int, stage string)

func (w *Worker) executeJob(ctx context.Context, job Job, report progressReporter) (bench.JobResult, error) {
	if job.Run != nil {
		report(1, "Rendering piloted run")
		return renderPilotRun(*job.Run), nil
	}

	config := job.Config
	levelID := config.LevelID
	if levelID == "" {
		levelID = "farp1"
	}

	// A piloted level has nothing to simulate. Without the recording there is no
	// replay to rebuild, so fail loudly rather than benchmark the wrong thing.
	if bench.IsPilotLevel(levelID) {
		return bench.JobResult{}, fmt.Errorf("level %d is piloted but job %s carries no recorded run", bench.LevelNumber(levelID), job.JobID)
	}

	algorithm := blocks.NormalizeToScripts(job.Algorithm)
	if len(algorithm) == 0 {
		return bench.JobResult{}, fmt.Errorf("job %s carries no algorithm blocks", job.JobID)
	}

	options := bench.Options{
		LevelID:    levelID,
		Algorithm:  algorithm,
		Placements: job.benchPlacements(),
		Trials:     job.Trials,
		// A job is a whole evaluation, so it always runs the full range.
		TrialStart:       0,
		TrialCount:       job.Trials,
		NStart:           1,
		NCount:           config.SweepMax,
		Seed:             config.Seed,
		SweepMax:         config.SweepMax,
		SweepTrials:      config.SweepTrials,
		MatchSeconds:     config.MatchSeconds,
		GoalTailSeconds:  config.GoalTailSeconds,
		EnemyStart:       godot.Vec{X: config.EnemyX, Y: config.EnemyY},
		HasEnemyStart:    true,
		Collisions:       config.Collisions,
		VariedSweepSpawn: w.variedSweepSpawn(config),
		Record:           true,
		Workers:          w.settings.SimWorkers,
		Context:          ctx,
		Progress:         jobProgress(job, report),
	}

	_, result := bench.RunJob(options)
	if ctx.Err() != nil {
		return bench.JobResult{}, ErrCancelled
	}
	return result, nil
}

// The server decides how the ring-sweep evader spawns, so every worker and the
// CLI grade an entry the same way. The local setting is only a fallback for a
// server that does not send one.
func (w *Worker) variedSweepSpawn(config JobConfig) bool {
	if config.SweepSpawn == "" {
		return w.settings.VariedSweepSpawn
	}
	return config.SweepSpawn == bench.SweepSpawnVaried
}

func jobProgress(job Job, report progressReporter) func(done, total int) {
	placementUnits := job.Trials
	return func(done, total int) {
		stage := ""
		if done <= placementUnits {
			stage = fmt.Sprintf("Placement: trial %d/%d", done, placementUnits)
		} else {
			stage = fmt.Sprintf("Ring sweep: %d/%d runs over n=1..%d",
				done-placementUnits, total-placementUnits, job.Config.SweepMax)
		}
		report(done, stage)
	}
}

func renderPilotRun(run PilotRun) bench.JobResult {
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

	return bench.JobResult{
		Runs: []bench.ReplayRun{{
			Trial:         0,
			Outcome:       outcome,
			DetectionTime: run.DetectionTime,
			CaptureTime:   run.CaptureTime,
			GoalTime:      run.GoalTime,
			FramesPacked:  bench.PackFrames(run.Frames),
			Stats:         run.Stats,
		}},
		SweepRuns: []bench.ReplaySweepRun{},
		Meta: bench.JobMeta{
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
