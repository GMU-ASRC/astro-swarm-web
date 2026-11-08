package admin

import (
	"sync"

	"astroswarm/worker/internal/server/deps"
	"astroswarm/worker/internal/server/httpx"
)

type API struct {
	deps      *deps.Deps
	measuring sync.Mutex
}

func New(dependencies *deps.Deps) *API {
	return &API{deps: dependencies}
}

func (api *API) Register(router *httpx.Router) {
	router.Handle("POST /api/admin/login", api.login)
	router.Handle("POST /api/admin/logout", api.logout)
	router.Handle("GET /api/admin/me", api.me)
	router.Handle("POST /api/admin/password", api.changePassword)
	router.Handle("GET /api/admin/users", api.listUsers)
	router.Handle("POST /api/admin/users", api.createUser)
	router.Handle("DELETE /api/admin/users/{user_id}", api.deleteUser)
	router.Handle("GET /api/admin/stats", api.stats)
	router.Handle("POST /api/metrics/page-load", api.recordPageLoad)
	router.Handle("GET /api/admin/storage", api.storage)
	router.Handle("POST /api/admin/storage/reclaim", api.reclaim)
	router.Handle("GET /api/admin/export/info", api.exportInfo)
	router.Handle("GET /api/admin/export", api.export)
}
