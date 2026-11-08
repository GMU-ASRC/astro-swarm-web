package migrations

import (
	"context"
	"fmt"

	"astroswarm/worker/internal/server/benchmarks"
)

const (
	conversionBatchSize = 2000
	problemsShown       = 20
)

type binaryColumnsMigration struct{}

func (binaryColumnsMigration) Version() int      { return 6 }
func (binaryColumnsMigration) Name() string      { return "replay_binary_columns" }
func (binaryColumnsMigration) ChangesData() bool { return false }

func (binaryColumnsMigration) Describe(ctx context.Context, run *Context) []string {
	return []string{"Add benchmark_runs.binary_key and binary_bytes for replay files in the binary format"}
}

func (binaryColumnsMigration) Run(ctx context.Context, run *Context) error {
	if err := run.Execute(ctx, "ALTER TABLE benchmark_runs ADD COLUMN IF NOT EXISTS binary_key text, ADD COLUMN IF NOT EXISTS binary_bytes bigint"); err != nil {
		return err
	}
	return run.Execute(ctx, "CREATE INDEX IF NOT EXISTS ix_benchmark_runs_binary_key ON benchmark_runs (binary_key text_pattern_ops)")
}

func (binaryColumnsMigration) Verify(ctx context.Context, run *Context) []string {
	present, err := columnExists(ctx, run, "benchmark_runs", "binary_key")
	if err != nil {
		return []string{err.Error()}
	}
	if !present {
		return []string{"benchmark_runs.binary_key is missing"}
	}
	return nil
}

type convertReplaysMigration struct{}

func (convertReplaysMigration) Version() int      { return 7 }
func (convertReplaysMigration) Name() string      { return "convert_replay_files" }
func (convertReplaysMigration) ChangesData() bool { return true }
func (convertReplaysMigration) Manual() bool      { return true }

func (convertReplaysMigration) Describe(ctx context.Context, run *Context) []string {
	var pending int64
	run.Scalar(ctx, "SELECT count(*) FROM benchmark_runs WHERE binary_key IS NULL AND frames_key IS NOT NULL", &pending)
	return []string{
		fmt.Sprintf("Write a binary copy next to each of %d replay files; the old files are not changed", pending),
		"Each copy is read back and compared value by value before its row points at it",
		"Verification reads every old and new file again and compares every frame",
		"Runs only with `astroserver migrate --manual`; the site can stay up while it runs",
	}
}

func (convertReplaysMigration) Run(ctx context.Context, run *Context) error {
	files := benchmarks.NewFiles(run.UploadDir)
	summary, err := benchmarks.ConvertPendingRuns(ctx, run.Pool, files, conversionBatchSize, func(done int, summary benchmarks.ConvertSummary) {
		run.Log("Checked %d runs: %d converted, %d kept in the old format", done, summary.Converted, summary.Kept)
	})
	for index, problem := range summary.Problems {
		if index >= problemsShown {
			run.Log("... and %d more runs kept in the old format", len(summary.Problems)-problemsShown)
			break
		}
		run.Log("%s", problem)
	}
	return err
}

func (convertReplaysMigration) Verify(ctx context.Context, run *Context) []string {
	files := benchmarks.NewFiles(run.UploadDir)
	problems, err := benchmarks.VerifyConvertedRuns(ctx, run.Pool, files, conversionBatchSize, func(done int) {
		run.Log("Verified %d converted runs", done)
	})
	if err != nil {
		problems = append(problems, err.Error())
	}
	return problems
}

type retireLegacyReplaysMigration struct{}

func (retireLegacyReplaysMigration) Version() int      { return benchmarks.RetireLegacyFramesVersion }
func (retireLegacyReplaysMigration) Name() string      { return "retire_legacy_replay_files" }
func (retireLegacyReplaysMigration) ChangesData() bool { return true }
func (retireLegacyReplaysMigration) Manual() bool      { return true }

func (retireLegacyReplaysMigration) Describe(ctx context.Context, run *Context) []string {
	return []string{
		"Delete each old replay file whose binary copy matches it exactly, then clear its frames_key",
		"Runs that could not be converted keep their old file",
		"Afterwards the server retires the old file of every new result the same way, every 10 minutes",
		"Runs only with `astroserver migrate --manual`, after migration 7 is done and a fresh backup exists",
	}
}

func (retireLegacyReplaysMigration) Run(ctx context.Context, run *Context) error {
	var converted string
	if err := run.Scalar(ctx, "SELECT coalesce((SELECT status FROM schema_migrations WHERE version = 7), '')", &converted); err != nil {
		return err
	}
	if converted != statusDone {
		return fmt.Errorf("migration 007 convert_replay_files must be done first")
	}
	files := benchmarks.NewFiles(run.UploadDir)
	problems, err := benchmarks.RetireAllLegacy(ctx, run.Pool, files, conversionBatchSize, func(removed int) {
		run.Log("Removed %d old replay files", removed)
	})
	for index, problem := range problems {
		if index >= problemsShown {
			run.Log("... and %d more", len(problems)-problemsShown)
			break
		}
		run.Log("%s", problem)
	}
	return err
}

func (retireLegacyReplaysMigration) Verify(ctx context.Context, run *Context) []string {
	files := benchmarks.NewFiles(run.UploadDir)
	rows, err := run.Pool.Query(ctx, "SELECT id, binary_key FROM benchmark_runs WHERE binary_key IS NOT NULL AND frames_key IS NULL")
	if err != nil {
		return []string{err.Error()}
	}
	defer rows.Close()
	problems := []string{}
	for rows.Next() {
		var runID int64
		var binaryKey string
		if err := rows.Scan(&runID, &binaryKey); err != nil {
			return append(problems, err.Error())
		}
		if !files.Exists(binaryKey) {
			problems = append(problems, fmt.Sprintf("run %d has no old file and its binary file %s is missing", runID, binaryKey))
		}
	}
	if err := rows.Err(); err != nil {
		problems = append(problems, err.Error())
	}
	return problems
}
