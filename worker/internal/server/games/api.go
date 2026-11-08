package games

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"astroswarm/worker/internal/server/deps"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/values"
)

type API struct {
	deps    *deps.Deps
	version versionCache
}

type versionCache struct {
	lock      sync.Mutex
	data      map[string]any
	fetchedAt float64
}

func New(dependencies *deps.Deps) *API {
	return &API{deps: dependencies}
}

func (api *API) Register(router *httpx.Router) {
	router.Handle("GET /api/version", api.getVersion)
	router.Handle("GET /api/leaderboard", api.listLeaderboard)
	router.Handle("POST /api/leaderboard", api.submitLeaderboard)
	router.Handle("GET /api/leaderboard/{entry_id}", api.getLeaderboardEntry)
	router.Handle("DELETE /api/leaderboard/{entry_id}", api.deleteLeaderboardEntry)
	router.Handle("GET /api/leaderboard/{entry_id}/export", api.exportLeaderboardEntry)
	router.Handle("GET /api/survive/matches", api.listMatches)
	router.Handle("POST /api/survive/matches", api.submitMatch)
	router.Handle("GET /api/survive/matches/{match_id}", api.getMatch)
	router.Handle("DELETE /api/survive/matches/{match_id}", api.deleteMatch)
	router.Handle("GET /api/training/leaderboard", api.trainingLeaderboard)
	router.Handle("GET /api/training/scores/{player_id}", api.getTrainingScore)
	router.Handle("POST /api/training/scores", api.submitTrainingScore)
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

func listOrEmpty(raw []byte) any {
	decoded := decodeJSON(raw)
	if !values.Truthy(decoded) {
		return []any{}
	}
	return decoded
}

func intOr(value *int, fallback int) int {
	if value == nil || *value == 0 {
		return fallback
	}
	return *value
}

func stringOr(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func limitQuery(r *http.Request, fallback int, ceiling int) int {
	text := r.URL.Query().Get("limit")
	if text == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return fallback
	}
	return max(1, min(ceiling, parsed))
}
