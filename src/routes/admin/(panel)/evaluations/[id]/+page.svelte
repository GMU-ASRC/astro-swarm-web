<script lang="ts">
	import { goto } from '$app/navigation';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import { apiUrl } from '$lib/ts/api';
	import type { PlayerEvaluation, Replay } from '$lib/ts/evaluation';

	let { data } = $props();

	type ChartKind = 'line' | 'bar' | 'sweep' | 'times';

	let ev = $state<PlayerEvaluation | null>(null);
	let loading = $state(true);
	let message = $state('');
	let zoomed: null | ChartKind = $state(null);

	let selectedTrial: number | null = $state(null);
	let selectedReplay: Replay | null = $state(null);
	let loadedReplays = false;

	$effect(() => {
		let active = true;
		loading = true;
		data.entryPromise.then((row) => {
			if (!active) return;
			ev = row;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	let outcomes = $derived(ev?.results?.outcomes ?? []);
	let successRate = $derived(ev?.results?.success_rate ?? 0);
	let dateLabel = $derived(ev ? (ev.completed_at ?? ev.created_at).slice(0, 10) : '');

	let counts = $derived.by(() => {
		const c = { win: 0, lose: 0, timeout: 0 };
		for (const o of outcomes) {
			if (o === 'win') c.win++;
			else if (o === 'lose') c.lose++;
			else c.timeout++;
		}
		return c;
	});

	let chartBust = $derived(ev ? (ev.completed_at ?? ev.status) : '');
	function chartUrl(kind: ChartKind): string {
		if (!ev) return '';
		return apiUrl(`/api/evaluations/${ev.id}/chart/${kind}.png?v=${encodeURIComponent(chartBust)}`);
	}

	function cellClass(o: string): string {
		if (o === 'win') return 'run-win';
		if (o === 'lose') return 'run-lose';
		return 'run-timeout';
	}

	async function loadReplay(trial: number) {
		if (!ev) return;
		selectedTrial = trial;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${ev.id}/replay/${trial}`));
			if (!res.ok) return;
			selectedReplay = await res.json();
		} catch (err) {
			console.error('Error loading replay:', err);
		}
	}

	$effect(() => {
		if (ev && ev.status === 'done' && outcomes.length > 0 && !loadedReplays) {
			loadedReplays = true;
			loadReplay(0);
		}
	});

	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape') zoomed = null;
	}

	let resimulating = $state(false);

	async function resimulate() {
		if (!confirm('Re-run this evaluation with the current simulator build? This overwrites its results and replays.')) return;
		resimulating = true;
		message = '';
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${data.id}/resimulate`), {
				method: 'POST',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (!res.ok && res.status !== 202) {
				message = `Failed to start re-simulation: ${res.status}`;
				resimulating = false;
				return;
			}
			loadedReplays = false;
			selectedReplay = null;
			selectedTrial = null;
			while (true) {
				await new Promise((resolve) => setTimeout(resolve, 2000));
				const poll = await fetch(apiUrl(`/api/evaluations/${data.id}`));
				if (!poll.ok) continue;
				ev = await poll.json();
				if (ev && ev.status !== 'queued' && ev.status !== 'running') break;
			}
		} catch (err) {
			message = `Re-simulate failed: ${err}`;
		}
		resimulating = false;
	}

	async function remove() {
		if (!confirm('Delete this evaluation? This cannot be undone.')) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${data.id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 204) {
				await goto('/admin/evaluations');
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<p><a href="/admin/evaluations">← All evaluations</a></p>

{#if message}<div class="message">{message}</div>{/if}

{#if loading}
	<p>Loading entry...</p>
{:else if !ev}
	<p>Entry not found.</p>
{:else}
	<h1>{ev.username}</h1>
	<p class="meta">
		{(ev.level_id ?? 'farp').toUpperCase()} · {ev.placements?.length ?? 0} defenders · {ev.trials} trials · evaluated {dateLabel}
	</p>
	<p class="meta">{ev.id}</p>

	<div class="actions">
		<a class="admin-btn" href={apiUrl(`/api/evaluations/${ev.id}/export`)}>Download ZIP</a>
		<button onclick={resimulate} disabled={resimulating || ev.status === 'queued' || ev.status === 'running'}>
			{resimulating ? 'Re-simulating...' : 'Re-simulate'}
		</button>
		<button class="admin-btn-danger" onclick={remove}>Delete</button>
	</div>

	<hr />

	{#if ev.status === 'failed'}
		<p>Evaluation failed{ev.error ? `: ${ev.error}` : '.'}</p>
	{:else if ev.status === 'queued' || ev.status === 'running'}
		<p>Benchmark still running ({Math.round((ev.progress ?? 0) * 100)}%). Results appear when the headless run finishes.</p>
	{:else if outcomes.length === 0}
		<p>No benchmark data available.</p>
	{:else}
		<div class="stat-grid">
			<div class="stat">
				<div class="label">Detection rate</div>
				<div>{successRate}%</div>
			</div>
			<div class="stat">
				<div class="label">Intercepts</div>
				<div>{counts.win}</div>
			</div>
			<div class="stat">
				<div class="label">Planet hits</div>
				<div>{counts.lose}</div>
			</div>
			<div class="stat">
				<div class="label">Timeouts</div>
				<div>{counts.timeout}</div>
			</div>
		</div>

		<div class="charts">
			<button type="button" onclick={() => (zoomed = 'line')}><img src={chartUrl('line')} alt="Cumulative detection rate" /></button>
			<button type="button" onclick={() => (zoomed = 'bar')}><img src={chartUrl('bar')} alt="Outcome breakdown" /></button>
			<button type="button" onclick={() => (zoomed = 'sweep')}><img src={chartUrl('sweep')} alt="Detection rate vs number of defenders" /></button>
			<button type="button" onclick={() => (zoomed = 'times')}><img src={chartUrl('times')} alt="Detection and capture times per trial" /></button>
		</div>

		<h2>Runs ({outcomes.length})</h2>
		<p class="meta">Each cell is one trial — green intercepted, red reached the planet, yellow timed out. Click a run to replay it.</p>
		<div class="runs-grid">
			{#each outcomes as o, i}
				<button
					type="button"
					title={`Trial ${i + 1}: ${o}`}
					onclick={() => loadReplay(i)}
					class="{cellClass(o)} {selectedTrial === i ? 'run-selected' : ''}"
					aria-label={`Trial ${i + 1} ${o}`}
				>{i + 1}</button>
			{/each}
		</div>
		{#if selectedReplay}
			<div>
				<p class="meta">Trial {(selectedTrial ?? 0) + 1}</p>
				<FarpReplay replay={selectedReplay} />
			</div>
		{/if}

		<h2>Defender Algorithm</h2>
		<div class="admin-table-wrap">
			<AlgorithmView scripts={ev.algorithm} />
		</div>
	{/if}
{/if}

{#if zoomed && ev}
	<div
		class="zoom-overlay"
		role="button"
		tabindex="-1"
		aria-label="Close zoomed chart"
		onclick={() => (zoomed = null)}
		onkeydown={onKey}
	>
		<img src={chartUrl(zoomed)} alt="Chart" />
	</div>
{/if}

<style>
	hr {
		border: none;
		border-top: 1px solid #d1d5db;
		margin: 1rem 0;
	}

	.zoom-overlay {
		position: fixed;
		inset: 0;
		z-index: 50;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(0, 0, 0, 0.8);
		padding: 1rem;
		cursor: zoom-out;
	}

	.zoom-overlay img {
		max-width: 95vw;
		max-height: 92vh;
		background: #ffffff;
	}
</style>
