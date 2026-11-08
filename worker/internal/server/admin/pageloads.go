package admin

import (
	"net/http"
	"regexp"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/values"
)

const (
	maxPageLoadBody = 2048
	maxPageLoadMs   = 120000.0
)

var routeIDPattern = regexp.MustCompile(`^/[A-Za-z0-9/_\[\]().-]{0,119}$`)

var pageLoadKinds = map[string]string{"first": "Full load", "navigation": "In-app"}

func (api *API) recordPageLoad(w http.ResponseWriter, r *http.Request) error {
	body, err := httpx.ReadBody(r, maxPageLoadBody)
	if err != nil {
		return httpx.BadRequest("Invalid page load report")
	}
	data, isObject := httpx.DecodeObject(body)
	if !isObject {
		return httpx.BadRequest("Invalid page load report")
	}
	route, _ := data["route"].(string)
	kindLabel, knownKind := pageLoadKinds[values.ToText(data["kind"])]
	totalMs, isNumber := values.Number(data["total_ms"])
	if !routeIDPattern.MatchString(route) || !knownKind || !isNumber || totalMs < 0 || totalMs > maxPageLoadMs {
		return httpx.BadRequest("Invalid page load report")
	}
	api.deps.Metrics.PageLoads.Record(kindLabel+" "+route, totalMs)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
