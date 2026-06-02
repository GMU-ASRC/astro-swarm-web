# AstroSwarm Web

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
Browse community-uploaded simulator setups and recorded runs. Each card shows the species list with their colors, robot count, arena dimensions, and frame count. Configurations (`.cfg`) and runs (`.run`) are uploaded directly from Godot and parsed automatically by the backend.

### Leaderboard (`/leaderboard`)
Rankings for the Timed Local game mode showing username, completion time, and the behavior algorithm the player used. Entries link to a detail page with the full algorithm breakdown.

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

### Simulator Configs

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/configs` | List configs (paginated) |
| `POST` | `/api/configs` | Upload a `.cfg` file |
| `GET` | `/api/configs/<id>` | Get a single config |
| `GET` | `/api/configs/<id>/download` | Download the raw `.cfg` file |

### Simulator Runs

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/runs` | List runs (paginated) |
| `POST` | `/api/runs` | Upload a `.run` file |
| `GET` | `/api/runs/<id>` | Get a single run |
| `GET` | `/api/runs/<id>/download` | Download the raw `.run` file |

### Leaderboard

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/leaderboard` | Top 100 entries |
| `GET` | `/api/leaderboard/<id>` | Single entry with full algorithm |
| `POST` | `/api/leaderboard` | Submit or update a time (`X-API-Key` required) |

### Health

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Server health check |

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
| `API_SECRET_KEY` | Key required for leaderboard POST | `dev_secret_key` |
| `PUBLIC_API_URL` | API base URL used by the frontend | `http://localhost:5050` |

In Docker the frontend is served by Flask on the same origin, so `PUBLIC_API_URL` is set to an empty string and all API requests are same-origin relative paths.

---

## Running with Docker

```bash
cp .env.example .env
docker compose up -d --build
```

The server is available at `http://localhost:5050`.

## Running Locally

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