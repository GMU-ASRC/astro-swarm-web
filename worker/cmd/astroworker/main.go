package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/worker"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	settings := worker.Settings{
		ServerURL:        env("SERVER_URL", "http://server:5050"),
		APIKey:           env("API_SECRET_KEY", "dev_secret_key"),
		Name:             env("WORKER_NAME", hostname()),
		Hostname:         hostname(),
		PollInterval:     envSeconds("WORKER_POLL_SECONDS", 3),
		CancelPoll:       envSeconds("WORKER_CANCEL_POLL_SECONDS", 2),
		JobTimeout:       envSeconds("EVAL_TIMEOUT_SECONDS", 3600),
		ID:               env("WORKER_ID", ""),
		IDFile:           env("WORKER_ID_FILE", "/data/worker_id"),
		Cores:            envInt("WORKER_CORES", 0, 0, 4096),
		JobSlots:         envInt("JOB_SLOTS", 0, 0, 256),
		CoresPerJob:      envInt("CORES_PER_JOB", 0, 0, 4096),
		SimWorkers:       envInt("SIM_WORKERS", 0, 0, 4096),
		UploadChunkBytes: envInt("UPLOAD_CHUNK_KB", 512, 16, 8192) * 1024,
		VariedSweepSpawn: env("EVAL_SWEEP_SPAWN", bench.SweepSpawnVaried) == bench.SweepSpawnVaried,
	}

	instance, err := worker.New(settings, logger)
	if err != nil {
		logger.Fatalf("could not start: %v", err)
	}

	capacity := instance.Capacity()
	logger.Printf("%d job slots sharing %d cores (%d host cpus), up to %d parallel matches per job, sweep_spawn=%s",
		capacity.JobSlots, capacity.Cores, runtime.NumCPU(), capacity.MatchesPerJob, env("EVAL_SWEEP_SPAWN", bench.SweepSpawnVaried))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	instance.Run(ctx)
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "worker"
	}
	return name
}

func env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int, low int, high int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func envSeconds(key string, fallback float64) time.Duration {
	seconds := fallback
	if parsed, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil && parsed > 0 {
		seconds = parsed
	}
	return time.Duration(seconds * float64(time.Second))
}
