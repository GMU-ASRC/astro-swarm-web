<script lang="ts">
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();
	let settings = $state<any>(null);
	let loading = $state(true);
	let enemyX = $state(0);
	let enemyY = $state(0);
	let saving = $state(false);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.settingsPromise.then((row) => {
			if (!active) return;
			settings = row;
			if (row) {
				enemyX = row.enemy_start_x;
				enemyY = row.enemy_start_y;
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
			if (settings) {
				settings.enemy_start_x = updated.enemy_start_x;
				settings.enemy_start_y = updated.enemy_start_y;
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
</script>

<h1>Settings</h1>
<p class="meta">Every entry is benchmarked with these static values so that different entries are tested under identical conditions.</p>

{#if loading}
	<p>Loading settings...</p>
{:else if !settings}
	<p>Could not load settings.</p>
{:else}
	{#if message}<div class="message">{message}</div>{/if}
	<p class="meta">Max parallel jobs is configured per worker on the <a href="/admin/workers">Workers</a> page.</p>

	<h2>Enemy start coordinate</h2>
	<p class="meta">Where the enemy ship starts for all 100 ring sweep runs. The placement runs use their own seeded per-trial enemy spawns.</p>
	<div class="stat-grid">
		<div class="stat"><div class="label">Arena size</div><div>{settings.arena_width} × {settings.arena_height}</div></div>
		<div class="stat"><div class="label">Target (planet) center</div><div>{settings.planet_x}, {settings.planet_y}</div></div>
		<div class="stat"><div class="label">Valid X range</div><div>0 – {settings.arena_width}</div></div>
		<div class="stat"><div class="label">Valid Y range</div><div>0 – {settings.arena_height}</div></div>
	</div>
	<p class="meta">The enemy spawns at this point and flies straight toward the target center. Pick a point near an arena edge for a realistic approach.</p>
	<div class="actions">
		<label for="enemyX">X</label>
		<input id="enemyX" type="number" step="1" min="0" max={settings.arena_width} bind:value={enemyX} style="width:6rem" />
		<label for="enemyY">Y</label>
		<input id="enemyY" type="number" step="1" min="0" max={settings.arena_height} bind:value={enemyY} style="width:6rem" />
		<button onclick={saveEnemyStart} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
	</div>

	<h2>Simulation</h2>
	<div class="admin-table-wrap">
		<table>
			<tbody>
				<tr><th>Base seed</th><td>{settings.seed}</td></tr>
				<tr><th>Placement trials</th><td>{settings.placement_trials}</td></tr>
				<tr><th>Sweep range (n)</th><td>1 to {settings.sweep_max}</td></tr>
				<tr><th>Trials per sweep n</th><td>{settings.sweep_trials}</td></tr>
				<tr><th>Match time cap</th><td>{settings.match_cap_seconds}s</td></tr>
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
