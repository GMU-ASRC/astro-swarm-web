# AstroSwarm Web

![Version](https://img.shields.io/badge/version-0.0.4-blue)
![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte&logoColor=white)
![TailwindCSS](https://img.shields.io/badge/TailwindCSS-4-06B6D4?logo=tailwindcss&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-6-3178C6?logo=typescript&logoColor=white)
![Flask](https://img.shields.io/badge/Flask-3.1-000000?logo=flask&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)
![Bun](https://img.shields.io/badge/Bun-1.3.14-FBF0DF?logo=bun&logoColor=black)

The companion website for AstroSwarm, a pixel-art swarm-behaviour simulator built in Godot 4. The site lets players share simulator configurations and recorded runs, view a community gallery, check the leaderboard, and download the game.

---

## Pages

### Home
Landing page with an animated starfield background, a game overview, and navigation to all other sections.

### Simulator Gallery (`/simulator`)
Browse community-uploaded recorded runs. Each card shows the species list with their colors, robot count, arena dimensions, and frame count. Clicking on a run opens a dedicated detail page featuring a streaming video player and run statistics. Runs (`.run`) are uploaded directly from Godot and parsed automatically by the backend.

### Leaderboard (`/leaderboard`)
Rankings for the Timed Local game mode showing username, completion time, and the behavior algorithm the player used. Entries link to a detail page with the full algorithm breakdown.

### Levels (`/levels`)
Per-level benchmark results for player algorithms. Each completed FARP scenario is evaluated headlessly over many trials on the dedicated server, then listed here with its entry ID, username, status, detection rate, and date. A sidebar provides a search bar (username or ID) plus filters for minimum detection rate, date range, and sort order. Clicking an entry opens a detail page with cumulative/outcome charts, a per-trial replay grid, and the defender algorithm.

### Admin CMS (`/admin`)
API-key gated management panel (client-side session stored in `localStorage`). It lists evaluations, leaderboard entries, and simulator runs with pagination, per-entry viewer pages, and a one-click ZIP export of each entry (metadata plus per-run JSON). The evaluation viewer can **re-simulate** an entry, re-running it with the current Godot build to refresh its results and replays.

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
| `POST` | `/api/evaluations` | Submit an algorithm for benchmarking (`X-API-Key` required) |
| `GET` | `/api/evaluations/<id>` | Get a single evaluation |
| `GET` | `/api/evaluations/baseline` | Average success rate across completed runs |
| `GET` | `/api/evaluations/<id>/replays` | Replay index for an evaluation |
| `GET` | `/api/evaluations/<id>/replay/<trial>` | Replay frames for one trial |
| `GET` | `/api/evaluations/<id>/chart/<kind>.png` | Rendered line/bar chart |
| `GET` | `/api/evaluations/<id>/export` | Download a ZIP of the entry and per-run JSON |
| `POST` | `/api/evaluations/<id>/resimulate` | Re-run an evaluation on the current build (`X-API-Key` required) |
| `DELETE` | `/api/evaluations/<id>` | Delete an evaluation (`X-API-Key` required) |

### Leaderboard

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/leaderboard` | Top 100 entries |
| `GET` | `/api/leaderboard/<id>` | Single entry with full algorithm |
| `GET` | `/api/leaderboard/<id>/export` | Download a ZIP of the entry |
| `POST` | `/api/leaderboard` | Submit or update a time (`X-API-Key` required) |
| `DELETE` | `/api/leaderboard/<id>` | Delete an entry (`X-API-Key` required) |

### Health

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Server health check |

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
| `API_SECRET_KEY` | Key required for write/admin endpoints | `dev_secret_key` |
| `PUBLIC_API_URL` | API base URL used by the frontend | `http://localhost:5050` |
| `GODOT_SERVER_BIN` | Path to the headless Godot dedicated-server binary used to run evaluations | _(unset)_ |
| `GODOT_PCK` | Path to the exported game `.pck` loaded by the server binary | _(unset)_ |
| `EVAL_TIMEOUT_SECONDS` | Max wall-clock time for a single evaluation run | `1800` |
| `EVAL_FIXED_FPS` | Fixed physics step the headless benchmark runs at | `60` |

In Docker the frontend is served by Flask on the same origin, so `PUBLIC_API_URL` is set to an empty string and all API requests are same-origin relative paths.

---

## Running with Docker

The `server` container expects an NVIDIA GPU to be available to generate video thumbnails efficiently using `ffmpeg`. Ensure the host machine has the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) installed.

```bash
cp .env.example .env
docker compose up -d --build
```

The server is available at `http://localhost:5050`.

## Running Locally

**Prerequisites:** You must have `ffmpeg` installed on your system to generate video thumbnails when running locally.

```bash
# Frontend
bun install
bun run dev

# Backend (in server/)
python3 -m venv venv
source ./venv/bin/activate
pip install -r requirements.txt
flask --app main run --port 5050
```

---