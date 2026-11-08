package metrics

import (
	"net/http"
	"strings"
)

type Recorder struct {
	Requests  *Timings
	PageLoads *Timings
}

func NewRecorder() *Recorder {
	return &Recorder{Requests: NewTimings(), PageLoads: NewTimings()}
}

func requestKey(r *http.Request) string {
	pattern := r.Pattern
	if pattern == "" || pattern == "/" {
		segment, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if strings.Contains(segment, ".") {
			return "GET static files"
		}
		return "GET page /" + segment
	}
	if !strings.Contains(pattern, " ") {
		return r.Method + " " + pattern
	}
	return pattern
}

func (recorder *Recorder) ObserveRequest(r *http.Request, status int, milliseconds float64) {
	if strings.HasPrefix(r.URL.Path, "/_app/") || status == http.StatusNotFound {
		return
	}
	recorder.Requests.Record(requestKey(r), milliseconds)
}
