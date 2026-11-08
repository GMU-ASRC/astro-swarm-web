package evaluations

import (
	"net/http"
	"sort"
	"strconv"

	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/values"
)

func (api *API) levelSweep(w http.ResponseWriter, r *http.Request) error {
	levelID := levelQueryValue(r, "farp1")
	limit := 6
	if text := r.URL.Query().Get("limit"); text != "" {
		if parsed, err := strconv.Atoi(text); err == nil {
			limit = max(1, min(12, parsed))
		}
	}
	ctx := r.Context()
	ranked, err := api.doneOnLevel(r, levelID,
		" AND benchmarks.success_rate IS NOT NULL ORDER BY benchmarks.success_rate DESC, player_evaluations.created_at ASC")
	if err != nil {
		return err
	}
	rows := []any{}
	for _, entry := range ranked {
		if len(rows) >= limit {
			break
		}
		summary, err := entries.Summary(ctx, api.deps.Store.Pool, entry)
		if err != nil {
			return err
		}
		sweep, _ := summary["sweep"].([]any)
		if len(sweep) == 0 {
			continue
		}
		sweepRows := make([]any, 0, len(sweep))
		for _, point := range sweep {
			pointObject, _ := point.(map[string]any)
			sweepRows = append(sweepRows, sweepRow(pointObject))
		}
		attrition := summary["attrition"]
		if !values.Truthy(attrition) {
			attrition = []any{}
		}
		rows = append(rows, map[string]any{
			"id": entry.ID, "username": entry.Username, "player_id": entry.PlayerID,
			"success_rate": values.RoundTo(*entry.SuccessRate, 1), "defenders": intValue(entry.DefenderCount),
			"created_at": pytime.IsoformatOrNil(entry.CreatedAt), "sweep": sweepRows, "attrition": attrition,
			"circliness": summary["circliness"],
		})
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
	return nil
}

func sweepRow(point map[string]any) map[string]any {
	rateValue, present := point["capture_rate"]
	if !present || rateValue == nil {
		rateValue = point["success_rate"]
	}
	rate := 0.0
	if values.Truthy(rateValue) {
		rate, _ = values.ToFloat(rateValue)
	}
	rate = values.RoundTo(rate, 1)
	row := map[string]any{"n": point["n"], "capture_rate": rate, "risk": values.RoundTo(100.0-rate, 1)}
	if circliness, present := point["circliness"]; present && circliness != nil {
		if number, err := values.ToFloat(circliness); err == nil {
			row["circliness"] = values.RoundTo(number, 3)
		}
	}
	return row
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func (api *API) profile(w http.ResponseWriter, r *http.Request) error {
	playerID := r.PathValue("player_id")
	table, err := api.allPlayers(r)
	if err != nil {
		return err
	}
	me := table.byID[playerID]
	if me == nil {
		return httpx.NotFound("Player not found")
	}
	ranked := table.sortedBySuccess()
	var overallRank any
	for index, current := range ranked {
		if current.playerID == playerID {
			overallRank = index + 1
			break
		}
	}
	weightedByLevel := map[int]map[string]any{}
	for _, entry := range me.ratingLevels {
		weightedByLevel[entry["level_number"].(int)] = entry
	}
	levelNumbers := make([]int, 0, len(me.levels))
	for levelNumber := range me.levels {
		levelNumbers = append(levelNumbers, levelNumber)
	}
	sort.Ints(levelNumbers)
	levelsOut := []any{}
	for _, levelNumber := range levelNumbers {
		contenders := []*player{}
		for _, current := range table.order {
			if level := current.levels[levelNumber]; level != nil && level.successRate != nil {
				contenders = append(contenders, current)
			}
		}
		sort.SliceStable(contenders, func(left, right int) bool {
			return weightedLevelRate(contenders[left], levelNumber) > weightedLevelRate(contenders[right], levelNumber)
		})
		var rank any
		for index, current := range contenders {
			if current.playerID == playerID {
				rank = index + 1
				break
			}
		}
		level := me.levels[levelNumber]
		weighted := weightedByLevel[levelNumber]
		levelsOut = append(levelsOut, map[string]any{
			"level_number": levelNumber, "success_rate": optionalNumber(level.successRate),
			"weighted_rate": weighted["weighted_rate"], "level_average": weighted["level_average"],
			"entries": level.count, "xp": level.xp, "rank": rank, "players": len(contenders),
		})
	}
	recent, err := entries.List(r.Context(), api.deps.Store.Pool,
		entries.SelectEntries("player_evaluations.player_id = $1 AND player_evaluations.status IS DISTINCT FROM 'cancelled'")+
			" ORDER BY player_evaluations.created_at DESC", playerID)
	if err != nil {
		return err
	}
	ratingLevels := make([]any, len(me.ratingLevels))
	for index, entry := range me.ratingLevels {
		ratingLevels[index] = entry
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"player_id": playerID, "username": me.username, "total_xp": me.totalXP,
		"overall_success": optionalNumber(me.overallSuccess), "best_success": optionalNumber(me.bestSuccess),
		"rating": optionalNumber(me.rating), "rating_levels": ratingLevels, "levels_played": me.levelsPlayed,
		"levels_total": me.levelsTotal, "overall_rank": overallRank, "total_players": len(ranked),
		"entries": me.entries, "last_active": pytime.IsoformatOrNil(me.last), "levels": levelsOut,
		"recent_entries": listMaps(recent),
	})
	return nil
}
