package admin

import (
	"context"
	"net/http"
	"time"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
)

const workerOnlineSeconds = 30

type statsQuery struct {
	ctx  context.Context
	api  *API
	errs []error
}

func (query *statsQuery) number(sql string, arguments ...any) int64 {
	var value *int64
	if err := query.api.deps.Store.Pool.QueryRow(query.ctx, sql, arguments...).Scan(&value); err != nil {
		query.errs = append(query.errs, err)
		return 0
	}
	if value == nil {
		return 0
	}
	return *value
}

func (query *statsQuery) decimal(sql string, arguments ...any) any {
	var value *float64
	if err := query.api.deps.Store.Pool.QueryRow(query.ctx, sql, arguments...).Scan(&value); err != nil {
		query.errs = append(query.errs, err)
		return nil
	}
	if value == nil {
		return nil
	}
	return float64(int64(*value*10+0.5)) / 10
}

func (api *API) stats(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	query := &statsQuery{ctx: r.Context(), api: api}
	now := pytime.Now()
	dayAgo, weekAgo := now.Add(-24*time.Hour), now.Add(-7*24*time.Hour)
	entries := map[string]any{
		"total":           query.number("SELECT count(*) FROM player_evaluations"),
		"last_24h":        query.number("SELECT count(*) FROM player_evaluations WHERE created_at >= $1", dayAgo),
		"last_7d":         query.number("SELECT count(*) FROM player_evaluations WHERE created_at >= $1", weekAgo),
		"players":         query.number("SELECT count(DISTINCT player_id) FROM player_evaluations"),
		"active_players":  query.number("SELECT count(DISTINCT player_id) FROM player_evaluations WHERE created_at >= $1", weekAgo),
		"benchmarks":      query.number("SELECT count(*) FROM benchmarks"),
		"shared_results":  query.number("SELECT count(*) - count(DISTINCT benchmark_id) FROM player_evaluations WHERE benchmark_id IS NOT NULL"),
		"average_success": query.decimal("SELECT avg(b.success_rate) FROM player_evaluations e JOIN benchmarks b ON b.id = e.benchmark_id WHERE e.status = 'done'"),
		"turnaround_s_7d": query.decimal("SELECT avg(extract(epoch FROM completed_at - created_at)) FROM player_evaluations WHERE status = 'done' AND completed_at >= $1 AND completed_at > created_at", weekAgo),
		"benchmark_runs":  query.number("SELECT greatest(reltuples, 0)::bigint FROM pg_class WHERE relname = 'benchmark_runs'"),
	}
	statuses, err := api.statusCounts(r.Context())
	if err != nil {
		return err
	}
	entries["by_status"] = statuses
	daily, err := api.dailySubmissions(r.Context(), now)
	if err != nil {
		return err
	}
	levels, err := api.levelStats(r.Context())
	if err != nil {
		return err
	}
	onlineSince := now.Add(-workerOnlineSeconds * time.Second)
	workers := map[string]any{
		"total":   query.number("SELECT count(*) FROM workers"),
		"enabled": query.number("SELECT count(*) FROM workers WHERE enabled IS NOT FALSE"),
		"online":  query.number("SELECT count(*) FROM workers WHERE last_seen >= $1", onlineSince),
		"busy":    query.number("SELECT count(*) FROM workers WHERE last_seen >= $1 AND current_job_id IS NOT NULL", onlineSince),
	}
	content := map[string]any{
		"simulator_entries":   query.number("SELECT count(*) FROM simulator_entries"),
		"survive_matches":     query.number("SELECT count(*) FROM survive_matches"),
		"training_scores":     query.number("SELECT count(*) FROM training_scores"),
		"leaderboard_entries": query.number("SELECT count(*) FROM leaderboard_entries"),
		"sim_runs":            query.number("SELECT count(*) FROM sim_runs"),
		"admin_users":         query.number("SELECT count(*) FROM admin_users"),
	}
	if len(query.errs) > 0 {
		return query.errs[0]
	}
	recorder := api.deps.Metrics
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"generated_at": pytime.Isoformat(now), "entries": entries, "daily": daily, "levels": levels,
		"workers": workers, "content": content,
		"timings": map[string]any{
			"since":      pytime.Isoformat(recorder.Requests.Since().UTC()),
			"requests":   recorder.Requests.Summaries(),
			"page_loads": recorder.PageLoads.Summaries(),
		},
	})
	return nil
}

func (api *API) statusCounts(ctx context.Context) (map[string]int64, error) {
	rows, err := api.deps.Store.Pool.Query(ctx, "SELECT coalesce(status, 'unknown'), count(*) FROM player_evaluations GROUP BY 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (api *API) dailySubmissions(ctx context.Context, now time.Time) ([]any, error) {
	rows, err := api.deps.Store.Pool.Query(ctx, `SELECT day::date::text,
			count(e.id), count(e.id) FILTER (WHERE e.status = 'done'), count(DISTINCT e.player_id)
		FROM generate_series(($1::timestamptz AT TIME ZONE 'UTC')::date - 13, ($1::timestamptz AT TIME ZONE 'UTC')::date, interval '1 day') AS day
		LEFT JOIN player_evaluations e ON (e.created_at AT TIME ZONE 'UTC')::date = day::date
		GROUP BY day ORDER BY day`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	days := []any{}
	for rows.Next() {
		var day string
		var submitted, done, players int64
		if err := rows.Scan(&day, &submitted, &done, &players); err != nil {
			return nil, err
		}
		days = append(days, map[string]any{"date": day, "submitted": submitted, "done": done, "players": players})
	}
	return days, rows.Err()
}

func (api *API) levelStats(ctx context.Context) ([]any, error) {
	rows, err := api.deps.Store.Pool.Query(ctx, `SELECT coalesce(nullif(e.level_id, ''), 'farp'), count(*), count(DISTINCT e.player_id),
			count(*) FILTER (WHERE e.status = 'done'), max(b.success_rate), avg(b.success_rate), max(e.created_at)
		FROM player_evaluations e LEFT JOIN benchmarks b ON b.id = e.benchmark_id
		WHERE e.status IS DISTINCT FROM 'cancelled'
		GROUP BY 1 ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	levels := []any{}
	for rows.Next() {
		var levelID string
		var entries, players, done int64
		var best, average *float64
		var latest *time.Time
		if err := rows.Scan(&levelID, &entries, &players, &done, &best, &average, &latest); err != nil {
			return nil, err
		}
		levels = append(levels, map[string]any{
			"level_id": levelID, "entries": entries, "players": players, "done": done,
			"best_rate": roundedOrNil(best), "average_rate": roundedOrNil(average), "last_entry": pytime.IsoformatOrNil(latest),
		})
	}
	return levels, rows.Err()
}

func roundedOrNil(value *float64) any {
	if value == nil {
		return nil
	}
	return float64(int64(*value*10+0.5)) / 10
}
