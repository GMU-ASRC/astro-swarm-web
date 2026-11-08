# Migrations

Every schema or data change is a numbered file in `worker/internal/server/migrations/`. The `schema_migrations` table records which ones have run. A migration that is marked `done` never runs again.

## Commands

All commands run the `astroserver` binary inside the server image.

| Command | What it does |
|---|---|
| `docker compose run --rm migrate migrate --status` | List every migration and its status |
| `docker compose run --rm migrate migrate --dry-run` | Describe pending migrations without writing anything |
| `docker compose run --rm migrate migrate` | Run pending migrations |
| `docker compose run --rm migrate migrate --manual --only <version>` | Run one manual migration (long conversions and cleanups) |
| `docker compose run --rm migrate migrate --verify-only` | Re-run verification for finished migrations |
| `docker compose run --rm migrate backup` | Back up the database and upload files, then verify the backup |

`docker compose up` runs the `migrate` service first. The server starts only if it finishes without errors. Manual migrations (5, 7 and 8) never run on `docker compose up`; they wait for `--manual`.

## Rules

- Migrations run in version order, under a Postgres advisory lock, so two copies can never run at once.
- A migration that stopped part way is marked `running` or `failed`. Running `migrate` again resumes it.
- A migration that changes data refuses to run unless a verified backup from the last 24 hours exists in `BACKUP_DIR`. An empty database is exempt.
- Migrations copy data into new tables or files. Old data is removed only by a separate cleanup migration after verification passes.

## Backups

`backup` writes a folder named `astroswarm-YYYYMMDD-HHMMSS` into `BACKUP_DIR` (`./backups` on the host by default, set `BACKUP_PATH` in `.env` to use another disk):

| Path | Contents |
|---|---|
| `astroswarm-.../database.dump` | `pg_dump --format=custom` of the whole database |
| `astroswarm-.../manifest.json` | Time, sizes, file count and `verified: true` when every check passed |
| `uploads-mirror/` | One shared copy of the uploads volume, replays included |

Upload files are copied into `uploads-mirror/` only when they are new or their size or time changed, so after the first backup a new one takes seconds instead of copying 60 GB again. Files are never deleted from the mirror, so every older database dump still has its replay files.

The backup is checked with `pg_restore --list` and by comparing the size of every copied file. A backup with problems is written with `verified: false` and the command exits with an error.

Stop the workers before a backup (`docker compose stop worker`, and any remote workers) so no replay upload is in progress while files are copied.

## Restore

1. Stop everything except the database:
   `docker compose stop server worker`
2. Restore the database into a clean schema:
   `docker compose exec -T db psql -U $DB_USER -d $DB_NAME -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"`
   `docker compose exec -T db pg_restore --no-owner --no-privileges -U $DB_USER -d $DB_NAME < backups/<folder>/database.dump`
3. Restore the upload files:
   `docker compose run --rm --entrypoint sh migrate -c "rm -rf /app/uploads/* && cp -a /app/backups/uploads-mirror/. /app/uploads/"`
   Backups made before the mirror existed keep their files in `backups/<folder>/uploads/` instead.
4. Check the migration state. The restored `schema_migrations` table matches the backup, so migrations that ran after the backup show as pending:
   `docker compose run --rm migrate migrate --status`
5. Start the stack:
   `docker compose up -d`

To run a migration again from scratch after a restore, delete its row first:
`docker compose exec db psql -U $DB_USER -d $DB_NAME -c "DELETE FROM schema_migrations WHERE version = <number>;"`

## Migration 5 (archive the old results and replays columns)

`player_evaluations` still holds the `results` and `replays` columns that migration 4 copied into the benchmark tables. They are 2.4 GB of the database. Migration 5 copies both columns into `player_evaluations_archive`, checks every value matches, drops the columns and rewrites `player_evaluations` so the space is given back. The archive table itself is only dropped by hand.

Run it no earlier than two weeks after the Go server is live, so Flask can still be restored until then:

1. Stop the server: `docker compose stop server worker`
2. Back up (needs about 3 GB free for the archive copy): `docker compose run --rm migrate backup`
3. Run it: `docker compose run --rm migrate migrate --manual --only 5`
4. Start again: `docker compose up -d`

After this, the Flask image can no longer read the database.

## Deploying Migrations 3 and 4 (benchmark tables)

Migration 4 copies every entry's results and replay data into the `benchmarks` and `benchmark_runs` tables and writes replay frames kept inside the database out to `replays/<entry id>/inline/`. The `results` and `replays` columns are not changed. It needs a verified backup from the last 24 hours.

1. In the admin panel, disconnect every worker and wait until no job is running.
2. Stop the server so no result is written while the data is copied:
   `docker compose stop server worker`
3. Check there is about 65 GB free where `BACKUP_PATH` points (`df -h`), then back up:
   `docker compose run --rm migrate backup`
4. Deploy. The `migrate` service copies and verifies every row before the server starts:
   `git pull`
   `docker compose up -d --build`
5. Check the result:
   `docker compose logs migrate`
   `docker compose run --rm migrate migrate --status`
6. Reconnect the workers in the admin panel.

If migration 4 reports problems, it is marked `failed`, the server does not start, and no original data has been changed. Send the `docker compose logs migrate` output, or restore the previous image with `git checkout <previous commit>` and `docker compose up -d --build`.

## Migrations 6 to 8 (binary replay files)

Migration 6 runs on deploy and only adds two columns. New results get a binary replay file (`.asr`) next to the old `.zlib` file from then on.

Migration 7 converts the existing files. The site can stay up while it runs:

1. Back up: `docker compose run --rm migrate backup`
2. Convert: `docker compose run --rm migrate migrate --manual --only 7`

It writes a binary copy next to each old file and never changes the old one. Each copy is read back and compared value by value before its row points at it, and verification then reads every old and new file again. Runs whose frames are not plain number lists keep only the old file and are listed in the log.

Migration 8 deletes the old files. Run it once the site has worked on the binary files for a while:

1. Back up again: `docker compose run --rm migrate backup`
2. `docker compose run --rm migrate migrate --manual --only 8`

It compares each old file with its binary copy once more before deleting it. Afterwards the server does the same for new results every 10 minutes.

## Comparing Flask and Go before the switch

`docker-compose.compare.yml` runs Flask and the Go server side by side on a restored copy of the database. Both read the real uploads volume read-only, so production is never written to. `apicompare` then sends the same GET requests to both and lists every JSON difference.

1. Make a fresh backup of production: `docker compose run --rm migrate backup`
2. Find the uploads volume name (`docker volume ls`, usually `web_uploads`) and set `UPLOADS_VOLUME` in `.env` if it differs.
3. Start the copy's database and restore the dump into it:
   `docker compose -f docker-compose.compare.yml up -d db`
   `docker compose -f docker-compose.compare.yml exec db pg_restore --no-owner --no-privileges -U $DB_USER -d $DB_NAME /backups/<folder>/database.dump`
4. Start both servers: `docker compose -f docker-compose.compare.yml up -d --build flask go`
5. Compare: `docker compose -f docker-compose.compare.yml run --rm apicompare`
   Add `-verbose` to list matching routes with both timings, or `-per-level 1000 -per-list 1000` to check every entry.
6. Remove the copy when done: `docker compose -f docker-compose.compare.yml down -v`

Routes marked NOTE are expected to vary (random tie-breaks, GitHub lookups). Chart images and thumbnails are compared by status and type only, because the Go charts are drawn by a different library.
