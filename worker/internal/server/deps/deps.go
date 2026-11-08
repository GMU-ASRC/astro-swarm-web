package deps

import (
	"log"

	"astroswarm/worker/internal/server/auth"
	"astroswarm/worker/internal/server/benchmarks"
	"astroswarm/worker/internal/server/config"
	"astroswarm/worker/internal/server/metrics"
	"astroswarm/worker/internal/server/settings"
	"astroswarm/worker/internal/server/store"
)

type Deps struct {
	Config   config.Config
	Store    *store.Store
	Auth     *auth.Service
	Settings *settings.Settings
	Files    *benchmarks.Files
	Builder  *benchmarks.Builder
	Logger   *log.Logger
	Metrics  *metrics.Recorder
}

func New(serverConfig config.Config, database *store.Store, logger *log.Logger) *Deps {
	files := benchmarks.NewFiles(serverConfig.UploadDir)
	return &Deps{
		Config:   serverConfig,
		Store:    database,
		Auth:     auth.New(database, serverConfig.APISecretKey),
		Settings: settings.New(database.Pool, serverConfig),
		Files:    files,
		Builder:  benchmarks.NewBuilder(files),
		Logger:   logger,
		Metrics:  metrics.NewRecorder(),
	}
}

func (dependencies *Deps) Replays() *benchmarks.Reader {
	return benchmarks.NewReader(dependencies.Store.Pool, dependencies.Files)
}
