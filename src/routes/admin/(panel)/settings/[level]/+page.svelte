<script lang="ts">
	import { apiUrl } from '$lib/ts/api';
	import Pagination from '$lib/components/Pagination.svelte';
	import { isWave as waveLevel } from '$lib/ts/levels';

	let { data } = $props();
	let settings = $state<any>(null);
	let loading = $state(true);
	let enemyX = $state(0);
	let enemyY = $state(0);
	let sweepMax = $state(100);
	let sweepTrials = $state(10);
	let seed = $state(0);
	let saving = $state(false);
	let message = $state('');

	const levelNum = $derived.by(() => {
		const m = (data.levelSlug ?? '').match(/\d+/);
		return m ? parseInt(m[0], 10) : 1;
	});
	const levelId = $derived(`farp${levelNum}`);
	const levelInfo = $derived((settings?.levels ?? []).find((l: any) => l.id === levelId) ?? null);
	const isPilot = $derived((settings?.pilot_level_ids ?? []).includes(levelId));
	const isWave = $derived(waveLevel(levelNum));
	const runLimits = $derived(
		settings?.pilot_run_limits?.[levelId] ?? {
			max_seconds: settings?.pilot_time_limit_seconds,
			max_ships: 6
		}
	);

	let seedPage = $state(1);
	const seedPageSize = 20;
	const seeds = $derived(settings?.sweep_trial_seeds ?? []);
	const pagedSeeds = $derived(seeds.slice((seedPage - 1) * seedPageSize, seedPage * seedPageSize));

	$effect(() => {
		let active = true;
		loading = true;
		data.settingsPromise.then((row) => {
			if (!active) return;
			settings = row;
			if (row) {
				enemyX = row.enemy_start_x;
				enemyY = row.enemy_start_y;
				sweepMax = row.sweep_max;
				sweepTrials = row.sweep_trials;
				seed = row.seed;
			}
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	async function save(body: Record<string, unknown>, successMessage: string) {
		saving = true;
		message = '';
		try {
			const res = await fetch(apiUrl('/api/evaluations/settings'), {
				method: 'PUT',
				headers: { 'X-API-Key': data.adminKey, 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});
			if (!res.ok) {
				message = `Failed to save: ${res.status}`;
				return;
			}
			const updated = await res.json();
			enemyX = updated.enemy_start_x;
			enemyY = updated.enemy_start_y;
			if (updated.sweep_max != null) sweepMax = updated.sweep_max;
			if (updated.sweep_trials != null) sweepTrials = updated.sweep_trials;
			if (updated.seed != null) seed = updated.seed;
			if (settings) {
				settings.enemy_start_x = updated.enemy_start_x;
				settings.enemy_start_y = updated.enemy_start_y;
				if (updated.sweep_max != null) settings.sweep_max = updated.sweep_max;
				if (updated.sweep_trials != null) settings.sweep_trials = updated.sweep_trials;
				if (updated.sweep_trial_seeds != null) settings.sweep_trial_seeds = updated.sweep_trial_seeds;
				if (updated.seed != null) settings.seed = updated.seed;
				if (updated.derived_seeds != null) settings.derived_seeds = updated.derived_seeds;
				if (updated.levels != null) settings.levels = updated.levels;
			}
			message = successMessage;
		} catch (err) {
			message = `Save failed: ${err}`;
		} finally {
			saving = false;
		}
	}

	function saveEnemyStart() {
		save({ enemy_start_x: enemyX, enemy_start_y: enemyY }, `Saved enemy start coordinate.`);
	}

	function saveSweep() {
		save({ sweep_max: sweepMax, sweep_trials: sweepTrials }, `Saved ring sweep settings.`);
	}

	function saveSeed() {
		save({ seed }, `Saved base seed. Existing evaluations were run under the previous seed.`);
	}

	function regenerateSeeds() {
		if (!confirm('Draw a new random base seed? Every derived seed changes, so new evaluations are no longer comparable with existing ones unless you re-simulate them.')) return;
		save({ regenerate_seeds: true }, `Regenerated all seeds.`);
	}

	function toggleLevel(enabled: boolean) {
		save({ level_id: levelId, enabled }, `Saved level settings.`);
	}
</script>

<p><a href="/admin/settings">← Settings</a></p>
<h1>Level {levelNum} {isPilot ? 'render settings' : 'benchmark settings'}</h1>
<p class="meta">
	{levelInfo?.name ?? levelId} — {isPilot
		? 'pilot level: the player flies the evader themselves against the best submitted level 2 algorithm. Nothing is simulated for this level, so there are no benchmark parameters to tune.'
		: isWave
			? 'wave level: the submitted algorithm drives the defenders against a wave of evaders, and a run only counts as held if every one of them is destroyed.'
			: 'defense level (the submitted algorithm drives the defenders against the incoming evader).'}
</p>

{#if loading}
	<p>Loading settings...</p>
{:else if !settings}
	<p>Could not load settings.</p>
{:else}
	{#if message}<div class="message">{message}</div>{/if}

	<h2>Level status</h2>
	<p class="meta">A disabled level rejects new submissions.</p>
	<div class="actions">
		<label class="toggle">
			<input
				type="checkbox"
				checked={levelInfo?.enabled ?? true}
				onchange={(e) => toggleLevel(e.currentTarget.checked)}
				disabled={saving}
			/>
			<span>{(levelInfo?.enabled ?? true) ? 'Enabled' : 'Disabled'}</span>
		</label>
	</div>

	{#if isPilot}
		<h2>How a level {levelNum} entry is processed</h2>
		<p class="meta">
			A submitted run was already flown in the game client, so nothing is simulated for this level.
			The server queues the entry as a single render job; a worker claims it and packs the recorded
			trajectory into a watchable replay. The event times come from the run itself.
		</p>
		<div class="stat-grid">
			<div class="stat"><div class="label">Simulated trials</div><div>None</div></div>
			<div class="stat"><div class="label">Ring sweep</div><div>None</div></div>
			<div class="stat"><div class="label">Work per entry</div><div>1 render job</div></div>
			<div class="stat">
				<div class="label">Scenario</div>
				<div>
					{levelNum === 6
						? 'Two milling swarms and one player-flown leader'
						: 'Best submitted farp2 algorithm'}
				</div>
			</div>
		</div>

		<h2>Run limits</h2>
		<p class="meta">Enforced when a run is submitted — a run past these limits is rejected, so nobody can upload an hour-long flight.</p>
		<div class="stat-grid">
			<div class="stat"><div class="label">Time limit</div><div>{runLimits.max_seconds}s</div></div>
			<div class="stat"><div class="label">Max record rate</div><div>{settings.pilot_max_fps} fps</div></div>
			<div class="stat"><div class="label">Max recorded ships</div><div>{runLimits.max_ships}</div></div>
			<div class="stat"><div class="label">XP for a finished run</div><div>{settings.pilot_max_xp}</div></div>
		</div>
	{:else}
	{#if isWave}
		<h2>How a level {levelNum} entry is benchmarked</h2>
		<p class="meta">
			A wave level is not graded against a single evader. Every trial sends several evaders in off
			the ring and only counts as held if all of them are destroyed before any reaches the planet.
			{levelNum === 4
				? 'A capture here destroys the defender as well as the evader, so the line thins as the wave goes on.'
				: 'A defender that catches an evader survives and carries on to the next.'}
			The run always plays out until every evader has been sent and resolved, so one getting through
			early does not cut the trial short.
		</p>
		<div class="stat-grid">
			<div class="stat">
				<div class="label">Trials</div>
				<div>{settings.wave_trials ?? 100}</div>
			</div>
			<div class="stat">
				<div class="label">Wave styles</div>
				<div>Sequential and simultaneous, split evenly</div>
			</div>
			<div class="stat">
				<div class="label">Evaders per wave</div>
				<div>{settings.wave_evaders ?? 3}, never more than the defenders</div>
			</div>
			<div class="stat">
				<div class="label">Defender sweep</div>
				<div>
					Grows until {settings.wave_clean_streak ?? 3} consecutive clean counts, capped at n={settings.wave_sweep_max ?? 40}
					x {settings.wave_sweep_trials ?? 20} trials
				</div>
			</div>
		</div>
		<p class="meta">
			Each trial gets its own seeded defender scatter and spawn angles, so the same entry and seed
			always reproduce the same result. A wave match runs two full waves, so these levels use their
			own sweep budget rather than the ring-sweep one, which would put a job past the worker
			timeout. The sweep stops as soon as the algorithm holds, so the cap is only a ceiling.
		</p>
	{/if}

	<p class="meta">The benchmark parameters below are shared across all benchmarked levels so every entry is tested under identical conditions.</p>

	<h2>Enemy start coordinate</h2>
	<p class="meta">Where the enemy ship starts for all ring sweep runs. The placement runs use their own seeded per-trial enemy spawns.</p>
	<div class="stat-grid">
		<div class="stat"><div class="label">Arena size</div><div>{settings.arena_width} × {settings.arena_height}</div></div>
		<div class="stat"><div class="label">Target (planet) center</div><div>{settings.planet_x}, {settings.planet_y}</div></div>
		<div class="stat"><div class="label">Valid X range</div><div>0 – {settings.arena_width}</div></div>
		<div class="stat"><div class="label">Valid Y range</div><div>0 – {settings.arena_height}</div></div>
	</div>
	<div class="actions">
		<label for="enemyX">X</label>
		<input id="enemyX" type="number" step="1" min="0" max={settings.arena_width} bind:value={enemyX} style="width:6rem" />
		<label for="enemyY">Y</label>
		<input id="enemyY" type="number" step="1" min="0" max={settings.arena_height} bind:value={enemyY} style="width:6rem" />
		<button onclick={saveEnemyStart} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
	</div>

	<h2>Ring sweep</h2>
	<p class="meta">n is the largest defender ring tested (each run sweeps 1 to n). n2 is how many times each n is simulated; the runs are averaged for the rate graphs and only one is kept for replay. Changing these takes effect on the next submitted or re-simulated evaluation.</p>
	<div class="actions">
		<label for="sweepMax">n (max defenders)</label>
		<input id="sweepMax" type="number" step="1" min="1" max="1000" bind:value={sweepMax} style="width:6rem" />
		<label for="sweepTrials">n2 (trials per n)</label>
		<input id="sweepTrials" type="number" step="1" min="1" max="1000" bind:value={sweepTrials} style="width:6rem" />
		<button onclick={saveSweep} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
	</div>

	<h2>Base seed</h2>
	<p class="meta">
		Every other seed is derived from this one: the enemy spawn points, the per-trial ring rotation
		that decides where the defenders sit, their orientations, and each match's RNG. Set it to a
		specific value to reproduce a past benchmark, or regenerate to draw a new random one.
		Regenerating changes every derived seed, so evaluations already in the database were graded
		under different conditions until they are re-simulated. Takes effect on the next submitted or
		re-simulated evaluation.
	</p>
	<div class="actions">
		<label for="seed">Seed</label>
		<input id="seed" type="number" step="1" min="1" max="2147483647" bind:value={seed} style="width:10rem" />
		<button onclick={saveSeed} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
		<button onclick={regenerateSeeds} disabled={saving}>Regenerate all seeds</button>
	</div>

	<h2>Per-trial seeds</h2>
	<p class="meta">
		Spaceship placement and orientation seeds — one per trial (n2 = {settings.sweep_trials}).
		Each of the n2 ring-sweep repeats for a given n uses its own seed, so the repeats place the ring
		at a different rotation but stay reproducible. Ring size n uses <code>seed + n</code>.
	</p>
	<div class="admin-table-wrap">
		<table>
			<thead>
				<tr><th>Trial</th><th>Placement / orientation seed</th></tr>
			</thead>
			<tbody>
				{#each pagedSeeds as s}
					<tr><td>{s.trial}</td><td>{s.seed}</td></tr>
				{:else}
					<tr><td colspan="2">No seeds.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
	<Pagination bind:page={seedPage} total={seeds.length} pageSize={seedPageSize} />

	<h2>Simulation</h2>
	<div class="admin-table-wrap">
		<table>
			<tbody>
				<tr><th>Base seed</th><td>{settings.seed}</td></tr>
				<tr><th>Sweep range (n)</th><td>1 to {settings.sweep_max}</td></tr>
				<tr><th>Trials per sweep n</th><td>{settings.sweep_trials}</td></tr>
				<tr><th>Match time cap</th><td>{settings.match_cap_seconds}s</td></tr>
				<tr><th>Extra time after T_goal</th><td>{settings.goal_tail_seconds}s</td></tr>
				<tr><th>Sweep enemy start</th><td>({settings.enemy_start_x}, {settings.enemy_start_y})</td></tr>
			</tbody>
		</table>
	</div>

	<h2>Static seeds</h2>
	<div class="admin-table-wrap">
		<table>
			<thead>
				<tr>
					<th>Purpose</th>
					<th>Formula</th>
					<th>Value</th>
				</tr>
			</thead>
			<tbody>
				{#each settings.derived_seeds as seed}
					<tr>
						<td>{seed.name}</td>
						<td><code>{seed.formula}</code></td>
						<td>{seed.value}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	{/if}
{/if}
