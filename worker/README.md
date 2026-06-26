# AstroSwarm Evaluation Worker

A standalone service that runs FARP benchmark jobs for the AstroSwarm web server. Workers pull queued evaluations from the server, run the Godot dedicated server locally, and post the merged results and replays back. Run as many workers as you like, on as many machines as you like, all pointed at the same server.

## How it works

1. On startup the worker generates a stable id (persisted to `WORKER_ID_FILE`) and registers with the server.
2. It polls `/api/worker/claim`; when it claims a queued evaluation it runs the Godot dedicated server, splitting the work into `WORKER_MAX_JOBS` parallel processes.
3. It streams progress to the server (which also signals cancellation), then posts the result.
4. The admin Workers page shows it live and can set its max parallel jobs or connect/disconnect/remove it.

## Configuration (environment variables)

| Variable | Description | Default |
|---|---|---|
| `SERVER_URL` | Base URL of the web server | `http://server:5050` |
| `API_SECRET_KEY` | Must match the server's key | `dev_secret_key` |
| `WORKER_NAME` | Display name in the admin panel | hostname |
| `WORKER_MAX_JOBS` | Default parallel Godot processes (admin can override) | `4` |
| `GODOT_SERVER_BIN` | Path to the headless Godot dedicated-server binary | `/app/server_build/AstroSwarm_Linux.x86_64` |
| `GODOT_PCK` | Path to the exported `.pck` (only if the binary needs a separate pack) | _(unset)_ |
| `EVAL_TIMEOUT_SECONDS` | Max wall-clock time per job | `1800` |
| `EVAL_FIXED_FPS` | Fixed physics step for the benchmark | `60` |
| `WORKER_POLL_SECONDS` | Idle poll interval | `3` |
| `WORKER_ID_FILE` | Where the worker id is stored | `/data/worker_id` |

## Running

With Docker Compose (from `web/`): `docker compose up -d --scale worker=N`.

Standalone: `pip install -r requirements.txt` then `python worker.py` with the variables above set. The Godot dedicated-server build must be available at `GODOT_SERVER_BIN`.
