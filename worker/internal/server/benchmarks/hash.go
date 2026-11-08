package benchmarks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"astroswarm/worker/internal/server/config"
	"astroswarm/worker/internal/server/settings"
)

type HashInput struct {
	LevelID        string
	Algorithm      any
	Placements     any
	GenomeSchedule []any
	Trials         any
	Collisions     bool
}

func SweepParams(ctx context.Context, appSettings *settings.Settings, levelID string) (int, int, error) {
	if settings.IsAssaultLevel(levelID) {
		return settings.AssaultSweepMaxFor(levelID), settings.AssaultSweepTrials, nil
	}
	sweepMax, err := appSettings.SweepMax(ctx)
	if err != nil {
		return 0, 0, err
	}
	sweepTrials, err := appSettings.SweepTrials(ctx)
	return sweepMax, sweepTrials, err
}

func JobConfig(ctx context.Context, appSettings *settings.Settings, serverConfig config.Config, levelID string, collisions bool) (map[string]any, error) {
	enemyX, enemyY, err := appSettings.EnemyStart(ctx)
	if err != nil {
		return nil, err
	}
	sweepMax, sweepTrials, err := SweepParams(ctx, appSettings, levelID)
	if err != nil {
		return nil, err
	}
	seed, err := appSettings.Seed(ctx)
	if err != nil {
		return nil, err
	}
	if levelID == "" {
		levelID = "farp"
	}
	return map[string]any{
		"seed":              seed,
		"sweep_max":         sweepMax,
		"sweep_trials":      sweepTrials,
		"match_seconds":     serverConfig.EvalMatchCapSeconds,
		"goal_tail_seconds": serverConfig.EvalGoalTailSeconds,
		"enemy_x":           enemyX,
		"enemy_y":           enemyY,
		"sweep_spawn":       serverConfig.EvalSweepSpawn,
		"level_id":          levelID,
		"collisions":        collisions,
	}, nil
}

func ContentHash(input HashInput, jobConfig any, simulatorVersion string) (string, error) {
	levelID := input.LevelID
	if levelID == "" {
		levelID = "farp"
	}
	content := map[string]any{
		"level_id":          levelID,
		"algorithm":         orEmptyList(input.Algorithm),
		"placements":        orEmptyList(input.Placements),
		"trials":            input.Trials,
		"config":            jobConfig,
		"simulator_version": simulatorVersion,
	}
	if len(input.GenomeSchedule) > 1 {
		content["genome_schedule"] = input.GenomeSchedule
	}
	encoded, err := PythonJSON(content)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(encoded))
	return hex.EncodeToString(digest[:]), nil
}

func orEmptyList(value any) any {
	if value == nil {
		return []any{}
	}
	if list, isList := value.([]any); isList && len(list) == 0 {
		return []any{}
	}
	return value
}
