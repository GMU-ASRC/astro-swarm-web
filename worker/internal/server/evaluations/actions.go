package evaluations

import (
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"astroswarm/worker/internal/server/benchmarks"
	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/settings"
	"astroswarm/worker/internal/server/values"
)

const (
	maxXPPerLevelUnit   = 100
	entryBaseCoins      = 10
	entryCoinsPerLevel  = 15
	paidEntriesPerDay   = 5
	coinLimitReachedMsg = "Coin limit reached for this level today. Entries pay coin again tomorrow."
)

func successFraction(entry *entries.Entry) float64 {
	if entry.SuccessRate == nil {
		return 0.0
	}
	return max(0.0, min(1.0, *entry.SuccessRate/100.0))
}

func pythonRound(value float64) int {
	return int(math.RoundToEven(value))
}

func (api *API) requireEntry(r *http.Request) (*entries.Entry, error) {
	entry, err := entries.Get(r.Context(), api.deps.Store.Pool, r.PathValue("eval_id"))
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, httpx.BadRequest("Evaluation not found")
	}
	return entry, nil
}

func (api *API) get(w http.ResponseWriter, r *http.Request) error {
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	return api.respondWithEntry(w, r, entry.ID, http.StatusOK)
}

func (api *API) claimXP(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	if entry.StatusText() != "done" {
		return httpx.BadRequest("Evaluation is not finished yet")
	}
	maxXP := maxXPPerLevelUnit * entry.LevelNumber()
	if settings.IsPilotLevel(entry.Level()) {
		maxXP = pilotLevelMaxXP
	}
	rate := successFraction(entry)
	if entry.XPAwarded != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"xp": *entry.XPAwarded, "coins": intValue(entry.CoinsAwarded), "already_claimed": true,
			"max_xp": maxXP, "success_rate": values.RoundTo(rate*100.0, 1),
		})
		return nil
	}
	ctx := r.Context()
	others, err := entries.List(ctx, api.deps.Store.Pool, entries.SelectEntries(
		"player_evaluations.player_id = $1 AND player_evaluations.status IS DISTINCT FROM 'cancelled' AND player_evaluations.id <> $2"),
		entry.PlayerID, entry.ID)
	if err != nil {
		return err
	}
	xp := api.entryXP(entry, others, rate, maxXP)
	coins, coinNote := entryCoins(entry, others, rate)
	if _, err := api.deps.Store.Pool.Exec(ctx, "UPDATE player_evaluations SET xp_awarded = $2, coins_awarded = $3 WHERE id = $1", entry.ID, xp, coins); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"xp": xp, "coins": coins, "coin_note": coinNote, "already_claimed": false,
		"max_xp": maxXP, "success_rate": values.RoundTo(rate*100.0, 1),
	})
	return nil
}

func (api *API) entryXP(entry *entries.Entry, others []*entries.Entry, rate float64, maxXP int) int {
	levelNumber := entry.LevelNumber()
	earned, best, hasPrior := 0, 0.0, false
	for _, other := range others {
		if other.LevelNumber() != levelNumber || other.XPAwarded == nil {
			continue
		}
		hasPrior = true
		earned += *other.XPAwarded
		best = max(best, successFraction(other))
	}
	if !hasPrior {
		return pythonRound(rate * float64(maxXP))
	}
	if rate <= best {
		return 0
	}
	xp := pythonRound(float64(maxXP-earned) * 1.5 * (rate - best))
	return max(0, min(xp, maxXP-earned))
}

func entryCoins(entry *entries.Entry, others []*entries.Entry, rate float64) (int, string) {
	windowStart := pytime.Now().Add(-24 * time.Hour)
	paidToday := 0
	for _, other := range others {
		if other.LevelNumber() == entry.LevelNumber() && other.CoinsAwarded != nil && *other.CoinsAwarded > 0 &&
			other.CreatedAt != nil && !other.CreatedAt.Before(windowStart) {
			paidToday++
		}
	}
	if paidToday >= paidEntriesPerDay {
		return 0, coinLimitReachedMsg
	}
	return entryBaseCoins + pythonRound(entryCoinsPerLevel*float64(entry.LevelNumber())*rate), ""
}

func (api *API) deleteEntry(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	ctx := r.Context()
	err = api.deps.Store.InTransaction(ctx, func(transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, "DELETE FROM player_evaluations WHERE id = $1", entry.ID); err != nil {
			return err
		}
		return benchmarks.Release(ctx, transaction, entry.BenchmarkID)
	})
	if err != nil {
		return err
	}
	api.removeUnusedUploads(r, entry.ID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (api *API) removeUnusedUploads(r *http.Request, entryID string) {
	if err := benchmarks.RemoveUnusedUploads(r.Context(), api.deps.Store.Pool, api.deps.Files, entryID, ""); err != nil {
		api.deps.Logger.Printf("[WARNING] Could not clean up uploads for %s: %v", entryID, err)
	}
}

func (api *API) cancel(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	if status := entry.StatusText(); status != "queued" && status != "running" {
		return httpx.BadRequest("Evaluation is not running")
	}
	_, err = api.deps.Store.Pool.Exec(r.Context(), `UPDATE player_evaluations SET status = 'cancelled', error = 'cancelled',
		worker_id = NULL, stage = NULL, completed_at = $2 WHERE id = $1`, entry.ID, pytime.Now())
	if err != nil {
		return err
	}
	return api.respondWithEntry(w, r, entry.ID, http.StatusAccepted)
}

func (api *API) resimulate(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	if status := entry.StatusText(); status == "queued" || status == "running" {
		return httpx.BadRequest("Evaluation is already running")
	}
	ctx := r.Context()
	if settings.IsPilotLevel(entry.Level()) {
		pendingRun, err := entries.PendingRun(ctx, api.deps.Store.Pool, entry.ID)
		if err != nil {
			return err
		}
		if pendingRun == nil {
			return httpx.BadRequest("This piloted entry has no recorded run left to render")
		}
	}
	err = api.deps.Store.InTransaction(ctx, func(transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, `UPDATE player_evaluations SET completed_at = NULL, benchmark_id = NULL, job_config = NULL
			WHERE id = $1`, entry.ID); err != nil {
			return err
		}
		if err := entries.Queue(ctx, transaction, api.deps.Settings, entry); err != nil {
			return err
		}
		return benchmarks.Release(ctx, transaction, entry.BenchmarkID)
	})
	if err != nil {
		return err
	}
	api.removeUnusedUploads(r, entry.ID)
	return api.respondWithEntry(w, r, entry.ID, http.StatusAccepted)
}

func (api *API) deletePlayer(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	ctx := r.Context()
	playerID := r.PathValue("player_id")
	listed, err := entries.List(ctx, api.deps.Store.Pool, entries.SelectEntries("player_evaluations.player_id = $1"), playerID)
	if err != nil {
		return err
	}
	if len(listed) == 0 {
		return httpx.NotFound("Player not found")
	}
	err = api.deps.Store.InTransaction(ctx, func(transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, "DELETE FROM player_evaluations WHERE player_id = $1", playerID); err != nil {
			return err
		}
		released := map[string]bool{}
		for _, entry := range listed {
			if entry.BenchmarkID == nil || released[*entry.BenchmarkID] {
				continue
			}
			released[*entry.BenchmarkID] = true
			if err := benchmarks.Release(ctx, transaction, entry.BenchmarkID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, entry := range listed {
		api.removeUnusedUploads(r, entry.ID)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": len(listed)})
	return nil
}

func (api *API) renamePlayer(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	username := strings.TrimSpace(values.ToText(valueOr(data, "username", "")))
	if username == "" {
		return httpx.BadRequest("username is required")
	}
	if values.Length(username) > 30 {
		return httpx.BadRequest("username must be 30 characters or fewer")
	}
	playerID := r.PathValue("player_id")
	result, err := api.deps.Store.Pool.Exec(r.Context(), "UPDATE player_evaluations SET username = $2 WHERE player_id = $1", playerID, username)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return httpx.NotFound("Player not found")
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"player_id": playerID, "username": username, "updated": result.RowsAffected()})
	return nil
}

func valueOr(data map[string]any, key string, fallback any) any {
	if value, found := data[key]; found {
		return value
	}
	return fallback
}
