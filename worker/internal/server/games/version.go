package games

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"astroswarm/worker/internal/server/httpx"
)

const (
	releasesURL     = "https://api.github.com/repos/GMU-ASRC/astro-swarm/releases"
	versionCacheTTL = 300.0
)

func (api *API) getVersion(w http.ResponseWriter, r *http.Request) error {
	api.version.lock.Lock()
	defer api.version.lock.Unlock()
	now := float64(time.Now().UnixNano()) / 1e9
	if api.version.data != nil && now-api.version.fetchedAt < versionCacheTTL {
		httpx.WriteJSON(w, http.StatusOK, api.version.data)
		return nil
	}
	tag, status, err := fetchLatestTag(r.Context())
	if status == http.StatusNotFound {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"error": "No releases found"})
		return nil
	}
	if err != nil {
		api.deps.Logger.Printf("[ERROR] Failed to fetch version from GitHub: %v", err)
		if api.version.data != nil {
			httpx.WriteJSON(w, http.StatusOK, api.version.data)
			return nil
		}
		httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Could not fetch release info"})
		return nil
	}
	api.version.data = map[string]any{"version": tag}
	api.version.fetchedAt = now
	httpx.WriteJSON(w, http.StatusOK, api.version.data)
	return nil
}

func fetchLatestTag(ctx context.Context) (string, int, error) {
	requestContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, releasesURL, nil)
	if err != nil {
		return "", 0, err
	}
	request.Header.Set("User-Agent", "AstroSwarm-Web")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", 0, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return "", http.StatusNotFound, nil
	}
	if response.StatusCode >= 400 {
		return "", response.StatusCode, &httpx.Error{Status: response.StatusCode, Message: response.Status}
	}
	var releases []map[string]any
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return "", 0, err
	}
	if len(releases) == 0 {
		return "", http.StatusNotFound, nil
	}
	tag, _ := releases[0]["tag_name"].(string)
	return tag, http.StatusOK, nil
}
