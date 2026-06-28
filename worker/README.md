# AstroSwarm Evaluation Worker

A standalone service that runs FARP benchmark jobs for the AstroSwarm web server. Workers pull queued evaluations from the server, run the Godot dedicated server locally, and post the merged results and replays back. Run as many workers as you like, on as many machines as you like, all pointed at the same server.

## How it works

1. On startup the worker downloads the dedicated-server build (`AstroSwarm_Linux_Server.zip`) from the GitHub release (`GODOT_RELEASE_TAG`, default `latest`) and unzips it into `GODOT_DIR` — re-downloading only when the release tag changes. Set `GODOT_SERVER_BIN` to a provided binary to skip this.
2. It generates a stable id (persisted to `WORKER_ID_FILE`) and registers with the server.
3. The server splits each evaluation into many small work shards. The worker polls `/api/worker/claim` for as many shards as it has free capacity (`WORKER_MAX_JOBS` minus the shards it is already running) and runs each as one Godot process. Multiple workers therefore share a single evaluation in proportion to their capacity.
4. It streams each shard's progress to the server (which also signals cancellation), then posts the shard result. The server merges the shards once all are done.
5. The admin Workers page shows it live and can set its name/max parallel jobs or connect/disconnect/remove it.

## Configuration (environment variables)

| Variable | Description | Default |
|---|---|---|
| `SERVER_URL` | Base URL of the web server | `http://server:5050` |
| `API_SECRET_KEY` | Must match the server's key | `dev_secret_key` |
| `WORKER_NAME` | Display name in the admin panel | hostname |
| `WORKER_MAX_JOBS` | Default parallel Godot processes (admin can override) | `4` |
| `GODOT_RELEASE_TAG` | Release to download the build from (`latest` or a tag) | `latest` |
| `GODOT_RELEASE_REPO` | GitHub repo to download the build from | `GMU-ASRC/astro-swarm` |
| `GODOT_RELEASE_ASSET` | Release asset name to download | `AstroSwarm_Linux_Server.zip` |
| `GODOT_DIR` | Where the downloaded build is unzipped | `/data/server_build` |
| `GODOT_SERVER_BIN` | Optional path to a provided binary; set to skip the download | _(unset)_ |
| `GODOT_PCK` | Path to the exported `.pck` (only when providing a binary that needs a separate pack) | _(unset)_ |
| `EVAL_TIMEOUT_SECONDS` | Max wall-clock time per job | `1800` |
| `EVAL_FIXED_FPS` | Fixed physics step for the benchmark | `60` |
| `WORKER_POLL_SECONDS` | Idle poll interval | `3` |
| `WORKER_ID_FILE` | Where the worker id is stored | `/data/worker_id` |

## Running

With Docker Compose (from `web/`): the `worker` service starts automatically. To add more compute, run a worker on another machine (each with its own data volume so it gets a stable, distinct id) pointed at the server.

Standalone: `pip install -r requirements.txt` then `python worker.py` with the variables above set. The worker downloads the dedicated-server build itself, so only `SERVER_URL` and `API_SECRET_KEY` are required.
