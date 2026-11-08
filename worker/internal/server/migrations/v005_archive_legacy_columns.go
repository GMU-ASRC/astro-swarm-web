package migrations

import (
	"context"
	"fmt"
)

const (
	archiveTable     = "player_evaluations_archive"
	archiveBatchSize = 10
)

type archiveLegacyColumnsMigration struct{}

func (archiveLegacyColumnsMigration) Version() int      { return 5 }
func (archiveLegacyColumnsMigration) Name() string      { return "archive_legacy_columns" }
func (archiveLegacyColumnsMigration) ChangesData() bool { return true }
func (archiveLegacyColumnsMigration) Manual() bool      { return true }

func (archiveLegacyColumnsMigration) Describe(ctx context.Context, run *Context) []string {
	return []string{
		"Copy player_evaluations.results and .replays into " + archiveTable + ", " + fmt.Sprint(archiveBatchSize) + " rows at a time",
		"Compare every archived value with the original (md5 of the stored text)",
		"Drop results and replays from player_evaluations, then rewrite the table with VACUUM FULL to give the space back",
		"Runs only with `astroserver migrate --manual`, after migration 4 is done",
	}
}

func legacyColumnsPresent(ctx context.Context, run *Context) (bool, error) {
	hasResults, err := columnExists(ctx, run, "player_evaluations", "results")
	if err != nil {
		return false, err
	}
	hasReplays, err := columnExists(ctx, run, "player_evaluations", "replays")
	return hasResults && hasReplays, err
}

func (migration archiveLegacyColumnsMigration) Run(ctx context.Context, run *Context) error {
	var copyDone string
	if err := run.Scalar(ctx, "SELECT coalesce((SELECT status FROM schema_migrations WHERE version = 4), '')", &copyDone); err != nil {
		return err
	}
	if copyDone != statusDone {
		return fmt.Errorf("migration 004 copy_to_benchmarks must be done before the old columns are archived")
	}
	present, err := legacyColumnsPresent(ctx, run)
	if err != nil || !present {
		return err
	}
	if err := run.Execute(ctx, `CREATE TABLE IF NOT EXISTS `+archiveTable+` (
		id varchar PRIMARY KEY,
		results json,
		replays json,
		archived_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}
	if err := migration.copyRows(ctx, run); err != nil {
		return err
	}
	if problems := compareArchive(ctx, run); len(problems) > 0 {
		return fmt.Errorf("archive does not match the original columns: %s", problems[0])
	}
	run.Log("Archive matches every row, dropping the old columns")
	if err := run.Execute(ctx, "ALTER TABLE player_evaluations DROP COLUMN results, DROP COLUMN replays"); err != nil {
		return err
	}
	run.Log("Rewriting player_evaluations to give the space back")
	return run.ExecuteOutsideTransaction(ctx, "VACUUM (FULL, ANALYZE) player_evaluations")
}

func (archiveLegacyColumnsMigration) copyRows(ctx context.Context, run *Context) error {
	var total int64
	if err := run.Scalar(ctx, "SELECT count(*) FROM player_evaluations", &total); err != nil {
		return err
	}
	lastID, copied := "", int64(0)
	for {
		rows, err := run.Connection.Query(ctx, `INSERT INTO `+archiveTable+` (id, results, replays)
			SELECT id, results, replays FROM player_evaluations WHERE id > $1 ORDER BY id LIMIT $2
			ON CONFLICT (id) DO NOTHING RETURNING id`, lastID, archiveBatchSize)
		if err != nil {
			return err
		}
		rows.Close()
		var nextID *string
		if err := run.Scalar(ctx, "SELECT max(id) FROM (SELECT id FROM player_evaluations WHERE id > $1 ORDER BY id LIMIT $2) batch",
			&nextID, lastID, archiveBatchSize); err != nil {
			return err
		}
		if nextID == nil {
			break
		}
		lastID = *nextID
		copied += archiveBatchSize
		run.Log("Archived %d of %d rows", min(copied, total), total)
	}
	return nil
}

func compareArchive(ctx context.Context, run *Context) []string {
	rows, err := run.Connection.Query(ctx, `SELECT e.id FROM player_evaluations e
		LEFT JOIN `+archiveTable+` a ON a.id = e.id
		WHERE a.id IS NULL
			OR md5(e.results::text) IS DISTINCT FROM md5(a.results::text)
			OR md5(e.replays::text) IS DISTINCT FROM md5(a.replays::text)
		LIMIT 50`)
	if err != nil {
		return []string{err.Error()}
	}
	defer rows.Close()
	problems := []string{}
	for rows.Next() {
		var entryID string
		if err := rows.Scan(&entryID); err != nil {
			return append(problems, err.Error())
		}
		problems = append(problems, "entry "+entryID+" is missing from the archive or differs from it")
	}
	if err := rows.Err(); err != nil {
		problems = append(problems, err.Error())
	}
	return problems
}

func (archiveLegacyColumnsMigration) Verify(ctx context.Context, run *Context) []string {
	exists, err := tableExists(ctx, run, archiveTable)
	if err != nil {
		return []string{err.Error()}
	}
	present, err := legacyColumnsPresent(ctx, run)
	if err != nil {
		return []string{err.Error()}
	}
	if present {
		return compareArchive(ctx, run)
	}
	if !exists {
		return []string{archiveTable + " is missing although the old columns were dropped"}
	}
	return nil
}
