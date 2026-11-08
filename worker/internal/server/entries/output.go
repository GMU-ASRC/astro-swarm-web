package entries

import (
	"context"

	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
)

func Summary(ctx context.Context, db store.Querier, entry *Entry) (map[string]any, error) {
	if entry.BenchmarkID == nil {
		return map[string]any{}, nil
	}
	var raw []byte
	err := db.QueryRow(ctx, "SELECT summary FROM benchmarks WHERE id = $1", *entry.BenchmarkID).Scan(&raw)
	if store.IsNoRows(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	summary, isObject := decodeAny(raw).(map[string]any)
	if !isObject {
		return map[string]any{}, nil
	}
	return summary, nil
}

func (entry *Entry) baseMap() map[string]any {
	return map[string]any{
		"id":             entry.ID,
		"player_id":      entry.PlayerID,
		"username":       entry.Username,
		"level_id":       entry.Level(),
		"level_number":   entry.LevelNumber(),
		"is_attack":      entry.IsAttackLevel(),
		"game_version":   stringOr(entry.GameVersion, "v0.0.4"),
		"defender_count": intOr(entry.DefenderCount, 0),
		"status":         entry.Status,
		"progress":       floatOr(entry.Progress, 0.0),
		"stage":          entry.Stage,
		"trials":         entry.Trials,
		"xp_awarded":     entry.XPAwarded,
		"coins_awarded":  entry.CoinsAwarded,
		"created_at":     createdAtText(entry),
		"completed_at":   pytime.IsoformatOrNil(entry.CompletedAt),
	}
}

func (entry *Entry) ToMap(summary map[string]any) map[string]any {
	output := entry.baseMap()
	output["algorithm"] = entry.AlgorithmValue()
	output["genome_schedule"] = entry.GenomeScheduleValue()
	output["placements"] = entry.PlacementsValue()
	output["results"] = summary
	output["attacker_rate"] = summary["attacker_rate"]
	output["defender_rate"] = summary["defender_rate"]
	output["error"] = entry.Error
	return output
}

func (entry *Entry) ToListMap() map[string]any {
	output := entry.baseMap()
	output["success_rate"] = optionalFloat(entry.SuccessRate)
	output["attacker_rate"] = optionalFloat(entry.AttackerRate)
	output["defender_rate"] = optionalFloat(entry.DefenderRate)
	return output
}

func optionalFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func (entry *Entry) SuccessFraction(summary map[string]any) float64 {
	rate, isNumber := numberValue(summary["success_rate"])
	if !isNumber {
		return 0.0
	}
	return max(0.0, min(1.0, rate/100.0))
}
