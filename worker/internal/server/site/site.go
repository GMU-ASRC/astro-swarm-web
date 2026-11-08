package site

import (
	"encoding/json"
	"html"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"astroswarm/worker/internal/server/deps"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/store"
)

const fallbackPage = "200.html"

type Site struct {
	deps *deps.Deps
}

func New(dependencies *deps.Deps) *Site {
	return &Site{deps: dependencies}
}

func (site *Site) Register(router *httpx.Router) {
	router.Handle("/", site.serve)
}

func isNoRows(err error) bool {
	return store.IsNoRows(err)
}

func decodeList(raw []byte) ([]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var list []any
	err := json.Unmarshal(raw, &list)
	return list, err
}

func (site *Site) clientFile(relative string) (string, bool) {
	cleaned := path.Clean("/" + relative)
	full := filepath.Join(site.deps.Config.ClientDir, filepath.FromSlash(cleaned))
	info, err := os.Stat(full)
	return full, err == nil && !info.IsDir()
}

func isProbePath(requestPath string) bool {
	segments := pathSegments(requestPath)
	if len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return strings.Contains(segments[len(segments)-1], ".")
}

func (site *Site) serve(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return httpx.NewError(http.StatusMethodNotAllowed, "The method is not allowed for the requested URL.")
	}
	requestPath := strings.TrimPrefix(r.URL.Path, "/")
	if strings.HasPrefix(requestPath, "api/") || strings.Contains(requestPath, "..") {
		return httpx.NotFound("")
	}
	candidates := []string{"index.html"}
	if requestPath != "" {
		candidates = []string{requestPath, requestPath + ".html", path.Join(requestPath, "index.html")}
	}
	for _, candidate := range candidates {
		full, found := site.clientFile(candidate)
		if !found {
			continue
		}
		if strings.HasSuffix(candidate, ".html") {
			return site.renderHTML(w, r, full, requestPath)
		}
		http.ServeFile(w, r, full)
		return nil
	}
	if isProbePath(requestPath) {
		return httpx.NotFound("")
	}
	if full, found := site.clientFile(fallbackPage); found {
		return site.renderHTML(w, r, full, requestPath)
	}
	return httpx.NotFound("")
}

func (site *Site) renderHTML(w http.ResponseWriter, r *http.Request, file string, requestPath string) error {
	page, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	meta, err := site.resolveMeta(r.Context(), r, requestPath)
	if err != nil {
		site.deps.Logger.Printf("[WARNING] Could not build page meta for /%s: %v", requestPath, err)
		meta = site.defaultMeta(r, requestPath)
	}
	text := string(page)
	for _, replacement := range [][2]string{
		{"__META_TITLE__", meta.title}, {"__META_DESC__", meta.description}, {"__META_TYPE__", meta.kind},
		{"__META_IMAGE__", meta.image}, {"__META_URL__", meta.url},
	} {
		text = strings.ReplaceAll(text, replacement[0], html.EscapeString(replacement[1]))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return nil
	}
	_, err = w.Write([]byte(text))
	return err
}
