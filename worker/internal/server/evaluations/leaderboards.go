package evaluations

import (
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/settings"
	"astroswarm/worker/internal/server/values"
)

func levelQueryValue(r *http.Request, fallback string) string {
	levelID := strings.TrimSpace(r.URL.Query().Get("level_id"))
	if levelID == "" {
		return fallback
	}
	return levelID
}

func (api *API) list(w http.ResponseWriter, r *http.Request) error {
	conditions := []string{}
	arguments := []any{}
	query := r.URL.Query()
	if levelID := strings.TrimSpace(query.Get("level_id")); levelID != "" {
		arguments = append(arguments, settings.LevelIDsFor(levelID))
		conditions = append(conditions, "player_evaluations.level_id = ANY($"+strconv.Itoa(len(arguments))+")")
	}
	switch query.Get("exclude_cancelled") {
	case "1", "true", "yes":
		conditions = append(conditions, "player_evaluations.status IS DISTINCT FROM 'cancelled'")
	}
	if playerID := strings.TrimSpace(query.Get("player_id")); playerID != "" {
		arguments = append(arguments, playerID)
		conditions = append(conditions, "player_evaluations.player_id = $"+strconv.Itoa(len(arguments)))
	}
	listed, err := entries.List(r.Context(), api.deps.Store.Pool,
		entries.SelectEntries(strings.Join(conditions, " AND "))+" ORDER BY player_evaluations.created_at DESC LIMIT 500",
		arguments...)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, listMaps(listed))
	return nil
}

func listMaps(listed []*entries.Entry) []any {
	output := make([]any, 0, len(listed))
	for _, entry := range listed {
		output = append(output, entry.ToListMap())
	}
	return output
}

func (api *API) doneOnLevel(r *http.Request, levelID string, order string) ([]*entries.Entry, error) {
	return entries.List(r.Context(), api.deps.Store.Pool,
		entries.SelectEntries("player_evaluations.level_id = ANY($1) AND player_evaluations.status = 'done'")+order,
		settings.LevelIDsFor(levelID))
}

func (api *API) best(w http.ResponseWriter, r *http.Request) error {
	levelID := levelQueryValue(r, "farp2")
	candidates, err := api.doneOnLevel(r, levelID, "")
	if err != nil {
		return err
	}
	var leaders []*entries.Entry
	bestRate := 0.0
	for _, entry := range candidates {
		if entry.SuccessRate == nil {
			continue
		}
		rate := *entry.SuccessRate
		if len(leaders) == 0 || rate > bestRate {
			leaders, bestRate = []*entries.Entry{entry}, rate
		} else if rate == bestRate {
			leaders = append(leaders, entry)
		}
	}
	if len(leaders) == 0 {
		return httpx.NotFound("No submitted algorithm for this level yet")
	}
	winner := leaders[rand.IntN(len(leaders))]
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"id": winner.ID, "level_id": winner.LevelID, "username": winner.Username, "player_id": winner.PlayerID,
		"success_rate": values.RoundTo(bestRate, 1), "algorithm": winner.AlgorithmValue(), "placements": winner.PlacementsValue(),
	})
	return nil
}

func (api *API) allPlayers(r *http.Request) (*playerTable, error) {
	listed, err := entries.List(r.Context(), api.deps.Store.Pool,
		entries.SelectEntries("player_evaluations.status IS DISTINCT FROM 'cancelled'")+" ORDER BY player_evaluations.created_at ASC")
	if err != nil {
		return nil, err
	}
	return aggregatePlayers(listed), nil
}

func (api *API) players(w http.ResponseWriter, r *http.Request) error {
	table, err := api.allPlayers(r)
	if err != nil {
		return err
	}
	rows := []any{}
	for index, current := range table.sortedBySuccess() {
		rows = append(rows, map[string]any{
			"player_id": current.playerID, "username": current.username, "total_xp": current.totalXP,
			"entries": current.entries, "best_success": optionalNumber(current.bestSuccess),
			"overall_success": optionalNumber(current.overallSuccess), "rating": optionalNumber(current.rating),
			"levels_played": current.levelsPlayed, "levels_total": current.levelsTotal, "rank": index + 1,
			"last_active": pytime.IsoformatOrNil(current.last),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
	return nil
}

type levelBoardRow struct {
	playerID, username, bestEntryID string
	entries, bestDefenders          int
	rateSum, bestRate               float64
	bestCreatedAt, lastActive       *timeValue
}

func (api *API) levelLeaderboard(w http.ResponseWriter, r *http.Request) error {
	levelID := levelQueryValue(r, "farp1")
	listed, err := api.doneOnLevel(r, levelID, " ORDER BY player_evaluations.created_at ASC")
	if err != nil {
		return err
	}
	byPlayer := map[string]*levelBoardRow{}
	order := []*levelBoardRow{}
	for _, entry := range listed {
		if entry.SuccessRate == nil {
			continue
		}
		rate := *entry.SuccessRate
		defenders := 0
		if entry.DefenderCount != nil {
			defenders = *entry.DefenderCount
		}
		row := byPlayer[entry.PlayerID]
		if row == nil {
			row = &levelBoardRow{playerID: entry.PlayerID, username: entry.Username, bestRate: rate, bestEntryID: entry.ID,
				bestDefenders: defenders, bestCreatedAt: newTimeValue(entry.CreatedAt), lastActive: newTimeValue(entry.CreatedAt)}
			byPlayer[entry.PlayerID] = row
			order = append(order, row)
		}
		row.entries++
		row.rateSum += rate
		if rate > row.bestRate {
			row.bestRate, row.bestEntryID, row.bestDefenders = rate, entry.ID, defenders
			row.bestCreatedAt = newTimeValue(entry.CreatedAt)
		}
		if entry.CreatedAt != nil && (row.lastActive == nil || !entry.CreatedAt.Before(row.lastActive.moment)) {
			row.username = entry.Username
			row.lastActive = newTimeValue(entry.CreatedAt)
		}
	}
	sort.SliceStable(order, func(left, right int) bool {
		if order[left].bestRate != order[right].bestRate {
			return order[left].bestRate > order[right].bestRate
		}
		return order[left].bestCreatedAt.before(order[right].bestCreatedAt)
	})
	rows := []any{}
	for index, row := range order {
		rows = append(rows, map[string]any{
			"rank": index + 1, "player_id": row.playerID, "username": row.username, "entries": row.entries,
			"best_rate": values.RoundTo(row.bestRate, 1), "average_rate": values.RoundTo(row.rateSum/float64(row.entries), 1),
			"best_entry_id": row.bestEntryID, "best_defenders": row.bestDefenders, "last_active": row.lastActive.isoformat(),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
	return nil
}

func (api *API) baseline(w http.ResponseWriter, r *http.Request) error {
	listed, err := entries.List(r.Context(), api.deps.Store.Pool, entries.SelectEntries("player_evaluations.status = 'done'"))
	if err != nil {
		return err
	}
	total, samples := 0.0, 0
	for _, entry := range listed {
		if entry.SuccessRate != nil {
			total += *entry.SuccessRate
			samples++
		}
	}
	var average any
	if samples > 0 {
		average = values.RoundTo(total/float64(samples), 1)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success_rate": average, "samples": samples})
	return nil
}
