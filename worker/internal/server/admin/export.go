package admin

import (
	"net/http"
	"sort"
	"strings"

	"astroswarm/worker/internal/server/backup"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
)

var exportParts = []string{"database", "tables", "files"}

func isNoRows(err error) bool {
	return store.IsNoRows(err)
}

func requestedParts(r *http.Request) (map[string]bool, error) {
	raw := "database,tables,files"
	if query := r.URL.Query(); query.Has("parts") {
		raw = query.Get("parts")
	}
	parts := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts[trimmed] = true
		}
	}
	unknown := []string{}
	for part := range parts {
		known := false
		for _, name := range exportParts {
			known = known || name == part
		}
		if !known {
			unknown = append(unknown, part)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, httpx.BadRequest("Unknown export parts: " + strings.Join(unknown, ", "))
	}
	if len(parts) == 0 {
		return nil, httpx.BadRequest("Choose at least one of: " + strings.Join(exportParts, ", "))
	}
	return parts, nil
}

func (api *API) exportInfo(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"pg_dump": backup.PgDumpAvailable(), "parts": exportParts})
	return nil
}

func (api *API) export(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	parts, err := requestedParts(r)
	if err != nil {
		return err
	}
	if parts["database"] && len(parts) == 1 && !backup.PgDumpAvailable() {
		return httpx.BadRequest("pg_dump is not installed on the server, so a database dump cannot be made")
	}
	stamp := pytime.Now().Format("20060102-150405")
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="astroswarm-`+stamp+`.zip"`)
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	config := api.deps.Config
	if err := backup.StreamExport(r.Context(), w, api.deps.Store.Pool, config.DatabaseURL, config.UploadDir,
		parts["database"], parts["tables"], parts["files"]); err != nil {
		api.deps.Logger.Printf("[WARNING] Export stopped: %v", err)
	}
	return nil
}
