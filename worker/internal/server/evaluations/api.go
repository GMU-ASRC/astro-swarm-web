package evaluations

import (
	"net/http"

	"astroswarm/worker/internal/server/deps"
	"astroswarm/worker/internal/server/httpx"
)

type API struct {
	deps *deps.Deps
}

func New(dependencies *deps.Deps) *API {
	return &API{deps: dependencies}
}

func (api *API) Register(router *httpx.Router) {
	router.Handle("GET /api/evaluations", api.list)
	router.Handle("POST /api/evaluations", api.submit)
	router.Handle("POST /api/evaluations/run", api.submitRun)
	router.Handle("GET /api/evaluations/best", api.best)
	router.Handle("GET /api/evaluations/players", api.players)
	router.Handle("GET /api/evaluations/level-leaderboard", api.levelLeaderboard)
	router.Handle("GET /api/evaluations/level-sweep", api.levelSweep)
	router.Handle("GET /api/evaluations/baseline", api.baseline)
	router.Handle("GET /api/evaluations/settings", api.getSettings)
	router.Handle("PUT /api/evaluations/settings", api.updateSettings)
	router.Handle("DELETE /api/evaluations/players/{player_id}", api.deletePlayer)
	router.Handle("PUT /api/evaluations/players/{player_id}", api.renamePlayer)
	router.Handle("GET /api/evaluations/{eval_id}", api.get)
	router.Handle("DELETE /api/evaluations/{eval_id}", api.deleteEntry)
	router.Handle("POST /api/evaluations/{eval_id}/claim-xp", api.claimXP)
	router.Handle("POST /api/evaluations/{eval_id}/cancel", api.cancel)
	router.Handle("POST /api/evaluations/{eval_id}/resimulate", api.resimulate)
	router.Handle("GET /api/evaluations/{first}/{second}", api.twoPartGet)
	router.Handle("GET /api/evaluations/{eval_id}/chart/{chart_file}", api.chart)
	router.Handle("GET /api/evaluations/{eval_id}/replay/{trial}", api.replay)
	router.Handle("GET /api/replay-frames/{run_id}", api.replayFrames)
	router.Handle("GET /api/evaluations/{eval_id}/sweep-replay/{n}", api.sweepReplay)
	router.Handle("GET /api/evaluations/{eval_id}/sweep-replay/{n}/trials", api.sweepTrialReplays)
	router.Handle("GET /api/evaluations/{eval_id}/sweep-replay/{n}/trial/{trial}", api.sweepTrialReplay)
}

func (api *API) twoPartGet(w http.ResponseWriter, r *http.Request) error {
	first, second := r.PathValue("first"), r.PathValue("second")
	if first == "players" {
		r.SetPathValue("player_id", second)
		return api.profile(w, r)
	}
	r.SetPathValue("eval_id", first)
	switch second {
	case "replays":
		return api.replays(w, r)
	case "sweep-replays":
		return api.sweepReplays(w, r)
	case "export":
		return api.export(w, r)
	case "thumbnail.png":
		return api.thumbnail(w, r)
	}
	return httpx.NotFound("")
}
