# AstroSwarm Worker

`astroworker` replaces the Python evaluation worker and the headless Godot build it drove.
It speaks the same
`/api/worker/*` protocol, claims the same shards, and posts results in the same shape — but
it simulates the matches in-process instead of shelling out to a headless Godot build.

Nothing on the server changes. Shard splitting, merging, progress, cancellation, and the
admin Workers page all behave exactly as before.

## What it removes

| Before | Now |
|---|---|
| Downloaded a Godot dedicated-server build on every startup | Nothing to download |
| Needs Mesa, Vulkan, X11, fontconfig, ALSA in the image | `ca-certificates` on Alpine |
| One Godot process per shard | One goroutine pool per job |
| Work split into ~32 shards across workers | One worker runs a whole evaluation |
| Progress scraped from `PROGRESS` lines on stdout | Reported directly |

The image is a static binary on Alpine, so no GPU passthrough or software-rendering
fallback is involved.

## Running

With Docker Compose from `web/`, the `worker` service already points at this build:

```
docker compose up -d --build worker
```

Standalone:

```
cd worker
go build -o astroworker ./cmd/astroworker
SERVER_URL=https://astroswarm.autonomousrobotics.club API_SECRET_KEY=... ./astroworker
```

## Configuration

| Variable | Description | Default |
|---|---|---|
| `SERVER_URL` | Base url of the web server | `http://server:5050` |
| `API_SECRET_KEY` | Must match the server's key | `dev_secret_key` |
| `WORKER_NAME` | Display name in the admin panel | hostname |
| `SIM_WORKERS` | Matches simulated in parallel within the job | core count |
| `WORKER_POLL_SECONDS` | Idle poll interval | `3` |
| `WORKER_CANCEL_POLL_SECONDS` | Progress and cancellation poll interval | `2` |
| `EVAL_TIMEOUT_SECONDS` | Max wall-clock time per job | `3600` |
| `WORKER_ID_FILE` | Where the worker id is stored | `/data/worker_id` |
| `EVAL_SWEEP_SPAWN` | `fixed` or `varied` ring-sweep evader spawn | `fixed` |

`GODOT_RELEASE_TAG`, `GODOT_SERVER_BIN`, `GODOT_PCK`, `GODOT_DIR`, `EVAL_FIXED_FPS`, and
`WORKER_MAX_JOBS` are gone — there is no Godot build, the physics step is fixed at 60 Hz in
code, and a worker holds exactly one job at a time.

An evaluation is never split. The server queues it as a single unit, one worker claims it,
and that worker simulates every match in it across `SIM_WORKERS` goroutines. To add
throughput, run more workers: each takes a different evaluation.

### EVAL_SWEEP_SPAWN

`fixed` is the server's historical default: every ring-sweep run launches the evader from the
one configured spawn, so only the ring rotation varies across the n2 repeats.

`varied` gives each trial its own stratified approach angle, the way the placement runs
already work. It is the better measurement — see the ring-sweep section of `README.md` — but
it changes every sweep number the site reports, so existing entries have to be re-simulated
before their curves mean anything again. The default stays `fixed` so swapping the worker in
does not silently regrade the leaderboard.

## Protocol

Identical to the Python worker:

1. Read or create a worker id at `WORKER_ID_FILE`, `POST /api/worker/register`.
2. Poll `POST /api/worker/claim` while idle; the server hands back at most one job.
3. Per job, `POST /api/worker/shards/<id>/progress` every `WORKER_CANCEL_POLL_SECONDS`,
   which doubles as the cancellation channel — a `cancel: true` response aborts the run.
4. `POST /api/worker/shards/<id>/result` with `{runs, sweep_runs, meta}`, or
   `POST /api/worker/shards/<id>/fail` on error.
5. `POST /api/worker/heartbeat` every poll with status and system stats.

System stats are read from `/proc` and `statfs` rather than psutil, and report the same keys
(`cpu_percent`, `cpu_count`, `memory_*`, `disk_*`, `load_avg_1m`). On a non-Linux host only
`cpu_count` is sent.

Replay frames are delta-encoded, zlib-compressed at level 9, and base64 encoded — the format
`models._unpack_frames` reads.

Level 6 and Level 7 jobs carry a `run` payload rather than an algorithm. Nothing is simulated;
the recorded trajectory is packed into a single-run replay.

Level 3, 4 and 5 jobs are simulated as assaults rather than single-evader matches, described
in the assault benchmark section of `README.md`.

## Shard result shape

```json
{
  "runs":       [{"trial": 0, "outcome": "win", "detection_time": 1.2,
                  "capture_time": 3.4, "goal_time": 8.3, "frames_packed": "...",
                  "stats": {"sent": 26, "resolved": 25, "destroyed": 20,
                            "breaches": 5, "defenders": 5, "lost": 20,
                            "end_time": 240.0}}],
  "sweep_runs": [{"n": 1, "outcome": "lose", "detection_rate": 40.0, "capture_rate": 10.0,
                  "defenders": 1, "trial_runs": [...], "frames_packed": "..."}],
  "meta":       {"fps": 60, "defenders": 5, "view": 300, "fov": 70, "speed": 150,
                 "planet": [1920, 1080, 120], "arena": [3840, 2160]}
}
```

`stats` is only present on an assault run (levels 3 to 5). Everything else the site needs is
derived from it and from the `results` block the job also carries.

On levels 1 and 2 every placement trial is recorded, and in the ring sweep only trial 0 of
each n plus the replay trials (`n < 50` and `trial < 10`) are.
An assault run is far longer, so those levels record every third physics frame — `meta.fps`
drops to 20 to match — and only the first 25 placement trials and a smaller slice of the sweep
(`n <= 20` and `trial < 4`) keep frames at all.
