package httpx

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	if recorder.status == 0 {
		recorder.status = status
	}
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(data []byte) (int, error) {
	if recorder.status == 0 {
		recorder.status = http.StatusOK
	}
	return recorder.ResponseWriter.Write(data)
}

func (recorder *statusRecorder) Flush() {
	if flusher, canFlush := recorder.ResponseWriter.(http.Flusher); canFlush {
		flusher.Flush()
	}
}

func (recorder *statusRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}

type RequestObserver func(r *http.Request, status int, milliseconds float64)

func RequestLog(logger *log.Logger, observe RequestObserver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		elapsed := float64(time.Since(started).Microseconds()) / 1000.0
		if observe != nil {
			observe(r, status, elapsed)
		}
		if strings.HasPrefix(r.URL.Path, "/_app/") {
			return
		}
		logger.Printf("[INFO] %s %s %d - %.2fms", r.Method, r.URL.Path, status, elapsed)
	})
}

func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowAll := false
	allowed := map[string]bool{}
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		allowed[origin] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowAll || allowed[origin]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", "DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT")
				if requested := r.Header.Get("Access-Control-Request-Headers"); requested != "" {
					w.Header().Set("Access-Control-Allow-Headers", requested)
				}
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
