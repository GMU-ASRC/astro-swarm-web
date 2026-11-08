package admin

import (
	"context"
	"net/http"
	"os"
	"slices"
	"sort"
	"time"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
)

const maxReclaimBytes = 1024 * 1024 * 1024

var trackedTables = []string{
	"player_evaluations", "benchmarks", "benchmark_runs", "sim_runs", "sim_configs", "survive_matches",
	"leaderboard_entries", "workers", "admin_users", "admin_sessions", "app_settings", "simulator_entries",
	"training_scores", "player_evaluations_archive",
}

func (api *API) tableSize(ctx context.Context, name string) (int64, error) {
	var size *int64
	err := api.deps.Store.Pool.QueryRow(ctx, "SELECT pg_total_relation_size($1::regclass)", name).Scan(&size)
	if err != nil || size == nil {
		return 0, err
	}
	return *size, nil
}

func (api *API) tableStats(ctx context.Context, name string, table map[string]any) error {
	var live, dead *int64
	var lastVacuum, lastAutovacuum *time.Time
	err := api.deps.Store.Pool.QueryRow(ctx,
		"SELECT n_live_tup, n_dead_tup, last_vacuum, last_autovacuum FROM pg_stat_user_tables WHERE relname = $1", name).
		Scan(&live, &dead, &lastVacuum, &lastAutovacuum)
	if err != nil && !isNoRows(err) {
		return err
	}
	if isNoRows(err) {
		table["rows"] = 0
		table["dead_rows"] = 0
		table["last_vacuum"] = nil
		return nil
	}
	var latest *time.Time
	for _, stamp := range []*time.Time{lastVacuum, lastAutovacuum} {
		if stamp != nil && (latest == nil || stamp.After(*latest)) {
			latest = stamp
		}
	}
	table["rows"] = valueOrZero(live)
	table["live_rows"] = valueOrZero(live)
	table["dead_rows"] = valueOrZero(dead)
	table["last_vacuum"] = pytime.IsoformatOrNil(latest)
	return nil
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func (api *API) databaseSize(ctx context.Context) (int64, []map[string]any, error) {
	var total *int64
	if err := api.deps.Store.Pool.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&total); err != nil {
		return 0, nil, err
	}
	tables := []map[string]any{}
	for _, name := range trackedTables {
		size, err := api.tableSize(ctx, name)
		if err != nil {
			continue
		}
		table := map[string]any{"name": name, "bytes": size, "reclaimable": size <= maxReclaimBytes}
		if err := api.tableStats(ctx, name, table); err != nil {
			continue
		}
		tables = append(tables, table)
	}
	sort.SliceStable(tables, func(left, right int) bool {
		return tables[left]["bytes"].(int64) > tables[right]["bytes"].(int64)
	})
	return valueOrZero(total), tables, nil
}

func (api *API) storage(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	ctx := r.Context()
	var databaseError any
	databaseBytes, tables, err := api.databaseSize(ctx)
	if err != nil {
		databaseBytes, tables, databaseError = 0, []map[string]any{}, err.Error()
	}
	uploadDir := api.deps.Config.UploadDir
	var uploadsError any
	if info, err := os.Stat(uploadDir); err != nil || !info.IsDir() {
		uploadsError = "Upload directory not found"
	}
	force := slices.Contains([]string{"1", "true", "yes"}, r.URL.Query().Get("measure"))
	measuring, err := api.refreshUploadsIfStale(ctx, force)
	if err != nil {
		return err
	}
	uploads, err := api.cachedUploadsUsage(ctx)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"total_bytes": databaseBytes + uploads.bytes, "database_bytes": databaseBytes, "database_error": databaseError,
		"tables": tables, "uploads_bytes": uploads.bytes, "upload_files": uploads.files,
		"uploads_measured_at": uploads.measuredAt, "uploads_measuring": measuring, "upload_dir": uploadDir,
		"uploads_error": uploadsError,
	})
	return nil
}

func (api *API) reclaim(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	name := "player_evaluations"
	if _, found := data["table"]; found {
		name = textField(data, "table")
	}
	if !slices.Contains(trackedTables, name) {
		return httpx.BadRequest("Unknown table")
	}
	ctx := r.Context()
	before, err := api.tableSize(ctx, name)
	if err != nil {
		return err
	}
	if before > maxReclaimBytes {
		return httpx.BadRequest(name + " is larger than 1 GB. A full rewrite would lock it and take the site down while it runs, " +
			"so let autovacuum reclaim the space instead.")
	}
	if _, err := api.deps.Store.Pool.Exec(ctx, "VACUUM (FULL, ANALYZE) "+name); err != nil {
		return httpx.BadRequest("Reclaim failed: " + err.Error())
	}
	after, err := api.tableSize(ctx, name)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"table": name, "before_bytes": before, "after_bytes": after, "freed_bytes": max(0, before-after),
	})
	return nil
}
