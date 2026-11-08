package app

import (
	"context"
	"time"

	"astroswarm/worker/internal/server/benchmarks"
)

const (
	retireSweepEvery     = 10 * time.Minute
	retireSweepBatchSize = 500
)

func (application *App) runRetireSweeps(ctx context.Context) {
	ticker := time.NewTicker(retireSweepEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			application.retireSweep(ctx)
		}
	}
}

func (application *App) retireSweep(ctx context.Context) {
	dependencies := application.dependencies
	retired, err := benchmarks.LegacyFramesRetired(ctx, dependencies.Store.Pool)
	if err != nil || !retired {
		return
	}
	problems, err := benchmarks.RetireAllLegacy(ctx, dependencies.Store.Pool, dependencies.Files, retireSweepBatchSize, func(int) {})
	for _, problem := range problems {
		dependencies.Logger.Printf("[WARNING] Replay retire: %s", problem)
	}
	if err != nil {
		dependencies.Logger.Printf("[WARNING] Replay retire sweep stopped: %v", err)
	}
}
