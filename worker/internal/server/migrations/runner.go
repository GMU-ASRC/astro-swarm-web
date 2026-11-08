package migrations

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"astroswarm/worker/internal/server/backup"
)

const advisoryLockKey = 734021117

type Options struct {
	DryRun         bool
	VerifyOnly     bool
	IncludeManual  bool
	OnlyVersion    int
	BackupDir      string
	UploadDir      string
}

type Runner struct {
	pool   *pgxpool.Pool
	logger *log.Logger
}

func NewRunner(pool *pgxpool.Pool, logger *log.Logger) *Runner {
	return &Runner{pool: pool, logger: logger}
}

func (runner *Runner) Run(ctx context.Context, options Options) error {
	pooled, err := runner.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer pooled.Release()
	connection := pooled.Conn()
	if _, err := connection.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return err
	}
	defer connection.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockKey)

	if err := ensureStateTable(ctx, connection); err != nil {
		return err
	}
	recorded, err := loadRecorded(ctx, connection)
	if err != nil {
		return err
	}
	if options.VerifyOnly {
		for _, migration := range All() {
			if recorded[migration.Version()].Status != statusDone {
				continue
			}
			run := &Context{Pool: runner.pool, Connection: connection, Migration: migration, Logger: runner.logger, UploadDir: options.UploadDir}
			if err := runner.verify(ctx, run); err != nil {
				return err
			}
			runner.logger.Printf("[INFO] Migration %03d %s verified", migration.Version(), migration.Name())
		}
		return nil
	}

	pending := 0
	for _, migration := range All() {
		if recorded[migration.Version()].Status == statusDone {
			continue
		}
		if options.OnlyVersion > 0 && migration.Version() != options.OnlyVersion {
			continue
		}
		if isManual(migration) && !options.IncludeManual {
			runner.logger.Printf("[INFO] Migration %03d %s waits for `astroserver migrate --manual`", migration.Version(), migration.Name())
			continue
		}
		pending++
		run := &Context{Pool: runner.pool, Connection: connection, Migration: migration, Logger: runner.logger, UploadDir: options.UploadDir}
		if options.DryRun {
			for _, line := range migration.Describe(ctx, run) {
				runner.logger.Printf("[INFO] [dry run %03d %s] %s", migration.Version(), migration.Name(), line)
			}
			continue
		}
		if err := runner.runOne(ctx, run, options); err != nil {
			return err
		}
	}
	if pending == 0 {
		runner.logger.Printf("[INFO] No pending migrations")
	}
	return nil
}

func (runner *Runner) runOne(ctx context.Context, run *Context, options Options) error {
	migration := run.Migration
	if err := runner.requireBackup(ctx, run, options.BackupDir); err != nil {
		return err
	}
	if err := markStarted(ctx, run.Connection, migration); err != nil {
		return err
	}
	runErr := migration.Run(ctx, run)
	if runErr == nil {
		runErr = runner.verify(ctx, run)
	}
	if runErr != nil {
		notes := runErr.Error()
		if len(notes) > 1000 {
			notes = notes[:1000]
		}
		markFinished(context.Background(), run.Connection, migration, statusFailed, &notes)
		return runErr
	}
	if err := markFinished(ctx, run.Connection, migration, statusDone, nil); err != nil {
		return err
	}
	runner.logger.Printf("[INFO] Migration %03d %s done", migration.Version(), migration.Name())
	return nil
}

func (runner *Runner) verify(ctx context.Context, run *Context) error {
	problems := run.Migration.Verify(ctx, run)
	if len(problems) == 0 {
		return nil
	}
	for index, problem := range problems {
		if index >= 50 {
			break
		}
		runner.logger.Printf("[ERROR] [%03d] %s", run.Migration.Version(), problem)
	}
	return fmt.Errorf("migration %03d %s failed verification with %d problem(s)", run.Migration.Version(), run.Migration.Name(), len(problems))
}

func (runner *Runner) requireBackup(ctx context.Context, run *Context, backupDir string) error {
	if !run.Migration.ChangesData() {
		return nil
	}
	hasData, err := databaseHasData(ctx, run)
	if err != nil || !hasData {
		return err
	}
	folder, found := backup.RecentVerified(backupDir)
	if !found {
		return fmt.Errorf(
			"migration %03d %s changes data and needs a verified backup from the last 24 hours. Run `astroserver backup` first",
			run.Migration.Version(), run.Migration.Name())
	}
	runner.logger.Printf("[INFO] Using backup %s", folder)
	return nil
}

func databaseHasData(ctx context.Context, run *Context) (bool, error) {
	exists, err := tableExists(ctx, run, "player_evaluations")
	if err != nil || !exists {
		return false, err
	}
	var hasRows bool
	err = run.Scalar(ctx, "SELECT EXISTS (SELECT 1 FROM player_evaluations)", &hasRows)
	return hasRows, err
}

func (runner *Runner) Status(ctx context.Context) ([]string, error) {
	pooled, err := runner.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer pooled.Release()
	if err := ensureStateTable(ctx, pooled.Conn()); err != nil {
		return nil, err
	}
	recorded, err := loadRecorded(ctx, pooled.Conn())
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for _, migration := range All() {
		row, found := recorded[migration.Version()]
		status, finished, notes := "pending", "", ""
		if found {
			status = row.Status
			if row.FinishedAt != nil {
				finished = row.FinishedAt.UTC().Format("2006-01-02T15:04:05.000000+00:00")
			}
			if row.Notes != nil {
				notes = *row.Notes
			}
		}
		lines = append(lines, fmt.Sprintf("%03d  %-32s  %-8s  %s  %s", migration.Version(), migration.Name(), status, finished, notes))
	}
	return lines, nil
}
