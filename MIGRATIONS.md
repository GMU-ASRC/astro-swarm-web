# Migrations

Every schema or data change is a numbered file in `server/migrator/versions/`. The `schema_migrations` table records which ones have run. A migration that is marked `done` never runs again.

## Commands

All commands run inside the server image.

| Command | What it does |
|---|---|
| `docker compose run --rm migrate python migrate.py --status` | List every migration and its status |
| `docker compose run --rm migrate python migrate.py --dry-run` | Describe pending migrations without writing anything |
| `docker compose run --rm migrate python migrate.py` | Run pending migrations |
| `docker compose run --rm migrate python migrate.py --verify-only` | Re-run verification for finished migrations |
| `docker compose run --rm migrate python make_backup.py` | Back up the database and upload files, then verify the backup |

`docker compose up` runs the `migrate` service first. The server starts only if it finishes without errors.

## Rules

- Migrations run in version order, under a Postgres advisory lock, so two copies can never run at once.
- A migration that stopped part way is marked `running` or `failed`. Running `migrate.py` again resumes it from its last saved batch.
- A migration that changes data refuses to run unless a verified backup from the last 24 hours exists in `BACKUP_DIR`. An empty database is exempt.
- Migrations copy data into new tables or files. Old data is removed only by a separate cleanup migration after verification passes.

## Backups

`make_backup.py` writes a folder named `astroswarm-YYYYMMDD-HHMMSS` into `BACKUP_DIR` (`./backups` on the host by default, set `BACKUP_PATH` in `.env` to use another disk):

| File | Contents |
|---|---|
| `database.dump` | `pg_dump --format=custom` of the whole database |
| `uploads/` | A copy of every file in the uploads volume, replays included |
| `manifest.json` | Time, sizes, file count and `verified: true` when every check passed |

The backup is checked with `pg_restore --list` and by comparing the size of every copied file. A backup with problems is written with `verified: false` and the command exits with an error.

Stop the workers before a backup (`docker compose stop worker`, and any remote workers) so no replay upload is in progress while files are copied.

## Restore

1. Stop everything except the database:
   `docker compose stop server worker`
2. Restore the database into a clean schema:
   `docker compose exec -T db psql -U $DB_USER -d $DB_NAME -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"`
   `docker compose exec -T db pg_restore --no-owner --no-privileges -U $DB_USER -d $DB_NAME < backups/<folder>/database.dump`
3. Restore the upload files:
   `docker compose run --rm --entrypoint sh migrate -c "rm -rf /app/uploads/* && cp -a /app/backups/<folder>/uploads/. /app/uploads/"`
4. Check the migration state. The restored `schema_migrations` table matches the backup, so migrations that ran after the backup show as pending:
   `docker compose run --rm migrate python migrate.py --status`
5. Start the stack:
   `docker compose up -d`

To run a migration again from scratch after a restore, delete its row first:
`docker compose exec db psql -U $DB_USER -d $DB_NAME -c "DELETE FROM schema_migrations WHERE version = <number>;"`

## Deploying Migrations 3 and 4 (benchmark tables)

Migration 4 copies every entry's results and replay data into the `benchmarks` and `benchmark_runs` tables and writes replay frames kept inside the database out to `replays/<entry id>/inline/`. The `results` and `replays` columns are not changed. It needs a verified backup from the last 24 hours.

1. In the admin panel, disconnect every worker and wait until no job is running.
2. Stop the server so no result is written while the data is copied:
   `docker compose stop server worker`
3. Check there is about 65 GB free where `BACKUP_PATH` points (`df -h`), then back up:
   `docker compose run --rm migrate python make_backup.py`
4. Deploy. The `migrate` service copies and verifies every row before the server starts:
   `git pull`
   `docker compose up -d --build`
5. Check the result:
   `docker compose logs migrate`
   `docker compose run --rm migrate python migrate.py --status`
6. Reconnect the workers in the admin panel.

If migration 4 reports problems, it is marked `failed`, the server does not start, and no original data has been changed. Send the `docker compose logs migrate` output, or restore the previous image with `git checkout <previous commit>` and `docker compose up -d --build`.
