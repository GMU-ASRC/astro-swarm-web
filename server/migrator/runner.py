from sqlalchemy import text

from migrator import backups, state
from migrator.context import MigrationContext
from migrator.registry import MIGRATIONS

ADVISORY_LOCK_KEY = 734021117


class MigrationError(RuntimeError):
    pass


def _database_has_data(connection):
    exists = connection.execute(text("SELECT to_regclass('public.player_evaluations') IS NOT NULL")).scalar()
    has_rows = False
    if exists:
        has_rows = connection.execute(text("SELECT EXISTS (SELECT 1 FROM player_evaluations)")).scalar()
    connection.commit()
    return has_rows


def _require_backup(connection, migration, logger):
    if not migration.changes_data:
        return
    if not _database_has_data(connection):
        return
    recent = backups.recent_verified_backup()
    if recent is None:
        raise MigrationError(
            "Migration %03d %s changes data and needs a verified backup from the last 24 hours. "
            "Run `python make_backup.py` first." % (migration.version, migration.name)
        )
    logger.info("Using backup %s", recent[0])


def pending_migrations(connection):
    recorded = state.load_all(connection)
    return [
        migration
        for migration in MIGRATIONS
        if recorded.get(migration.version, {}).get("status") != state.STATUS_DONE
    ]


def status_report(connection):
    recorded = state.load_all(connection)
    lines = []
    for migration in MIGRATIONS:
        row = recorded.get(migration.version)
        status = row["status"] if row else "pending"
        finished = row["finished_at"].isoformat() if row and row["finished_at"] else ""
        notes = row["notes"] if row and row["notes"] else ""
        lines.append("%03d  %-32s  %-8s  %s  %s" % (migration.version, migration.name, status, finished, notes))
    return lines


def _verify(context, migration):
    problems = migration.verify(context)
    if problems:
        for problem in problems[:50]:
            context.logger.error("[%03d] %s", migration.version, problem)
        raise MigrationError("Migration %03d %s failed verification with %d problem(s)" % (migration.version, migration.name, len(problems)))


def _run_one(engine, connection, migration, logger, dry_run):
    context = MigrationContext(engine, connection, migration, logger, dry_run=dry_run)
    if dry_run:
        for line in migration.describe(context) or ["no description"]:
            logger.info("[dry run %03d %s] %s", migration.version, migration.name, line)
        return
    _require_backup(connection, migration, logger)
    state.mark_started(connection, migration)
    try:
        migration.run(context)
        _verify(context, migration)
    except Exception as error:
        connection.rollback()
        state.mark_finished(connection, migration, state.STATUS_FAILED, str(error)[:1000])
        raise
    state.mark_finished(connection, migration, state.STATUS_DONE)
    logger.info("Migration %03d %s done", migration.version, migration.name)


def _verify_one(engine, connection, migration, logger):
    context = MigrationContext(engine, connection, migration, logger)
    _verify(context, migration)
    logger.info("Migration %03d %s verified", migration.version, migration.name)


def run(engine, logger, dry_run=False, verify_only=False):
    with engine.connect() as connection:
        connection.execute(text("SELECT pg_advisory_lock(:key)"), {"key": ADVISORY_LOCK_KEY})
        connection.commit()
        try:
            state.ensure_table(connection)
            if verify_only:
                recorded = state.load_all(connection)
                for migration in MIGRATIONS:
                    if recorded.get(migration.version, {}).get("status") == state.STATUS_DONE:
                        _verify_one(engine, connection, migration, logger)
                return
            pending = pending_migrations(connection)
            if not pending:
                logger.info("No pending migrations")
            for migration in pending:
                _run_one(engine, connection, migration, logger, dry_run)
        finally:
            connection.execute(text("SELECT pg_advisory_unlock(:key)"), {"key": ADVISORY_LOCK_KEY})
            connection.commit()


def show_status(engine):
    with engine.connect() as connection:
        state.ensure_table(connection)
        return status_report(connection)
