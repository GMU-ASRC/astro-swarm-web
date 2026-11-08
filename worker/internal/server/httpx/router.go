package httpx

import (
	"log"
	"net/http"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

type Router struct {
	mux    *http.ServeMux
	logger *log.Logger
}

func NewRouter(logger *log.Logger) *Router {
	router := &Router{mux: http.NewServeMux(), logger: logger}
	router.Handle("/api/", func(w http.ResponseWriter, r *http.Request) error {
		return NotFound("")
	})
	return router
}

func (router *Router) Handle(pattern string, handler HandlerFunc) {
	router.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			writeError(w, err, router.logger)
		}
	})
}

func (router *Router) HandleHTTP(pattern string, handler http.Handler) {
	router.mux.Handle(pattern, handler)
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.mux.ServeHTTP(w, r)
}

func (router *Router) Logger() *log.Logger {
	return router.logger
}
