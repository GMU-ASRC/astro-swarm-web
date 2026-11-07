from sqlalchemy import text

STATUS_RUNNING = "running"
STATUS_DONE = "done"
STATUS_FAILED = "failed"

CREATE_TABLE = """
CREATE TABLE IF NOT EXISTS schema_migrations (
    version integer PRIMARY KEY,
    name varchar(120) NOT NULL,
    status varchar(20) NOT NULL,
    started_at timestamptz,
    finished_at timestamptz,
    last_batch_key text,
    notes text
)
"""


def ensure_table(connection):
    connection.execute(text(CREATE_TABLE))
    connection.commit()


def load_all(connection):
    rows = connection.execute(
        text("SELECT version, name, status, started_at, finished_at, last_batch_key, notes FROM schema_migrations")
    ).mappings()
    recorded = {row["version"]: dict(row) for row in rows}
    connection.commit()
    return recorded


def mark_started(connection, migration):
    connection.execute(
        text(
            "INSERT INTO schema_migrations (version, name, status, started_at, finished_at, notes)"
            " VALUES (:version, :name, :status, now(), NULL, NULL)"
            " ON CONFLICT (version) DO UPDATE SET status = :status, name = :name,"
            " started_at = COALESCE(schema_migrations.started_at, now()), finished_at = NULL, notes = NULL"
        ),
        {"version": migration.version, "name": migration.name, "status": STATUS_RUNNING},
    )
    connection.commit()


def mark_finished(connection, migration, status, notes=None):
    connection.execute(
        text(
            "UPDATE schema_migrations SET status = :status, finished_at = now(), notes = :notes"
            " WHERE version = :version"
        ),
        {"version": migration.version, "status": status, "notes": notes},
    )
    connection.commit()


def save_batch_key(connection, version, batch_key):
    connection.execute(
        text("UPDATE schema_migrations SET last_batch_key = :batch_key WHERE version = :version"),
        {"version": version, "batch_key": batch_key},
    )
    connection.commit()


def load_batch_key(connection, version):
    batch_key = connection.execute(
        text("SELECT last_batch_key FROM schema_migrations WHERE version = :version"),
        {"version": version},
    ).scalar()
    connection.commit()
    return batch_key
