package migrations

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	statusRunning = "running"
	statusDone    = "done"
	statusFailed  = "failed"
)

const createStateTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version integer PRIMARY KEY,
    name varchar(120) NOT NULL,
    status varchar(20) NOT NULL,
    started_at timestamptz,
    finished_at timestamptz,
    last_batch_key text,
    notes text
)`

type recordedMigration struct {
	Version    int
	Name       string
	Status     string
	FinishedAt *time.Time
	Notes      *string
}

func ensureStateTable(ctx context.Context, connection *pgx.Conn) error {
	_, err := connection.Exec(ctx, createStateTable)
	return err
}

func loadRecorded(ctx context.Context, connection *pgx.Conn) (map[int]recordedMigration, error) {
	rows, err := connection.Query(ctx, "SELECT version, name, status, finished_at, notes FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recorded := map[int]recordedMigration{}
	for rows.Next() {
		var row recordedMigration
		if err := rows.Scan(&row.Version, &row.Name, &row.Status, &row.FinishedAt, &row.Notes); err != nil {
			return nil, err
		}
		recorded[row.Version] = row
	}
	return recorded, rows.Err()
}

func markStarted(ctx context.Context, connection *pgx.Conn, migration Migration) error {
	_, err := connection.Exec(ctx, `
		INSERT INTO schema_migrations (version, name, status, started_at, finished_at, notes)
		VALUES ($1, $2, $3, now(), NULL, NULL)
		ON CONFLICT (version) DO UPDATE SET status = $3, name = $2,
			started_at = COALESCE(schema_migrations.started_at, now()), finished_at = NULL, notes = NULL`,
		migration.Version(), migration.Name(), statusRunning)
	return err
}

func markFinished(ctx context.Context, connection *pgx.Conn, migration Migration, status string, notes *string) error {
	_, err := connection.Exec(ctx,
		"UPDATE schema_migrations SET status = $2, finished_at = now(), notes = $3 WHERE version = $1",
		migration.Version(), status, notes)
	return err
}
