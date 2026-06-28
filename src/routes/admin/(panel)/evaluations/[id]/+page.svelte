<script lang="ts">
	import { goto } from '$app/navigation';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import ChartCard from '$lib/components/ChartCard.svelte';
	import { apiUrl } from '$lib/ts/api';
	import { barConfig, lineConfig, detectionRateConfig, captureRateConfig, timesConfig } from '$lib/ts/charts';
	import type { PlayerEvaluation, Replay } from '$lib/ts/evaluation';

	let { data } = $props();

	let ev = $state<PlayerEvaluation | null>(null);
	let loading = $state(true);
	let message = $state('');

	let selectedTrial: number | null = $state(null);
	let selectedReplay: Replay | null = $state(null);
	let loadedReplays = false;

	let sweepRuns = $state<{ n: number; outcome: string; detection_time?: number; capture_time?: number; detection_rate?: number; capture_rate?: number }[]>([]);
	let selectedN: number | null = $state(null);
	let selectedSweepReplay: Replay | null = $state(null);
	let loadedSweep = false;

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

	let detectionTimes = $derived(ev?.results?.detection_times ?? []);
	let captureTimes = $derived(ev?.results?.capture_times ?? []);

	function cellClass(o: string): string {
		if (o === 'win') return 'run-win';
		if (o === 'lose') return 'run-lose';
		return 'run-timeout';
	}

	function fmtTime(t: number | undefined): string {
		return t != null && t >= 0 ? `${t}s` : '—';
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

	async function loadSweepIndex() {
		if (!ev) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${ev.id}/sweep-replays`));
			if (!res.ok) return;
			sweepRuns = await res.json();
			if (sweepRuns.length > 0) loadSweepReplay(sweepRuns[0].n);
		} catch (err) {
			console.error('Error loading sweep index:', err);
		}
	}

	async function loadSweepReplay(n: number) {
		if (!ev) return;
		selectedN = n;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${ev.id}/sweep-replay/${n}`));
			if (!res.ok) return;
			selectedSweepReplay = await res.json();
		} catch (err) {
			console.error('Error loading sweep replay:', err);
		}
	}

	$effect(() => {
		if (ev && ev.status === 'done' && !loadedSweep) {
			loadedSweep = true;
			loadSweepIndex();
		}
	});

	let resimulating = $state(false);
	let cancelling = $state(false);

	async function cancelRun() {
		if (!confirm('Cancel this running evaluation?')) return;
		cancelling = true;
		message = '';
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${data.id}/cancel`), {
				method: 'POST',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (!res.ok && res.status !== 202) {
				message = `Failed to cancel: ${res.status}`;
				return;
			}
			const poll = await fetch(apiUrl(`/api/evaluations/${data.id}`));
			if (poll.ok) ev = await poll.json();
		} catch (err) {
			message = `Cancel failed: ${err}`;
		} finally {
			cancelling = false;
		}
	}

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
			loadedSweep = false;
			sweepRuns = [];
			selectedSweepReplay = null;
			selectedN = null;
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
		{#if ev.status === 'queued' || ev.status === 'running'}
			<button class="admin-btn-danger" onclick={cancelRun} disabled={cancelling}>
				{cancelling ? 'Cancelling...' : 'Cancel run'}
			</button>
		{/if}
		<button class="admin-btn-danger" onclick={remove}>Delete</button>
	</div>

	<hr />

	{#if ev.status === 'failed'}
		<p>Evaluation failed{ev.error ? `: ${ev.error}` : '.'}</p>
	{:else if ev.status === 'cancelled'}
		<p>This evaluation was cancelled. Re-simulate it to run the benchmark again.</p>
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
			<ChartCard config={lineConfig(outcomes)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/line.png`)} />
			<ChartCard config={barConfig(outcomes)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/bar.png`)} />
			{#if sweepRuns.length > 0}
				<ChartCard config={detectionRateConfig(sweepRuns)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/sweep.png`)} />
				<ChartCard config={captureRateConfig(sweepRuns)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/capture.png`)} />
			{/if}
			{#if detectionTimes.length > 0}
				<ChartCard config={timesConfig(detectionTimes, captureTimes)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/times.png`)} />
			{/if}
		</div>

		<h2>Placement Runs ({outcomes.length})</h2>
		<p class="meta">The player's own defender placements against {outcomes.length} random enemy spawns — green intercepted, red reached the planet, yellow timed out. Click a run to replay it.</p>
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
				<p class="meta">
					Trial {(selectedTrial ?? 0) + 1} · outcome: {selectedReplay.outcome} · detected:
					{fmtTime(selectedReplay.detection_time)} · captured: {fmtTime(selectedReplay.capture_time)}
				</p>
				<FarpReplay replay={selectedReplay} />
			</div>
		{/if}

		<h2>Ring Sweep Runs ({sweepRuns.length})</h2>
		<p class="meta">One sim per ring size — n defenders placed in a circle around the target at random orientations, against a fixed enemy spawn. Click an n to replay it.</p>
		{#if sweepRuns.length > 0}
			<div class="runs-grid">
				{#each sweepRuns as run}
					<button
						type="button"
						title={`n=${run.n}: ${run.outcome}`}
						onclick={() => loadSweepReplay(run.n)}
						class="{cellClass(run.outcome)} {selectedN === run.n ? 'run-selected' : ''}"
						aria-label={`n ${run.n} ${run.outcome}`}
					>{run.n}</button>
				{/each}
			</div>
			{#if selectedSweepReplay}
				<div>
					<p class="meta">
						N = {selectedN} defenders · outcome: {selectedSweepReplay.outcome} · detected:
						{fmtTime(selectedSweepReplay.detection_time)} · captured: {fmtTime(selectedSweepReplay.capture_time)}
					</p>
					<FarpReplay replay={selectedSweepReplay} />
				</div>
			{/if}
		{:else}
			<div class="message">No ring sweep data for this entry. Re-simulate it with the latest simulator build to generate the sweep replays.</div>
		{/if}

		<h2>Defender Algorithm</h2>
		<div class="admin-table-wrap">
			<AlgorithmView scripts={ev.algorithm} />
		</div>
	{/if}
{/if}

<style>
	hr {
		border: none;
		border-top: 1px solid #d1d5db;
		margin: 1rem 0;
	}
</style>
