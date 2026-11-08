package app

import (
	"context"
	"log"
	"net/http"
	"os"

	"astroswarm/worker/internal/server/admin"
	"astroswarm/worker/internal/server/config"
	"astroswarm/worker/internal/server/deps"
	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/evaluations"
	"astroswarm/worker/internal/server/games"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/simulator"
	"astroswarm/worker/internal/server/site"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/workerapi"
)

type App struct {
	dependencies *deps.Deps
	router       *httpx.Router
	workers      *workerapi.API
	evaluations  *evaluations.API
	admin        *admin.API
	games        *games.API
	simulator    *simulator.API
	site         *site.Site
}

func New(ctx context.Context, serverConfig config.Config, database *store.Store, logger *log.Logger) (*App, error) {
	dependencies := deps.New(serverConfig, database, logger)
	application := &App{
		dependencies: dependencies,
		router:       httpx.NewRouter(logger),
		workers:      workerapi.New(dependencies),
		evaluations:  evaluations.New(dependencies),
		admin:        admin.New(dependencies),
		games:        games.New(dependencies),
		simulator:    simulator.New(dependencies),
		site:         site.New(dependencies),
	}
	application.registerRoutes()
	if err := application.ensureAdminUser(ctx); err != nil {
		logger.Printf("[WARNING] Could not create initial admin user: %v", err)
	}
	if err := application.recoverJobs(ctx); err != nil {
		logger.Printf("[WARNING] Could not recover evaluation jobs: %v", err)
	}
	return application, nil
}

func (application *App) registerRoutes() {
	application.router.Handle("GET /health", func(w http.ResponseWriter, r *http.Request) error {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
		return nil
	})
	application.workers.Register(application.router)
	application.evaluations.Register(application.router)
	application.admin.Register(application.router)
	application.games.Register(application.router)
	application.simulator.Register(application.router)
	application.site.Register(application.router)
}

func (application *App) Handler() http.Handler {
	logger := application.dependencies.Logger
	return httpx.RequestLog(logger, application.dependencies.Metrics.ObserveRequest, httpx.CORS(application.dependencies.Config.CORSOrigins, httpx.Gzip(application.router)))
}

func (application *App) StartBackgroundTasks(ctx context.Context) {
	go application.workers.Reaper().Run(ctx)
	go application.runRetireSweeps(ctx)
}

func (application *App) recoverJobs(ctx context.Context) error {
	pool := application.dependencies.Store.Pool
	pending, err := entries.List(ctx, pool, entries.SelectEntries("player_evaluations.status IN ('queued', 'running')"))
	if err != nil {
		return err
	}
	for _, entry := range pending {
		if err := entries.Queue(ctx, pool, application.dependencies.Settings, entry); err != nil {
			return err
		}
	}
	return nil
}

func (application *App) ensureAdminUser(ctx context.Context) error {
	dependencies := application.dependencies
	var userCount int
	if err := dependencies.Store.Pool.QueryRow(ctx, "SELECT count(*) FROM admin_users").Scan(&userCount); err != nil {
		return err
	}
	if userCount > 0 {
		return nil
	}
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = dependencies.Config.APISecretKey
	}
	if _, err := dependencies.Auth.CreateUser(ctx, username, password); err != nil {
		return err
	}
	dependencies.Logger.Printf("[INFO] Created initial admin user '%s'", username)
	return nil
}
