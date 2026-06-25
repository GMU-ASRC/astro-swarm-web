<script lang="ts">
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();
	let settings = $state<any>(null);
	let loading = $state(true);
	let maxJobs = $state(1);
	let saving = $state(false);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.settingsPromise.then((row) => {
			if (!active) return;
			settings = row;
			if (row) maxJobs = row.max_jobs;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	async function saveJobs() {
		saving = true;
		message = '';
		try {
			const res = await fetch(apiUrl('/api/evaluations/settings'), {
				method: 'PUT',
				headers: { 'X-API-Key': data.adminKey, 'Content-Type': 'application/json' },
				body: JSON.stringify({ max_jobs: maxJobs })
			});
			if (!res.ok) {
				message = `Failed to save: ${res.status}`;
				return;
			}
			const updated = await res.json();
			maxJobs = updated.max_jobs;
			if (settings) settings.max_jobs = updated.max_jobs;
			message = `Saved. Max parallel jobs set to ${updated.max_jobs}.`;
		} catch (err) {
			message = `Save failed: ${err}`;
		} finally {
			saving = false;
		}
	}
</script>

<h1>Settings</h1>
<p class="meta">Every entry is benchmarked with these static values so that different entries are tested under identical conditions.</p>

{#if loading}
	<p>Loading settings...</p>
{:else if !settings}
	<p>Could not load settings.</p>
{:else}
	<h2>Performance</h2>
	{#if message}<div class="message">{message}</div>{/if}
	<p class="meta">Each entry is split into this many parallel simulator processes to finish benchmarks faster (capped at {settings.max_jobs_cap}). Higher values use more CPU and memory.</p>
	<div class="actions">
		<label for="maxJobs">Max parallel jobs</label>
		<input id="maxJobs" type="number" min="1" max={settings.max_jobs_cap} bind:value={maxJobs} style="width:5rem" />
		<button onclick={saveJobs} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
	</div>

	<h2>Simulation</h2>
	<div class="admin-table-wrap">
		<table>
			<tbody>
				<tr><th>Base seed</th><td>{settings.seed}</td></tr>
				<tr><th>Placement trials</th><td>{settings.placement_trials}</td></tr>
				<tr><th>Sweep range (n)</th><td>1 to {settings.sweep_max}</td></tr>
				<tr><th>Trials per sweep n</th><td>{settings.sweep_trials}</td></tr>
				<tr><th>Static enemy spawn locations</th><td>{settings.spawn_points}</td></tr>
				<tr><th>Match time cap</th><td>{settings.match_cap_seconds}s</td></tr>
				<tr><th>Max parallel jobs</th><td>{settings.max_jobs}</td></tr>
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
