<script lang="ts">
	import Pagination from '$lib/components/Pagination.svelte';
	import type { AdminSettings, SaveSettings } from '$lib/ts/adminSettings';

	let {
		settings,
		saving,
		onsave
	}: { settings: AdminSettings; saving: boolean; onsave: SaveSettings } = $props();

	let enemyX = $state(0);
	let enemyY = $state(0);
	let sweepMax = $state(1);
	let sweepTrials = $state(1);
	let enemyEdited = $state(false);
	let sweepEdited = $state(false);
	let seedPage = $state(1);
	const seedPageSize = 20;

	$effect(() => {
		if (enemyEdited) return;
		enemyX = settings.enemy_start_x;
		enemyY = settings.enemy_start_y;
	});

	$effect(() => {
		if (sweepEdited) return;
		sweepMax = settings.sweep_max;
		sweepTrials = settings.sweep_trials;
	});

	const enemyChanged = $derived(enemyX !== settings.enemy_start_x || enemyY !== settings.enemy_start_y);
	const sweepChanged = $derived(sweepMax !== settings.sweep_max || sweepTrials !== settings.sweep_trials);
	const pagedSeeds = $derived(
		settings.sweep_trial_seeds.slice((seedPage - 1) * seedPageSize, seedPage * seedPageSize)
	);

	async function saveEnemy(event: SubmitEvent) {
		event.preventDefault();
		const body = { enemy_start_x: enemyX, enemy_start_y: enemyY };
		if (await onsave(body, 'Saved the enemy start coordinate.')) enemyEdited = false;
	}

	async function saveSweep(event: SubmitEvent) {
		event.preventDefault();
		const body = { sweep_max: sweepMax, sweep_trials: sweepTrials };
		if (await onsave(body, 'Saved the ring sweep size.')) sweepEdited = false;
	}
</script>

<details class="legacy">
	<summary>
		<span>Single-approach benchmark</span>
		<span class="pill">Not used by current levels</span>
	</summary>

	<p class="meta">
		These values drive the original one-evader benchmark. Every benchmarked level now runs as a
		stream or assault, which picks its own spawns and uses the assault sweep shown above, so changing
		them has no effect on today's entries. They are kept for older entries and the
		<code>astrosim</code> tool.
	</p>

	<form class="settings-panel" onsubmit={saveEnemy}>
		<div class="field-row">
			<label class="field">
				<span class="field-label">Enemy start X</span>
				<input type="number" step="1" min="0" max={settings.arena_width} bind:value={enemyX} oninput={() => (enemyEdited = true)} />
			</label>
			<label class="field">
				<span class="field-label">Enemy start Y</span>
				<input type="number" step="1" min="0" max={settings.arena_height} bind:value={enemyY} oninput={() => (enemyEdited = true)} />
			</label>
			<button type="submit" disabled={saving || !enemyChanged}>Save start</button>
		</div>
		<p class="meta">
			Arena {settings.arena_width} × {settings.arena_height}, planet at ({settings.planet_x},
			{settings.planet_y}).
		</p>
	</form>

	<form class="settings-panel" onsubmit={saveSweep}>
		<div class="field-row">
			<label class="field">
				<span class="field-label">Largest ring n</span>
				<input type="number" step="1" min="1" max="1000" bind:value={sweepMax} oninput={() => (sweepEdited = true)} />
			</label>
			<label class="field">
				<span class="field-label">Trials per n</span>
				<input type="number" step="1" min="1" max="1000" bind:value={sweepTrials} oninput={() => (sweepEdited = true)} />
			</label>
			<button type="submit" disabled={saving || !sweepChanged}>Save sweep</button>
		</div>
		<p class="meta">Match cap {settings.match_cap_seconds} s, plus {settings.goal_tail_seconds} s after the goal.</p>
	</form>

	<h3>Derived seeds</h3>
	<div class="admin-table-wrap">
		<table>
			<thead>
				<tr><th>Purpose</th><th>Formula</th><th>Value</th></tr>
			</thead>
			<tbody>
				{#each settings.derived_seeds as derived (derived.name)}
					<tr>
						<td>{derived.name}</td>
						<td><code>{derived.formula}</code></td>
						<td>{derived.value}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	<h3>Per-trial sweep seeds</h3>
	<div class="admin-table-wrap">
		<table>
			<thead>
				<tr><th>Trial</th><th>Placement and orientation seed</th></tr>
			</thead>
			<tbody>
				{#each pagedSeeds as trialSeed (trialSeed.trial)}
					<tr><td>{trialSeed.trial}</td><td>{trialSeed.seed}</td></tr>
				{:else}
					<tr><td colspan="2">No seeds.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
	<Pagination bind:page={seedPage} total={settings.sweep_trial_seeds.length} pageSize={seedPageSize} />
</details>

<style>
	.legacy {
		margin-top: 1.25rem;
		border: 1px solid var(--color-line);
		background: color-mix(in srgb, var(--color-surface) 50%, transparent);
		padding: 0 1.1rem;
	}

	.legacy[open] {
		padding-bottom: 1.1rem;
	}

	summary {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.9rem 0;
		cursor: pointer;
		list-style: none;
		color: var(--color-heading);
		font-weight: 600;
	}

	summary::-webkit-details-marker {
		display: none;
	}

	summary::before {
		content: '';
		width: 0.45rem;
		height: 0.45rem;
		border-right: 2px solid var(--color-faint);
		border-bottom: 2px solid var(--color-faint);
		transform: rotate(-45deg);
		transition: transform 0.15s;
	}

	.legacy[open] summary::before {
		transform: rotate(45deg);
	}

	summary:hover::before {
		border-color: var(--color-heading);
	}

	.legacy form {
		margin-top: 0.75rem;
	}

	h3 {
		margin: 1.5rem 0 0.6rem 0;
		font-size: 0.9rem;
		color: var(--color-heading);
	}
</style>
