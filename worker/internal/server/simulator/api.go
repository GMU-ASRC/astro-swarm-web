package simulator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"astroswarm/worker/internal/server/deps"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/values"
)

type API struct {
	deps *deps.Deps
}

func New(dependencies *deps.Deps) *API {
	return &API{deps: dependencies}
}

func (api *API) Register(router *httpx.Router) {
	router.Handle("GET /api/simulator/entries", api.listEntries)
	router.Handle("POST /api/simulator/entries", api.submitEntry)
	router.Handle("GET /api/simulator/entries/{entry_id}", api.getEntry)
	router.Handle("DELETE /api/simulator/entries/{entry_id}", api.deleteEntry)
	router.Handle("GET /api/simulator/entries/{entry_id}/replay", api.getEntryReplay)
	router.Handle("GET /api/runs", api.listRuns)
	router.Handle("POST /api/runs", api.uploadRun)
	router.Handle("GET /api/runs/{run_id}", api.getRun)
	router.Handle("DELETE /api/runs/{run_id}", api.deleteRun)
	router.Handle("GET /api/runs/{run_id}/export", api.exportRun)
	router.Handle("GET /api/runs/{run_id}/download", api.downloadRun)
	router.Handle("GET /api/runs/{run_id}/thumbnail", api.runThumbnail)
	router.Handle("GET /api/runs/{run_id}/video", api.runVideo)
	router.Handle("GET /api/runs/{run_id}/config", api.runConfig)
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func decodeJSON(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if decoder.Decode(&decoded) != nil {
		return nil
	}
	return decoded
}

func jsonOr(raw []byte, fallback any) any {
	decoded := decodeJSON(raw)
	if !values.Truthy(decoded) {
		return fallback
	}
	return decoded
}

func queryInt(r *http.Request, key string, fallback int) int {
	parsed, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return parsed
}
