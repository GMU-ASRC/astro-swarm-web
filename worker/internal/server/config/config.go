package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL      string
	UploadDir        string
	BackupDir        string
	ClientDir        string
	Port             string
	APISecretKey     string
	PublicBaseURL    string
	CORSOrigins      []string
	SimulatorVersion string
	DatabasePoolSize int32

	EvalSeed             int
	EvalSweepMax         int
	EvalSweepTrials      int
	EvalMatchCapSeconds  int
	EvalGoalTailSeconds  float64
	EvalEnemyX           float64
	EvalEnemyY           float64
	EvalSweepSpawn       string
	MaxUploadBytes       int64
	MaxWorkerResultBytes int64
}

const (
	EvalArenaWidth  = 3840.0
	EvalArenaHeight = 2160.0
)

func Load() Config {
	return Config{
		DatabaseURL:      databaseURL(env("DATABASE_URL", "")),
		UploadDir:        env("UPLOAD_DIR", "/app/uploads"),
		BackupDir:        env("BACKUP_DIR", "/app/backups"),
		ClientDir:        env("CLIENT_DIR", "/app/client"),
		Port:             env("PORT", "5050"),
		APISecretKey:     env("API_SECRET_KEY", "dev_secret_key"),
		PublicBaseURL:    env("PUBLIC_BASE_URL", ""),
		CORSOrigins:      corsOrigins(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		SimulatorVersion: env("SIMULATOR_VERSION", "1"),
		DatabasePoolSize: int32(envInt("DB_POOL_SIZE", 16)),

		EvalSeed:             envInt("EVAL_SEED", 987654321),
		EvalSweepMax:         envInt("EVAL_SWEEP_MAX", 100),
		EvalSweepTrials:      envInt("EVAL_SWEEP_TRIALS", 10),
		EvalMatchCapSeconds:  envInt("EVAL_MATCH_CAP_SECONDS", 240),
		EvalGoalTailSeconds:  envFloat("EVAL_GOAL_TAIL_SECONDS", 3),
		EvalEnemyX:           envFloat("EVAL_ENEMY_X", 1920),
		EvalEnemyY:           envFloat("EVAL_ENEMY_Y", 40),
		EvalSweepSpawn:       env("EVAL_SWEEP_SPAWN", "varied"),
		MaxUploadBytes:       10 * 1024 * 1024,
		MaxWorkerResultBytes: 512 * 1024 * 1024,
	}
}

func databaseURL(value string) string {
	for _, driverPrefix := range []string{"postgresql+psycopg2://", "postgresql+psycopg://"} {
		if strings.HasPrefix(value, driverPrefix) {
			return "postgresql://" + strings.TrimPrefix(value, driverPrefix)
		}
	}
	return value
}

func corsOrigins(value string) []string {
	trimmed := strings.TrimSpace(value)
	var fromJSON []string
	if strings.HasPrefix(trimmed, "[") && json.Unmarshal([]byte(trimmed), &fromJSON) == nil {
		return fromJSON
	}
	origins := []string{}
	for _, origin := range strings.Split(trimmed, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func env(key string, fallback string) string {
	if value, found := os.LookupEnv(key); found {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return value
}

func envFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(key)), 64)
	if err != nil {
		return fallback
	}
	return value
}
