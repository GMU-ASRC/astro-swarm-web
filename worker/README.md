# AstroSwarm Simulator

A Go port of the FARP benchmarker (`game/simulations/farp/`), with the rendering and the
scene tree taken out. It ships two binaries:

- **`astrosim`** — re-runs a published entry on your machine and checks the numbers the
  website is showing for it. This file.
- **`astroworker`** — a drop-in replacement for the Python evaluation worker and the headless
  Godot build it drives. See `WORKER.md`.

## Building

```
cd worker
go mod tidy
go build -o astrosim ./cmd/astrosim
go build -o astroworker ./cmd/astroworker
```

## Running

Point it at an entry id, or at the page url for that entry:

```
./astrosim 13569541-180c-4c51-bcfe-d4ab038359af
./astrosim https://astroswarm.autonomousrobotics.club/levels/13569541-180c-4c51-bcfe-d4ab038359af
```

It fetches the entry from `/api/evaluations/<id>`, simulates the placement trials and the
ring sweep, prints a comparison against the published rates, and writes
`out/<entry-id>/results.json` plus four charts.

To work offline, save the evaluation json (the `Export` button on the entry, or a raw
`{"algorithm": [...], "placements": [...]}` file) and pass it with `-file`:

```
./astrosim -file evaluation.json -out ./run1
```

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `-server` | the production site | Base url used when the argument is a bare id |
| `-file` | | Read the entry from a local json file instead of the server |
| `-out` | `out/<entry id>` | Where `results.json` and the charts go |
| `-trials` | the entry's own count | Placement trials to simulate |
| `-seed` | `987654321` | Evaluation seed; must match the server's `eval_seed` setting |
| `-n-max` | `100` | Largest defender count in the ring sweep |
| `-sweep-trials` | `100` | Trials per ring-sweep defender count (n2) |
| `-match-seconds` | `240` | Match time cap |
| `-goal-tail-seconds` | `3` | Extra time simulated after the evader reaches the planet |
| `-enemy-x`, `-enemy-y` | `1920`, `40` | Ring-sweep evader spawn, used only by `-sweep-spawn=fixed` |
| `-sweep-spawn` | `varied` | `varied` stratifies the evader's approach angle per trial; `fixed` pins it the way the server does |
| `-collisions` | `false` | Ship and planet collisions |
| `-single-precision` | `false` | Round positions and headings to float32 the way Godot stores them |
| `-workers` | cpu count | Parallel matches |
| `-tolerance` | `1.0` | Percentage points a published rate may drift before it is flagged |
| `-no-charts` | `false` | Skip the pngs |
| `-no-settings` | `false` | Do not read the seed and sweep settings from the server |

## Seeds

The seed decides every enemy spawn angle and every ring orientation, so a run against the
wrong seed proves nothing. On startup `astrosim` reads `/api/evaluations/settings` (a public
endpoint) and adopts the live `seed`, `sweep_trials`, `match_cap_seconds`,
`goal_tail_seconds`, and evader spawn from it. Anything you pass explicitly on the command
line wins; everything else follows the server. It prints which values it adopted.

`-no-settings` skips the fetch and falls back to the built-in defaults, which are the
values in `web/server/config.py` rather than whatever is in the server's database.

Given a base seed, the derivations match `BenchBase.gd` exactly, and the server publishes
the same four formulas under `derived_seeds` on that settings endpoint:

| Use | Seed |
|---|---|
| Enemy spawn ring | `seed` |
| Placement match `trial` | `seed + trial` |
| Ring-sweep trial | `seed + 100000 + trial * 1000000` |
| Ring-sweep match at `n` | `sweep_seed[trial] + 500000 + n` |
| Level 2 scatter fallback | `seed + 700000` |
| Wave trial, sequential | `seed + 1100000 + trial` |
| Wave trial, simultaneous | `seed + 2200000 + trial` |
| Wave defender sweep at `n` | `seed + 3300000 + n * 1000000 + trial` |

**The seed a given entry was graded with is not recorded.** `PlayerEvaluation` has no seed
column, and `_shard_payload` in `web/server/routers/workers.py` reads the current global
setting when a worker claims a shard. If anyone hits **Regenerate seeds** in the admin
panel, every entry benchmarked before that point becomes unreproducible - the settings
endpoint will hand `astrosim` a seed that entry never ran with, and the comparison will
report a mismatch that is not the algorithm's fault. Storing `seed` on the evaluation row
at queue time would fix it.

## Sample size

Each ring-sweep rate is a fraction of n2 runs, so n2 sets how finely a rate can land:
n2 = 10 can only produce multiples of 10 and scatters roughly plus or minus 15 points
around the true value, while n2 = 100 resolves to 1 point. Comparing a curve built from
10 samples against one built from 100 looks like a disagreement between the two
simulators when it is only sampling noise.

The default here is 100 to match the deployed server. `verify` also infers n2 back out of
the granularity of the published rates and warns when it disagrees with what was simulated,
so a mismatch is reported rather than left to the eye.

## Ring-sweep evader spawn

`BenchBase.gd` gives the ring sweep a single fixed evader spawn (`--enemy-x`, `--enemy-y`,
default `1920, 40`) reused by every trial at every n. The only thing that varies across the
n2 repeats is the ring's rotation and each defender's heading, so the evader always
approaches from the same bearing. At n=1 that measures one geometry rather than a
distribution, and a defender whose vision reaches the ring can score close to 100 percent
detection off a single lucky arrangement.

`-sweep-spawn=varied` (the default here) instead gives each trial its own approach angle,
using the same stratified ring the placement runs use: trial `t` of n2 spawns at radius 1000
and angle `(t + 0.2 + randf() * 0.6) / n2 * TAU`, seeded from `seed + 300000`. The same n2
angles are reused at every n, so defender count stays the only variable across the curve.

This deliberately diverges from the server. `verify` reports the sweep deltas but leaves
them out of the pass/fail verdict while `varied` is active, since the two are no longer
measuring the same thing. Use `-sweep-spawn=fixed` to reproduce a published entry.

Making the site agree means the same change in `BenchBase._start_match`, which invalidates
every existing benchmarked entry until it is re-simulated.

## Ring crowding

The sweep ring has a fixed radius of 300, about 1885px of circumference. Past roughly n=100
the defenders sit closer together than their own hulls. With collisions off (the default)
they simply overlap, which is what the Godot benchmarker does too; with `-collisions` they
shove each other off the ring and the layout stops being a ring at all.

## Wave benchmarks (levels 3 and 4)

Levels 3 and 4 are not single-evader matches, so they take a different path
(`internal/bench/wave.go` and `internal/bench/waverunner.go`). Several evaders come in off the
same ring and the run is a win only if every one of them is destroyed before any reaches the
planet. Level 3 removes the captured evader; Level 4 removes the defender that caught it as
well, so the line thins as the wave goes on. Running out of defenders ends the run.

Each entry is graded twice over, once per wave style:

| Wave style | Behaviour |
|---|---|
| Sequential | The next evader launches only once the previous one is gone |
| Simultaneous | Every evader is in the air from the first frame |

Each style runs the full trial count (100 by default). Every trial gets its own seeded
defender scatter, its own seeded spawn angles, and an evader count of `1 + trial % defenders`,
so counts vary across the run and never exceed the defender count. The two styles use separate
seed offsets, so they are independent but both reproducible: the same entry and seed always
produce the same result.

The report keeps the combined outcome arrays the existing charts read, and adds
`sequential_rate`, `simultaneous_rate`, `evaders_destroyed`, `evaders_total` and
`evader_destroyed_rate`.

### Adaptive defender sweep

The ring sweep for these levels does not run out to a fixed `n-max`. It starts at one
defender and grows, running `sweep-trials` matches at each count with alternating wave styles
and varying evader counts, and stops as soon as **three consecutive defender counts come back
at 100 percent**. `n-max` is only a ceiling.

The point is that simulating out to n=100 is wasted work once an algorithm is clearly holding.
A strong entry finishes in a handful of steps; a weak one keeps climbing until it either holds
or hits the ceiling. `WaveConsecutiveMax` in `internal/bench/wave.go` is the run length that
counts as settled.

Level 5 and Level 6 entries are piloted recordings rather than simulations, so `astrosim` refuses them.

## Charts

Written with gonum/plot into the output directory:

| File | What it shows |
|---|---|
| `detection_rate.png` | Detection, capture, and defender-win rate over the placement trials, simulated against published |
| `detection_rate_vs_defenders.png` | Detection rate against the number of defenders on the ring |
| `capture_rate_vs_defenders.png` | Capture success rate against the number of defenders on the ring |
| `rates_vs_defenders.png` | Both rates on one axis |

The published overlay on the two sweep charts comes from `/api/evaluations/<id>/sweep-replays`,
which carries a real `detection_rate` and `capture_rate` for every `n`. It does **not** come
from `results.sweep`: that field holds only one number per `n`, and `merge.py` fills it with
the capture rate under the name `success_rate`. Plotting it opposite a detection rate
compares seeing against touching and makes the algorithm look broken when it is not.

## How it maps onto the Godot benchmarker

| Godot | Here |
|---|---|
| `BenchBase.gd` match loop, event tracking, ring sweep | `internal/bench` |
| `Spaceship.gd` movement, vision, conditions, actions | `internal/sim` |
| `BlockExecutor.gd` | `internal/blocks/executor.go` |
| `SimulationManager.normalize_to_scripts`, `ship_config_from_scripts` | `internal/blocks/normalize.go`, `config.go` |
| Godot's `RandomNumberGenerator` and the global `seed()`/`randf()` | `internal/godot/rng.go` |
| `Vision` Area2D cone against a `Hull` CircleShape2D | `internal/geom/polygon.go` |
| `web/server/merge.py` shard merge | `internal/bench/runner.go` |

Every match seeds its own RNG exactly the way `BenchBase._start_match` does
(`seed + trial` for placement runs, `sweep_trial_seed(trial) + 500000 + n` for the sweep),
so matches are independent and run in parallel without changing the result.

The headline rates follow `merge.py`, not `BenchBase._write_output`: success rate is
counted over the trials that actually ran, and the per-n sweep number is the capture rate.
That is what the site displays.

## Known sources of drift against the Godot build

These are faithful ports of what the Godot benchmarker does, quirks included. They are
listed because they are the likely reasons a published entry looks slightly off.

1. **Two different detection tests.** The `Detected` event is a point test against
   `view_distance` and `fov_degrees`, but the `When I see an enemy` block is driven by the
   `Vision` Area2D cone overlapping the target's `Hull` circle. A defender can therefore
   react to an evader a frame or two before the run is marked as detected, or the other way
   round.
2. **The hull shape never resizes.** `Set size` changes `hull_radius`, which is what the
   capture test uses, but the `CollisionShape2D` in `Spaceship.tscn` stays at radius 9, which
   is what vision uses. Both behaviours are reproduced.
3. **The first replay frame is recorded twice.** `_start_match` snapshots the initial state
   and the first `_physics_process` snapshots it again before anything has moved.
4. **Vision lags one frame.** Godot's area signals land after the physics step, so a ship
   acts on what it could see at the end of the previous frame. Reproduced.
5. **Float precision.** Godot stores positions and rotations as 32-bit floats; this runs in
   double precision. `-single-precision` narrows to float32 at the same points Godot does,
   which is the closer comparison when a run is near a boundary.
6. **Godot's RNG.** `internal/godot/rng.go` is Godot's PCG32 with the same default
   increment and the same `seed()` semantics, and draws doubles from the full 32-bit range.
   If a future Godot release changes `randf()` precision, sweeps that depend on
   `Random walk` will diverge.
