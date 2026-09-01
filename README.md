# AstroSwarm Web

![Version](https://img.shields.io/badge/version-0.0.8-blue)
![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte&logoColor=white)
![TailwindCSS](https://img.shields.io/badge/TailwindCSS-4-06B6D4?logo=tailwindcss&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-6-3178C6?logo=typescript&logoColor=white)
![Flask](https://img.shields.io/badge/Flask-3.1-000000?logo=flask&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)
![Bun](https://img.shields.io/badge/Bun-1.3.14-FBF0DF?logo=bun&logoColor=black)

The companion website for AstroSwarm, a pixel-art swarm-behavior simulator built in Godot 4. The site lets players share simulator configurations and recorded runs, view a community gallery, check the leaderboard, and download the game.

---

## Pages

### Home
Landing page with an animated starfield background, a game overview, and navigation to all other sections.

### Simulator Gallery (`/simulator`)
Browse community-uploaded recorded runs. Each card shows the species list with their colors, robot count, arena dimensions, and frame count. Clicking on a run opens a dedicated detail page featuring a streaming video player and run statistics. Runs (`.run`) are uploaded directly from Godot and parsed automatically by the backend.

### Leaderboard (`/leaderboard`)
Commanders ranked by a **weighted rating** rather than a raw average success rate. Each row shows the rating, the unweighted average behind it, how many entries the commander has submitted, and how many of the levels they have played. Entries link to a profile with the per-level breakdown, including the weighted rate and the level average it was pulled toward.

#### How the weighted rating works

A plain mean over a commander's entries rewards two things it should not: a single lucky submission, and playing only the levels that are easy to win. One entry at 100% would top the board over someone with eighteen entries averaging 40%, and a commander who only ever played Level 1 would outrank one who played everything.

The rating fixes both, in three steps.

**1. Score each level on its own.** A commander's rate on a level is the mean success rate of their finished entries on that level. Levels are never mixed together at this stage, so a strong Level 1 result cannot paper over a weak Level 4 one.

**2. Pull each level rate toward that level's average.** A rate measured from one entry is mostly noise; a rate measured from fifty is close to the truth. Each level rate is therefore blended with the level's own average, weighted by how many entries back it:

```
weighted = (entries * rate + 2 * level_average) / (entries + 2)
```

The `2` is a constant number of imaginary average entries every commander is credited with (`PRIOR_ENTRIES` in `server/rating.py`). One entry at 100% on a level averaging 30% lands at roughly 53, not 100. Eighteen entries move barely at all: the evidence outweighs the prior. Nothing is thrown away and no minimum entry count is imposed — a small sample is simply trusted less.

The level average is the mean of each commander's *own* mean on that level, counting every commander once. Averaging over raw entries instead would let one prolific commander drag a level's baseline toward their personal results.

**3. Average across every level, filling in the ones not played.** A level a commander has never touched counts as that level's average rather than being skipped. This is what stops partial coverage from inflating a rating: a commander who only played the easy level is treated as merely average everywhere else, so their rating sits between their result and the field's. Playing more levels well is the only way to pull it up.

The final rating is the mean of the per-level weighted rates across all levels that have any data.

Worked example, with Level 1 averaging 60% and Level 2 averaging 30%:

| Commander | Level 1 | Level 2 | Rating |
|---|---|---|---|
| One lucky entry | 100% from 1 entry | never played | `(1*100 + 2*60)/3 = 73.3`, then `(73.3 + 30)/2` = **51.7** |
| Consistent, both levels | 80% from 10 entries | 60% from 10 entries | `76.7` and `55`, so **65.9** |

The second commander ranks higher despite a lower headline number on Level 1, which is the intent.

Ties break on levels played, then entry count, so the more thoroughly tested commander wins. A commander with no finished entries has no rating and sorts below everyone who does. The raw average is still shown everywhere the rating is, so nothing is hidden — the rating decides the order, the average tells you what actually happened.

The implementation is `server/rating.py`, applied in `_aggregate_players` so the public leaderboard, the profile pages and the admin tables all rank identically.

### Timed Local leaderboard (`/api/leaderboard`)
Separate from the above: rankings for the Timed Local game mode showing username, completion time, and the behavior algorithm the player used.

### Levels (`/gamemodes/levels`)
An index of the seven FARP levels, each with a page of its own at `/gamemodes/levels/<n>`. That page has three views, toggled from a row of buttons under the heading:

| View | What it shows |
|---|---|
| **Entries** | Every submission on the level, newest first, with a sidebar of filters: search by username or ID, a minimum-rate slider, a date range, and a sort order. Paged twelve at a time. |
| **Leaderboard** | The level's own board — every commander who has finished an entry on it, ranked by their best rate, with their average, entry count and the defender count behind their best run. A row links straight to that entry. |
| **Comparison** | The defender-sweep curves of the strongest entries drawn on one axis: capture success rate against ring size, and risk against ring size. One line per entry, plus a heavy dashed line for the best rate any entry reached at each `n`. |

The leaderboard and comparison views are served by `/api/evaluations/level-leaderboard` and `/api/evaluations/level-sweep`, so neither one pulls every entry on the site through the client.

An individual entry lives at `/levels/<id>`.

**Levels 1 and 2** are benchmarked: each submitted algorithm is evaluated headlessly by the worker service against the defender layout it was submitted with, and listed with its entry ID, username, status, capture rate, and date. Clicking an entry opens a detail page with the defender and evader configs (speed, turn rate, vision range, FOV), tiles for detection rate / capture rate / mean time to the planet, cumulative and outcome charts, detection- and capture-rate-vs-defender-count charts, frame-perfect placement and ring-sweep replays, and the defender algorithm.

Three events are measured, and they mean different things:

| Event | Definition |
|---|---|
| **Detected** | The first time any defender sees the evader inside its vision cone. |
| **Captured** | The first time any defender physically touches (collides with) the evader. |
| **Reached planet** | The time the evader reaches the center planet (`T_goal` in the admin panel). |

**Levels 3, 4 and 5** are benchmarked as *assaults*: a stream of evaders against one scattered line rather than a single approach. Levels 3 and 4 send them one at a time, wave after wave, each from a fresh random bearing on the ring, with the next launching as soon as the last is resolved. **The game stops at five waves; the benchmark does not.** In game five is enough to feel out an algorithm without sitting through a four-minute run, so the benchmark deliberately runs longer — until the line is spent or the simulated clock stops — and an entry's published rate is therefore over far more evaders than the player watched. Level 5 sends five at once, spread around the **edges of the arena** rather than a ring, so they arrive in a stagger. Level 3 destroys the captured evader alone and runs until the clock stops; Levels 4 and 5 destroy the defender that caught it as well, so each kill costs a body and the run also ends when the line is spent.

The headline number on these levels is the **capture success rate** — the share of every evader that reached a verdict that the line destroyed (one still in flight when the clock stopped counts for neither side) — and the entry also reports how many reached the planet and, where attrition applies, how many defenders it cost. Each entry is graded over 100 trials, each with its own seeded defender scatter and spawn bearings, followed by a defender sweep. The detail page adds a **risk** chart (`Risk = 1 - capture success rate`, plotted against the size of the ring), plus three ring-sweep charts that give **every ring size its own colored line** — cumulative capture success rate, risk, and defenders still standing, each read wave by wave against the number of evaders faced. Those three are drawn on all of Levels 3, 4 and 5; Level 3 has no attrition, so its line-still-standing curve is flat by definition. On the attrition levels there are two more: how the risk moved as the submitted layout thinned, and the same curve bucketed by ring size. Only the first trials keep a frame recording, since these runs are far longer than a single approach; the rest are graded but not replayable.

**Levels 6 and 7** are *piloted runs*, not benchmarks. In Level 6 the player flies the evader themselves against the best submitted Level 2 algorithm — and that entry's defender placements — capped at three minutes; Level 7 is the swarm merge. Any run is submitted, caught or clean. The recorded flight is uploaded and rendered by a worker into a replay, so the detail page shows the run's outcome, its detected / captured / reached-planet times, the recorded flight, and the opponent's algorithm. Charts and the ring sweep are hidden for these levels, since a single flight has no rates to plot.

The per-level admin settings page (`/admin/settings/level-6`) drops the benchmark parameters entirely for a pilot level and shows the render pipeline and run limits instead.

#### Level numbering

The wave levels were added in v0.0.7, moving the pilot and swarm levels up to 5 and 6. In v0.0.8 the two wave levels became continuous-wave levels, a siege level took slot 5, and the pilot and swarm levels moved up again to 6 and 7. Each shift is a guarded migration that renames the stored ids once (`farp` to `farp1`; then `farp3` to `farp5` and `farp4` to `farp6`; then `farp5` to `farp6` and `farp6` to `farp7`) and records a marker in `app_settings` so it can never fire a second time and shift genuine new entries. The v0.0.8 shift additionally waits for the earlier marker, so a database behind on both is never shifted twice in one pass. Only the current game version may submit, which is what stops an old client filing a run under an id that has since moved.

### Survive (`/survive`)
Match reports from the game's two-player Survive mode. Each card shows the two commanders, the winner (or a tie), how many evaders reached each planet, and both average APMs; a sidebar provides a search bar (commander or match ID) and sort order.

Clicking a match opens its detail page: the headline result, a per-player breakdown (evaders through, defenders left at the buzzer, ships herded, freezes used, total actions, average and peak APM), and the **actions-per-minute line chart** with both players plotted against match time. The chart has a "view as table" panel underneath with the raw per-sample values.

An action is a movement input changing state — a key going down or up, or a controller stick crossing into a new direction — plus every power-up activation. Samples are taken every 5 seconds and scaled to a per-minute rate. Matches are submitted automatically by the game when the clock runs out.

### Admin CMS (`/admin`)
API-key gated management panel (client-side session stored in `localStorage`) with a flat, light-gray UI. It lists evaluations, leaderboard entries, and simulator runs with pagination, per-entry viewer pages, and a one-click ZIP export of each entry (metadata plus per-run JSON). The evaluations list adds search and status/level/date/sort filters. The evaluation viewer can **re-simulate** an entry, requeueing it for the workers to refresh its results and replays. A **Workers** page shows every connected worker node with live status, and lets you rename a worker or connect/disconnect/remove it.

Row actions on the evaluations and players tables are icon buttons, and both tables support **multi-select**: tick any number of rows and delete them in bulk, or requeue a selection of evaluations for re-simulation. Every destructive bulk action confirms first and warns separately when the selection is large.

**Data storage.** The dashboard lists each table's size, dead-row count and last vacuum. Re-simulating rewrites an entry's whole replay blob, and Postgres keeps the old version as a dead tuple, so a heavily re-simulated table bloats well past the data it holds. A **reclaim** action per table runs `VACUUM (FULL, ANALYZE)` and reports the space returned; the table is locked for the rewrite, which the confirmation says plainly. `player_evaluations` and its TOAST table are also configured to autovacuum far more eagerly than the defaults, so the bloat stops accumulating in the first place.

**Game version.** Settings carries the version gate. Only the build named there may submit entries; everything else is rejected with a `426`. It is stored in `app_settings`, so releasing a new game version is a settings change rather than a deploy.

### Evaluation Workers

Benchmarks run on separate **worker** processes rather than in the web server, so compute can be scaled across machines. An evaluation is **one unit of work**: the server queues it whole, a single worker claims it, and that worker runs every match in it. A Level 6 or Level 7 piloted run is a **render job**: the run was already simulated in the game client, so the worker just packs the recorded trajectory into a replay without simulating anything.

A worker (`web/worker/`) is a single static Go binary that re-implements the match loop in process — there is no game build to download and nothing to bundle into the image. It registers with the server, claims one queued evaluation whenever it is idle, and simulates that evaluation's matches in parallel across all its cores (`SIM_WORKERS`). It holds exactly one job at a time. Throughput scales by running more workers: each picks up a different evaluation.

Workers auto-connect on startup; an admin can disconnect one from the Workers page (its in-flight job is requeued for another worker), and jobs from workers that go silent are automatically requeued — a finished job is never re-run. To add compute, run a worker on another machine (its own data volume gives it a stable, distinct id) pointed at the server's public URL with the matching `API_SECRET_KEY`.

### Downloads (`/downloads`)
Links to the latest AstroSwarm game releases fetched live from the GitHub Releases API.

### Previews (`/previews`)
Internal preview page for component and layout development.

---

## Stack

| Layer | Technology |
|---|---|
| Frontend framework | SvelteKit 2 + Svelte 5 (runes mode) |
| Styling | TailwindCSS 4 |
| Frontend runtime | Static SPA (`adapter-static`, `200.html` fallback) |
| Backend | Flask 3 + Gunicorn (4 workers, preload) |
| Database | PostgreSQL 16 via SQLAlchemy + psycopg2 |
| Package manager | Bun |
| Containerization | Docker Compose (multi-stage build) |

---

## API


### Simulator Runs

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/runs` | List runs (paginated) |
| `POST` | `/api/runs` | Upload a `.run` file, `.cfg` config, and video (zipped on server) |
| `GET` | `/api/runs/<id>` | Get a single run |
| `GET` | `/api/runs/<id>/download` | Download the zipped run files |
| `GET` | `/api/runs/<id>/thumbnail` | Get the generated video thumbnail |
| `GET` | `/api/runs/<id>/video` | Stream the raw video file |

### Evaluations

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/evaluations` | List evaluations (most recent) |
| `POST` | `/api/evaluations` | Submit an algorithm for benchmarking (levels 1-5); an identical submission (same level, algorithm, placements, trials) reuses the existing result instead of re-running. Rejected with `426` unless the client is the required game version (`X-API-Key` required) |
| `POST` | `/api/evaluations/run` | Submit a piloted run (levels 6-7): the recorded flight is queued for a worker to render into a replay. Rejected with `426` unless the client is the required game version (`X-API-Key` required) |
| `GET` | `/api/evaluations/best` | Best submitted algorithm and placements for a level (`?level_id=farp2`), picked at random among ties — this is the opponent the game's level 6 plays against |
| `GET` | `/api/evaluations/<id>` | Get a single evaluation |
| `GET` | `/api/evaluations/baseline` | Average success rate across completed runs |
| `GET` | `/api/evaluations/level-leaderboard` | One level's own board (`?level_id=farp3`): every commander who finished an entry on it, ranked by their best rate |
| `GET` | `/api/evaluations/level-sweep` | The defender-sweep curves behind a level's comparison graphs (`?level_id=farp3&limit=6`), strongest entries first |
| `GET` | `/api/evaluations/players` | Commanders ranked by weighted rating, with the raw average, entry count and level coverage behind it |
| `GET` | `/api/evaluations/players/<id>` | One commander's profile: rating, per-level weighted rates and level averages, ranks and recent entries |
| `GET` | `/api/evaluations/settings` | Benchmark parameters, the level list and the required game version |
| `PUT` | `/api/evaluations/settings` | Update benchmark parameters, enable/disable a level, or set the required game version (`X-API-Key` required) |
| `GET` | `/api/evaluations/<id>/replays` | Placement-run replay index |
| `GET` | `/api/evaluations/<id>/replay/<trial>` | Replay frames for one placement trial |
| `GET` | `/api/evaluations/<id>/sweep-replays` | Ring-sweep replay index (n, outcome, detection/capture time) |
| `GET` | `/api/evaluations/<id>/sweep-replay/<n>` | Replay frames for one ring-sweep run |
| `GET` | `/api/evaluations/<id>/chart/<kind>.png` | Rendered chart PNG (`line`, `bar`, `sweep` detection rate, `capture` capture rate, `risk`, `attrition`, `sweep-attrition`, `ring-capture`, `ring-risk`, `ring-attrition`, `times`) |
| `GET` | `/api/evaluations/<id>/export` | Download a ZIP of the entry and per-run JSON |
| `POST` | `/api/evaluations/<id>/claim-xp` | Claim the XP an entry earned, once per entry; a level-6 goal is worth far more than a benchmark (`X-API-Key` required) |
| `POST` | `/api/evaluations/<id>/resimulate` | Re-run an evaluation on the current build (`X-API-Key` required) |
| `POST` | `/api/evaluations/<id>/cancel` | Cancel a queued or running evaluation (`X-API-Key` required) |
| `DELETE` | `/api/evaluations/<id>` | Delete an evaluation (`X-API-Key` required) |

### Survive

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/survive/matches` | List Survive matches (most recent; `?limit=` and `?player_id=` supported) |
| `GET` | `/api/survive/matches/<id>` | Get a single match with both APM series |
| `POST` | `/api/survive/matches` | Submit a finished Survive match (`X-API-Key` required) |
| `DELETE` | `/api/survive/matches/<id>` | Delete a match (`X-API-Key` required) |

### Workers

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/workers` | List worker nodes with live status (`X-API-Key` required) |
| `GET` | `/api/workers/<id>` | Get a single worker (`X-API-Key` required) |
| `POST` | `/api/workers/<id>/settings` | Rename a worker (`X-API-Key` required) |
| `POST` | `/api/workers/<id>/connect` | Re-enable a worker (`X-API-Key` required) |
| `POST` | `/api/workers/<id>/disconnect` | Stop a worker taking jobs; requeue its current job (`X-API-Key` required) |
| `DELETE` | `/api/workers/<id>` | Remove a worker record (`X-API-Key` required) |
| `POST` | `/api/worker/register` | Worker announces itself (used by workers) |
| `POST` | `/api/worker/heartbeat` | Keep-alive and status report (used by workers) |
| `POST` | `/api/worker/claim` | Claim the next queued evaluation (used by workers) |
| `POST` | `/api/worker/shards/<id>/progress` | Report shard progress; response signals cancellation (used by workers) |
| `POST` | `/api/worker/shards/<id>/result` | Submit a shard's results/replays; the server merges when all shards are done (used by workers) |
| `POST` | `/api/worker/shards/<id>/fail` | Report a failed shard (used by workers) |

### Leaderboard

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/leaderboard` | Top 100 entries |
| `GET` | `/api/leaderboard/<id>` | Single entry with full algorithm |
| `GET` | `/api/leaderboard/<id>/export` | Download a ZIP of the entry |
| `POST` | `/api/leaderboard` | Submit or update a time (`X-API-Key` required) |
| `DELETE` | `/api/leaderboard/<id>` | Delete an entry (`X-API-Key` required) |

### Errors

Every endpoint returns errors as JSON (`{"error": "...", "status": 404}`) rather than an HTML page, so the game and the frontend can show the server's message directly.

### Health

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Server health check |

### Admin

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/admin/storage` | Database and upload sizes per table, with dead-row counts and the last vacuum time (`X-API-Key` required) |
| `POST` | `/api/admin/storage/reclaim` | `VACUUM (FULL, ANALYZE)` one table and report the bytes freed (`X-API-Key` required) |

### Version

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/version` | Get the latest GitHub release info |

---

## Environment Variables

Copy `.env.example` to `.env` and fill in the values.

| Variable | Description | Default |
|---|---|---|
| `DB_NAME` | PostgreSQL database name | `astroswarm` |
| `DB_USER` | PostgreSQL user | `astroswarm` |
| `DB_PASSWORD` | PostgreSQL password | `changeme` |
| `PORT` | Port the server listens on | `5050` |
| `FRONTEND_ORIGIN` | Allowed CORS origin | `http://localhost:5173` |
| `API_SECRET_KEY` | Key required for write/admin endpoints; also authenticates workers | `dev_secret_key` |
| `PUBLIC_API_URL` | API base URL used by the frontend | `http://localhost:5050` |
| `EVAL_MATCH_CAP_SECONDS` | Hard cap on a single benchmarked match | `240` |
| `EVAL_GOAL_TAIL_SECONDS` | Extra time the sim keeps running after T_goal, so a late capture is still recorded | `3` |

The following are used by the **worker** service (`web/worker/`), not the web server:

| Variable | Description | Default |
|---|---|---|
| `WORKER_SERVER_URL` | How the worker reaches the server (internal name in Docker, or a public URL on another machine) | `http://server:5050` |
| `WORKER_NAME` | Display name shown in the admin Workers page | `worker` |
| `SIM_WORKERS` | Matches simulated in parallel within the job | core count |
| `EVAL_SWEEP_SPAWN` | `fixed` reproduces the original ring-sweep spawn; `varied` stratifies it per trial and regrades every entry | `fixed` |
| `EVAL_TIMEOUT_SECONDS` | Max wall-clock time for a single job | `1800` |

The full worker reference is in `worker/WORKER.md`.

In Docker the frontend is served by Flask on the same origin, so `PUBLIC_API_URL` is set to an empty string and all API requests are same-origin relative paths.

---

## Running with Docker

The `server` container expects an NVIDIA GPU to be available to generate video thumbnails efficiently using `ffmpeg`. Ensure the host machine has the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) installed.

```bash
cp .env.example .env
docker compose up -d --build
```

The server is available at `http://localhost:5050`. Compose also starts one `worker` container that runs evaluations. To add more compute, run additional workers on other machines (each with its own data volume so it gets a stable, distinct id): use the image published to GHCR by the **Worker** GitHub Action (or build `worker/Dockerfile`) and set `WORKER_SERVER_URL` to the server's public URL with the matching `API_SECRET_KEY`.

### Database migrations

`server/migrations.py` holds every schema change as an idempotent statement. The server runs
them itself on startup, right after `db.create_all()`, so a normal deploy needs nothing extra.

To apply them by hand against a running stack:

```bash
docker compose exec server python migrations.py
```

It is safe to run repeatedly — each statement is `IF NOT EXISTS` / `IF EXISTS`, and anything
that cannot apply is logged and skipped rather than aborting the rest. Add new changes by
appending to `STATEMENTS`; never edit or reorder the existing entries.

## Running Locally

**Prerequisites:** You must have `ffmpeg` installed on your system to generate video thumbnails when running locally.

```bash
# Frontend
bun install
bun run dev

# Worker (in worker/) — runs evaluation shards
go build -o astroworker ./cmd/astroworker
SERVER_URL=http://localhost:5050 API_SECRET_KEY=dev_secret_key \
  WORKER_ID_FILE=./worker_id ./astroworker

# Backend (in server/)
python3 -m venv venv
source ./venv/bin/activate
pip install -r requirements.txt
flask --app main run --port 5050
```

---