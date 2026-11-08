package settings

import (
	"context"
	"errors"
	"math/rand/v2"
	"strconv"
	"strings"

	"astroswarm/worker/internal/server/config"
	"astroswarm/worker/internal/server/store"
)

const (
	enemyXKey      = "enemy_start_x"
	enemyYKey      = "enemy_start_y"
	sweepMaxKey    = "sweep_max"
	sweepTrialsKey = "sweep_trials"
	seedKey        = "eval_seed"
	gameVersionKey = "required_game_version"
)

type Settings struct {
	db     store.Querier
	config config.Config
}

func New(db store.Querier, serverConfig config.Config) *Settings {
	return &Settings{db: db, config: serverConfig}
}

func (s *Settings) Get(ctx context.Context, key string) (string, bool, error) {
	var value *string
	err := s.db.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = $1", key).Scan(&value)
	if store.IsNoRows(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if value == nil {
		return "", true, nil
	}
	return *value, true, nil
}

func (s *Settings) Set(ctx context.Context, key string, value string) error {
	_, err := s.db.Exec(ctx,
		"INSERT INTO app_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value",
		key, value)
	return err
}

func (s *Settings) RequiredGameVersion(ctx context.Context) (string, error) {
	stored, found, err := s.Get(ctx, gameVersionKey)
	if err != nil {
		return "", err
	}
	if !found || strings.TrimSpace(stored) == "" {
		return CurrentGameVersion, nil
	}
	return strings.TrimSpace(stored), nil
}

func (s *Settings) SetRequiredGameVersion(ctx context.Context, version string) (string, error) {
	cleaned := strings.TrimSpace(version)
	if cleaned == "" {
		return "", errors.New("version is required")
	}
	return cleaned, s.Set(ctx, gameVersionKey, cleaned)
}

func (s *Settings) VersionIsCurrent(ctx context.Context, version string) (bool, error) {
	required, err := s.RequiredGameVersion(ctx)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(version), required), nil
}

func (s *Settings) readFloat(ctx context.Context, key string, fallback float64) (float64, error) {
	stored, found, err := s.Get(ctx, key)
	if err != nil || !found {
		return fallback, err
	}
	parsed, parseErr := strconv.ParseFloat(strings.TrimSpace(stored), 64)
	if parseErr != nil {
		return fallback, nil
	}
	return parsed, nil
}

func (s *Settings) readInt(ctx context.Context, key string, fallback int, low int, high int) (int, error) {
	stored, found, err := s.Get(ctx, key)
	if err != nil || !found {
		return fallback, err
	}
	parsed, parseErr := strconv.Atoi(strings.TrimSpace(stored))
	if parseErr != nil {
		return fallback, nil
	}
	return max(low, min(high, parsed)), nil
}

func (s *Settings) EnemyStart(ctx context.Context) (float64, float64, error) {
	x, err := s.readFloat(ctx, enemyXKey, s.config.EvalEnemyX)
	if err != nil {
		return 0, 0, err
	}
	y, err := s.readFloat(ctx, enemyYKey, s.config.EvalEnemyY)
	return x, y, err
}

func (s *Settings) SetEnemyStart(ctx context.Context, x float64, y float64) error {
	if err := s.Set(ctx, enemyXKey, FormatFloat(x)); err != nil {
		return err
	}
	return s.Set(ctx, enemyYKey, FormatFloat(y))
}

func (s *Settings) SweepMax(ctx context.Context) (int, error) {
	return s.readInt(ctx, sweepMaxKey, s.config.EvalSweepMax, SweepSettingMinimum, SweepSettingMaximum)
}

func (s *Settings) SweepTrials(ctx context.Context) (int, error) {
	return s.readInt(ctx, sweepTrialsKey, s.config.EvalSweepTrials, SweepSettingMinimum, SweepSettingMaximum)
}

func (s *Settings) SetSweepMax(ctx context.Context, sweepMax int) error {
	return s.Set(ctx, sweepMaxKey, strconv.Itoa(max(SweepSettingMinimum, min(SweepSettingMaximum, sweepMax))))
}

func (s *Settings) SetSweepTrials(ctx context.Context, sweepTrials int) error {
	return s.Set(ctx, sweepTrialsKey, strconv.Itoa(max(SweepSettingMinimum, min(SweepSettingMaximum, sweepTrials))))
}

func (s *Settings) Seed(ctx context.Context) (int, error) {
	return s.readInt(ctx, seedKey, s.config.EvalSeed, SeedMinimum, SeedMaximum)
}

func (s *Settings) SetSeed(ctx context.Context, seed int64) (int, error) {
	clamped := max(int64(SeedMinimum), min(int64(SeedMaximum), seed))
	if err := s.Set(ctx, seedKey, strconv.FormatInt(clamped, 10)); err != nil {
		return 0, err
	}
	return s.Seed(ctx)
}

func (s *Settings) RegenerateSeed(ctx context.Context) (int, error) {
	return s.SetSeed(ctx, int64(SeedMinimum)+rand.Int64N(int64(SeedMaximum-SeedMinimum+1)))
}

func (s *Settings) SweepTrialSeeds(ctx context.Context) ([]map[string]any, error) {
	seed, err := s.Seed(ctx)
	if err != nil {
		return nil, err
	}
	trials, err := s.SweepTrials(ctx)
	if err != nil {
		return nil, err
	}
	seeds := make([]map[string]any, 0, trials)
	for trial := 0; trial < trials; trial++ {
		seeds = append(seeds, map[string]any{"trial": trial + 1, "seed": seed + SweepSeedOffset + trial*SweepSeedStride})
	}
	return seeds, nil
}

func (s *Settings) LevelEnabled(ctx context.Context, levelID string) (bool, error) {
	stored, found, err := s.Get(ctx, "level_enabled_"+levelID)
	if err != nil || !found {
		return true, err
	}
	return stored == "1", nil
}

func (s *Settings) SetLevelEnabled(ctx context.Context, levelID string, enabled bool) error {
	value := "0"
	if enabled {
		value = "1"
	}
	return s.Set(ctx, "level_enabled_"+levelID, value)
}

func (s *Settings) LevelsWithState(ctx context.Context) ([]map[string]any, error) {
	levels := []map[string]any{}
	for _, level := range AllLevels() {
		enabled, err := s.LevelEnabled(ctx, level.ID)
		if err != nil {
			return nil, err
		}
		levels = append(levels, map[string]any{"id": level.ID, "name": level.Name, "enabled": enabled})
	}
	return levels, nil
}

func FormatFloat(value float64) string {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}
