package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"astroswarm/worker/internal/server/app"
	"astroswarm/worker/internal/server/backup"
	"astroswarm/worker/internal/server/config"
	"astroswarm/worker/internal/server/migrations"
	"astroswarm/worker/internal/server/store"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	command := "serve"
	arguments := []string{}
	if len(os.Args) > 1 {
		command, arguments = os.Args[1], os.Args[2:]
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverConfig := config.Load()
	var err error
	switch command {
	case "serve":
		err = serve(ctx, serverConfig, logger)
	case "migrate":
		err = migrate(ctx, serverConfig, logger, arguments)
	case "backup":
		_, err = backup.Make(ctx, serverConfig.DatabaseURL, serverConfig.UploadDir, serverConfig.BackupDir, logger)
	default:
		err = fmt.Errorf("unknown command %q, expected serve, migrate or backup", command)
	}
	if err != nil {
		logger.Printf("[ERROR] %v", err)
		os.Exit(1)
	}
}

func serve(ctx context.Context, serverConfig config.Config, logger *log.Logger) error {
	database, err := store.Open(ctx, serverConfig.DatabaseURL, serverConfig.DatabasePoolSize)
	if err != nil {
		return err
	}
	defer database.Close()

	application, err := app.New(ctx, serverConfig, database, logger)
	if err != nil {
		return err
	}
	application.StartBackgroundTasks(ctx)

	httpServer := &http.Server{
		Addr:              ":" + serverConfig.Port,
		Handler:           application.Handler(),
		ReadHeaderTimeout: 30 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Printf("[INFO] Listening on :%s", serverConfig.Port)
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownContext)
	}
}

func migrate(ctx context.Context, serverConfig config.Config, logger *log.Logger, arguments []string) error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	dryRun := flags.Bool("dry-run", false, "describe pending migrations without writing anything")
	verifyOnly := flags.Bool("verify-only", false, "re-run verification for finished migrations")
	status := flags.Bool("status", false, "list every migration and its status")
	onlyVersion := flags.Int("only", 0, "run only this migration version")
	manual := flags.Bool("manual", false, "also run manual migrations: long conversions and cleanups that remove old data")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	database, err := store.Open(ctx, serverConfig.DatabaseURL, 4)
	if err != nil {
		return err
	}
	defer database.Close()

	runner := migrations.NewRunner(database.Pool, logger)
	if *status {
		lines, err := runner.Status(ctx)
		if err != nil {
			return err
		}
		for _, line := range lines {
			fmt.Println(line)
		}
		return nil
	}
	return runner.Run(ctx, migrations.Options{
		DryRun:        *dryRun,
		VerifyOnly:    *verifyOnly,
		IncludeManual: *manual,
		OnlyVersion:   *onlyVersion,
		BackupDir:     serverConfig.BackupDir,
		UploadDir:     serverConfig.UploadDir,
	})
}
