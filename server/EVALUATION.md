# Headless algorithm evaluation

When a player clears the FARP scenario in-game, the client POSTs their defender
algorithm to `POST /api/evaluations`. The server runs the game **headlessly** to
benchmark that algorithm across swarm sizes and stores a success-rate-per-N curve,
which the website renders on `/players/<player_id>`.

## Pieces

- `game/autoloads/BenchRunner.gd` — when launched with `--bench` (or as a
  dedicated-server export, feature `dedicated_server`), it simulates the FARP
  scenario for `N = 1..nmax` defenders, `trials` runs each, and writes
  `{"results": [{"n", "success_rate"}, ...]}` to `--out` (and stdout), then quits.
- `server/evaluator.py` — runs the Godot binary in a background thread.
- `server/routers/evaluations.py` — `POST /api/evaluations` (X-API-Key),
  `GET /api/evaluations` (players), `GET /api/evaluations/<player_id>` (latest),
  `GET /api/evaluations/baseline` (community average).

## Provide the server build

Export an AstroSwarm **Linux dedicated server** build (the `Linux Server` preset
in `game/export_presets.cfg` is set up for this) and drop it into
`web/server_build/` so the Dockerfile copies it into the image:

```
web/server_build/astroswarm.x86_64   # dedicated-server binary (loads its own .pck)
web/server_build/astroswarm.pck       # only if you export a plain binary + pack
```

The Dockerfile no longer downloads Godot — it uses your build directly.

## Run

```bash
cd web && docker compose up --build
```

Env vars (see `docker-compose.yml`):
- `GODOT_SERVER_BIN` — default `/app/server_build/astroswarm.x86_64`.
- `GODOT_PCK` — leave empty for a dedicated-server binary; set it only if
  `GODOT_SERVER_BIN` is a plain Godot binary needing `--main-pack`.
- `EVAL_TIMEOUT_SECONDS` — default 1800.

## Local smoke test

```bash
./astroswarm.x86_64 --headless -- --bench --trials=5 --nmax=8 --out=/tmp/farp.json
cat /tmp/farp.json
```
