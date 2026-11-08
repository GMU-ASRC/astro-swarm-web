package evaluations

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"astroswarm/worker/internal/server/config"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/settings"
	"astroswarm/worker/internal/server/validate"
	"astroswarm/worker/internal/server/values"
)

const pilotLevelMaxXP = 1000

func derivedSeeds(seed int) []any {
	return []any{
		map[string]any{"name": "Static enemy spawn locations", "formula": "seed", "value": seed},
		map[string]any{"name": "Placement match RNG (per trial)", "formula": "seed + trial_index", "value": fmt.Sprintf("%d + trial_index", seed)},
		map[string]any{"name": "Sweep ring placement and orientations (per n, trial)", "formula": "sweep_seed[trial] + n",
			"value": fmt.Sprintf("(%d + %d + trial * %d) + n", seed, settings.SweepSeedOffset, settings.SweepSeedStride)},
		map[string]any{"name": "Sweep match RNG (per n, trial)", "formula": "sweep_seed[trial] + 500000 + n",
			"value": fmt.Sprintf("(%d + %d + trial * %d) + %d + n", seed, settings.SweepSeedOffset, settings.SweepSeedStride, settings.SweepMatchOffset)},
	}
}

func (api *API) evaluationSettings(ctx context.Context) (map[string]any, error) {
	appSettings := api.deps.Settings
	seed, err := appSettings.Seed(ctx)
	if err != nil {
		return nil, err
	}
	enemyX, enemyY, err := appSettings.EnemyStart(ctx)
	if err != nil {
		return nil, err
	}
	sweepMax, err := appSettings.SweepMax(ctx)
	if err != nil {
		return nil, err
	}
	sweepTrials, err := appSettings.SweepTrials(ctx)
	if err != nil {
		return nil, err
	}
	trialSeeds, err := appSettings.SweepTrialSeeds(ctx)
	if err != nil {
		return nil, err
	}
	levels, err := appSettings.LevelsWithState(ctx)
	if err != nil {
		return nil, err
	}
	requiredVersion, err := appSettings.RequiredGameVersion(ctx)
	if err != nil {
		return nil, err
	}
	pilotIDs := []string{}
	pilotLimits := map[string]any{}
	for _, level := range settings.PilotLevels {
		pilotIDs = append(pilotIDs, level.ID)
		pilotLimits[level.ID] = validate.RunLimitsFor(level.ID)
	}
	return map[string]any{
		"seed": seed, "sweep_max": sweepMax, "sweep_trials": sweepTrials,
		"match_cap_seconds": api.deps.Config.EvalMatchCapSeconds, "enemy_start_x": enemyX, "enemy_start_y": enemyY,
		"sweep_spawn": api.deps.Config.EvalSweepSpawn, "arena_width": config.EvalArenaWidth, "arena_height": config.EvalArenaHeight,
		"planet_x": config.EvalArenaWidth / 2, "planet_y": config.EvalArenaHeight / 2, "sweep_trial_seeds": trialSeeds,
		"levels": levels, "pilot_level_ids": pilotIDs, "pilot_run_limits": pilotLimits,
		"pilot_time_limit_seconds": validate.MaxRunSeconds, "pilot_max_fps": validate.MaxRunFPS, "pilot_max_xp": pilotLevelMaxXP,
		"goal_tail_seconds": api.deps.Config.EvalGoalTailSeconds, "required_game_version": requiredVersion,
		"assault_levels": settings.AssaultLevels, "attrition_levels": settings.AttritionLevels,
		"assault_trials": settings.AssaultTrials, "assault_clean_streak": settings.AssaultCleanStreak,
		"assault_sweep_max": settings.AssaultSweepMax, "stream_levels": settings.StreamLevels,
		"stream_sweep_max": settings.StreamSweepMax, "assault_sweep_trials": settings.AssaultSweepTrials,
		"siege_evaders": settings.SiegeEvaders, "derived_seeds": derivedSeeds(seed),
	}, nil
}

func (api *API) getSettings(w http.ResponseWriter, r *http.Request) error {
	payload, err := api.evaluationSettings(r.Context())
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, payload)
	return nil
}

func (api *API) updateSettings(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	ctx := r.Context()
	appSettings := api.deps.Settings
	_, hasX := data["enemy_start_x"]
	_, hasY := data["enemy_start_y"]
	if hasX || hasY {
		x, errX := values.ToFloat(data["enemy_start_x"])
		y, errY := values.ToFloat(data["enemy_start_y"])
		if !hasX || !hasY || errX != nil || errY != nil {
			return httpx.BadRequest("enemy_start_x and enemy_start_y must both be numbers")
		}
		if err := appSettings.SetEnemyStart(ctx, x, y); err != nil {
			return err
		}
	}
	if err := api.updateSweepSettings(ctx, data); err != nil {
		return err
	}
	if values.Truthy(data["regenerate_seeds"]) {
		if _, err := appSettings.RegenerateSeed(ctx); err != nil {
			return err
		}
	} else if seedValue, present := data["seed"]; present {
		seed, err := values.ToInt(seedValue)
		if err != nil {
			return httpx.BadRequest("seed must be an integer")
		}
		if _, err := appSettings.SetSeed(ctx, seed); err != nil {
			return err
		}
	}
	levelValue, hasLevel := data["level_id"]
	enabledValue, hasEnabled := data["enabled"]
	if hasLevel && hasEnabled {
		if err := appSettings.SetLevelEnabled(ctx, values.ToText(levelValue), values.Truthy(enabledValue)); err != nil {
			return err
		}
	}
	if versionValue, present := data["required_game_version"]; present {
		if _, err := appSettings.SetRequiredGameVersion(ctx, values.ToText(versionValue)); err != nil {
			return httpx.BadRequest(err.Error())
		}
	}
	full, err := api.evaluationSettings(ctx)
	if err != nil {
		return err
	}
	response := map[string]any{}
	for _, key := range []string{"seed", "enemy_start_x", "enemy_start_y", "sweep_max", "sweep_trials", "sweep_trial_seeds", "derived_seeds", "levels"} {
		response[key] = full[key]
	}
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (api *API) updateSweepSettings(ctx context.Context, data map[string]any) error {
	sweepMaxValue, hasMax := data["sweep_max"]
	sweepTrialsValue, hasTrials := data["sweep_trials"]
	if !hasMax && !hasTrials {
		return nil
	}
	invalid := httpx.BadRequest("sweep_max and sweep_trials must be integers")
	if hasMax && sweepMaxValue != nil {
		sweepMax, err := values.ToInt(sweepMaxValue)
		if err != nil || isFractionText(sweepMaxValue) {
			return invalid
		}
		if err := api.deps.Settings.SetSweepMax(ctx, int(sweepMax)); err != nil {
			return err
		}
	}
	if hasTrials && sweepTrialsValue != nil {
		sweepTrials, err := values.ToInt(sweepTrialsValue)
		if err != nil || isFractionText(sweepTrialsValue) {
			return invalid
		}
		if err := api.deps.Settings.SetSweepTrials(ctx, int(sweepTrials)); err != nil {
			return err
		}
	}
	return nil
}

func isFractionText(value any) bool {
	text, isText := value.(string)
	return isText && strings.Contains(text, ".")
}
