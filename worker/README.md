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
`out/<entry-id>/results.json` plus the charts.

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

Given a base seed, the derivations are fixed, and the server publishes the same formulas
under `derived_seeds` on that settings endpoint:

| Use | Seed |
|---|---|
| Enemy spawn ring | `seed` |
| Placement match `trial` | `seed + trial` |
| Ring-sweep trial | `seed + 100000 + trial * 1000000` |
| Ring-sweep match at `n` | `sweep_seed[trial] + 500000 + n` |
| Level 2 scatter fallback | `seed + 700000` |
| Assault trial | `seed + 1100000 + trial` |
| Assault spawn bearings | `trial seed + 4400000`, drawn fresh for every launch |
| Assault defender sweep at `n` | `seed + 3300000 + n * 1000000 + trial` |

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

The server's default is a single fixed evader spawn for the whole ring sweep (`--enemy-x`,
`--enemy-y`, default `1920, 40`) reused by every trial at every n. The only thing that varies across the
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

Making the site agree means flipping `EVAL_SWEEP_SPAWN` to `varied` on the server, which
invalidates every existing benchmarked entry until it is re-simulated.

## Ring crowding

The sweep ring has a fixed radius of 300, about 1885px of circumference. Past roughly n=100
the defenders sit closer together than their own hulls. With collisions off (the default)
they simply overlap, which is what the game does too; with `-collisions` they
shove each other off the ring and the layout stops being a ring at all.

## Assault benchmarks (levels 3, 4 and 5)

Levels 3 to 5 are not single-evader matches, so they take a different path
(`internal/bench/assault.go` and `internal/bench/assaultrunner.go`). A stream of evaders comes
at one scattered line, and the score is the share of them the line destroyed rather than a
single win or loss.

| Level | Mode | Arrival | Attrition | Ends when |
|---|---|---|---|---|
| 3 | Waves | One at a time off the ring, a fresh random bearing each time | No | The clock stops |
| 4 | Waves | The same | Yes | The line is spent, or the clock stops |
| 5 | Siege | Five at once, spread around the arena border | Yes | Every evader is destroyed or has reached the planet |

In waves mode the next evader launches `WaveGapFrames` after the last one resolves, so a run
sends as many as the clock allows. **This is on purpose longer than the level.** The game caps
levels 3 and 4 at five waves, which is enough for a player to feel out an algorithm without
sitting through a four-minute run; the benchmark keeps going until the line is spent or the
clock stops, so a defense that only survives the opening does not score as though it had held.
An entry's published rate is therefore taken over far more evaders than the player watched. In siege mode `SiegeEvaders` launch together at
`SiegeEdgePoint` bearings, which walk out from the planet until they meet the arena border, so
the ones on the long sides arrive noticeably later than the ones above and below.

Attrition removes the defender that made the capture. In waves mode that is also the end
condition: once the last defender is gone the run stops, since nothing is left to send the next
wave at. An evader that reaches the planet is counted as a breach and removed, and the run
carries on either way.

Every trial gets its own seeded defender scatter and its own seeded spawn bearings, so the same
entry and seed always reproduce the same result.

The report keeps the combined outcome arrays the existing charts read, and adds
`evaders_resolved`, `evaders_destroyed`, `evader_destroyed_rate`, `breaches`, `defenders_lost`,
`risk`, `trials_held_rate`, the per-trial arrays (`trial_destroyed`, `trial_resolved`,
`trial_breaches`, `trial_lost`) the site charts, `attrition`, `sweep_attrition`, and
`sweep_progress`.

The rate is taken over the evaders that actually reached a verdict, not every one sent: an
evader still in flight when the clock stopped was never given the chance to be stopped, so it
counts for neither side. A run's `stats` carries both numbers, as `sent` and `resolved`, and
`destroyed + breaches` always equals `resolved`.

`success_rate` on an assault level is the share of evaders destroyed, not the share of trials
held outright: stopping nine of ten evaders is not the same result as stopping none. The strict
all-or-nothing number is reported alongside it as `trials_held_rate`.

### Risk and the attrition curve

`risk` is `100 - capture success rate`, reported both as a headline number and per sweep point,
so a curve of risk against ring size drops straight out of the same data.

`attrition` pools every launched evader by **how many defenders were standing when it left**,
and reports the risk the line carried at that size. On level 3 the line never thins, so the
curve is a single point; on levels 4 and 5 it is the shape the request asks for — a line of ten
that trades down to five carries a visibly different risk, and a strategy change shows up as a
different curve rather than a different single number.

`sweep_attrition` is the same measurement taken across the ring sweep instead of the submitted
layout: one curve per ring size, so a line that *started* at n=10 and one that started at n=5
can be compared on what their risk does as each thins. A ring that never lost a ship has a
single rung and is left out, which is every ring on level 3.

`sweep_progress` reads each ring size **wave by wave rather than in total**. For every ring
size it walks its trials in launch order and reports, at each evader faced, the cumulative
capture rate, the risk that leaves, and how much of the line was still standing. That is what
gives all three ring-sweep charts a line per `n` on levels 3, 4 *and* 5 — including level 3,
where the line never thins and the attrition curve is flat by definition. Each rung averages
only the trials that got that far, and the curve stops once fewer than half of them are still
running, which is where the average stops meaning anything.

Every chart that draws a line per `n` draws all of them, colored along a ramp from blue at the
smallest ring to red at the largest, so a line's color alone says where it sits in the sweep.
A legend cannot name forty lines, so the PNGs name an even spread of eight of them: they are
anchors on the ramp, and any line can be placed by the two named ones its color sits between.
On the site every line is named, and clicking a legend entry pulls that ring out of the group.
The site, the server PNGs and the worker's own charts all pick their colors the same way.

Replay frames carry a fixed slot per ship for the whole trial, dead ships included as `-1`, so
a defender lost to a trade does not shift every slot after it. In waves mode there is one
evader slot, since only one is ever in flight; in siege mode there is one per evader. These
runs are far longer than a single approach, so only every `AssaultRecordStride` physics frame
is recorded (the job's replay `fps` is set to match) and only the first `AssaultReplayTrials`
trials keep a recording at all. The rest are graded but not replayable, and the site marks
their cells accordingly.

### Running an assault entry

`astrosim` re-simulates level 3, 4 and 5 entries the same way it does levels 1 and 2:

```
./astrosim 0940aa30-d79c-4406-9fcb-002a547413f2
```

The banner, the progress line and the summary switch to assault wording, and the summary
reports the capture success rate, the risk, how many evaders got through, the defenders lost
and the risk at each line size. `-n-max` becomes a ceiling rather than a target, since the
sweep stops as soon as the algorithm holds, and the sweep curves are left out of the pass/fail
verdict for the same reason: a published entry and a fresh run can stop at different `n`.

These levels carry their own sweep budget: `DefaultAssaultSweepMax` x `DefaultAssaultSweepTrials`
(40 x 20), which mirrors the server's `assault_sweep_max` and `assault_sweep_trials`. An entry
on one of them adopts that budget rather than the level 1 and 2 one, whether the settings
endpoint reports it, is silent about it, or is not consulted at all under `-no-settings`. The
level 1 and 2 figures are 100 x 100, which on an assault level would be tens of thousands of
matches at ring sizes up to a hundred defenders.

### Adaptive defender sweep

The ring sweep for these levels does not run out to a fixed `n-max`. It starts at one
defender and grows, running `sweep-trials` matches at each count, and stops as soon as **three
consecutive defender counts come back at 100 percent**. `n-max` is only a ceiling.

The point is that simulating out to n=100 is wasted work once an algorithm is clearly holding.
A strong entry finishes in a handful of steps; a weak one keeps climbing until it either holds
or hits the ceiling. `WaveConsecutiveMax` in `internal/bench/assault.go` is the run length that
counts as settled.

Level 6 and Level 7 entries are piloted recordings rather than simulations, so `astrosim` refuses them.

## Charts

Written with gonum/plot into the output directory:

| File | What it shows |
|---|---|
| `detection_rate.png` | Detection, capture, and defender-win rate over the placement trials, simulated against published |
| `detection_rate_vs_defenders.png` | Detection rate against the number of defenders on the ring |
| `capture_rate_vs_defenders.png` | Capture success rate against the number of defenders on the ring |
| `rates_vs_defenders.png` | Both rates on one axis |
| `risk_vs_defenders.png` | `Risk = 1 - capture success rate` against the number of defenders, with the capture rate alongside it |
| `risk_vs_attrition.png` | Risk against how many defenders were still standing when the evader launched, over the submitted layout; only drawn when the line actually thinned |
| `risk_vs_attrition_by_ring.png` | The same curve taken across the ring sweep, one line per ring size |
| `capture_rate_by_ring.png` | Cumulative capture success rate against evaders faced, one line per ring size |
| `risk_by_ring.png` | The risk that leaves, against evaders faced, one line per ring size |
| `attrition_by_ring.png` | Defenders still standing as a share of the ring, against evaders faced, one line per ring size |

The published overlay on the two sweep charts comes from `/api/evaluations/<id>/sweep-replays`,
which carries a real `detection_rate` and `capture_rate` for every `n`. Prefer it over
`results.sweep`: `merge.py` writes `success_rate` there as a copy of the capture rate, so an
older entry can carry a `success_rate` that is not a win rate at all. Plotting that opposite a
detection rate compares seeing against touching and makes the algorithm look broken when it is
not. Newer entries also carry `capture_rate`, `detection_rate` and `risk` on each sweep point,
which is what the site's risk chart reads.

## How it maps onto the game

The game used to carry its own headless benchmarker in `BenchBase.gd`. That is gone: grading
happens here now, and the game only plays the level. What the worker ports is the rule set
those level scripts implement.

| Game | Here |
|---|---|
| `FARPBase.gd` match loop and event tracking | `internal/bench/match.go` |
| `AssaultBase.gd` and `WaveBase.gd` evader stream, attrition, breaches | `internal/bench/assault.go` |
| `Spaceship.gd` movement, vision, conditions, actions | `internal/sim` |
| `BlockExecutor.gd` | `internal/blocks/executor.go` |
| `SimulationManager.normalize_to_scripts`, `ship_config_from_scripts` | `internal/blocks/normalize.go`, `config.go` |
| Godot's `RandomNumberGenerator` and the global `seed()`/`randf()` | `internal/godot/rng.go` |
| `Vision` Area2D cone against a `Hull` CircleShape2D | `internal/geom/polygon.go` |
| `web/server/merge.py` shard merge | `internal/bench/runner.go` |

Every match seeds its own RNG from the table above (`seed + trial` for placement runs,
`sweep_trial_seed(trial) + 500000 + n` for the sweep), so matches are independent and run in
parallel without changing the result.

The headline rates follow `merge.py`: on levels 1 and 2 the success rate is counted over the
trials that actually ran, on levels 3 to 5 it is the share of resolved evaders destroyed, and
the per-n sweep number is the capture rate. That is what the site displays.

## Known sources of drift against the game

These are faithful ports of what the game does, quirks included. They are listed because they
are the likely reasons a published entry looks slightly off from what the player watched.

1. **Two different detection tests.** The `Detected` event is a point test against
   `view_distance` and `fov_degrees`, but the `When I see an enemy` block is driven by the
   `Vision` Area2D cone overlapping the target's `Hull` circle. A defender can therefore
   react to an evader a frame or two before the run is marked as detected, or the other way
   round.
2. **The hull shape never resizes.** `Set size` changes `hull_radius`, which is what the
   capture test uses, but the `CollisionShape2D` in `Spaceship.tscn` stays at radius 9, which
   is what vision uses. Both behaviors are reproduced.
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
