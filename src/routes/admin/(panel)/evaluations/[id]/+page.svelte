<script lang="ts">
	import { goto } from '$app/navigation';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import { apiUrl } from '$lib/ts/api';
	import type { PlayerEvaluation, Replay } from '$lib/ts/evaluation';

	let { data } = $props();

	let ev = $state<PlayerEvaluation | null>(null);
	let loading = $state(true);
	let message = $state('');
	let zoomed: null | 'line' | 'bar' = $state(null);

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
	function chartUrl(kind: 'line' | 'bar'): string {
		if (!ev) return '';
		return apiUrl(`/api/evaluations/${ev.id}/chart/${kind}.png?v=${encodeURIComponent(chartBust)}`);
	}

	function cellColor(o: string): string {
		if (o === 'win') return 'bg-green-500/70 hover:bg-green-400 border-green-300/40';
		if (o === 'lose') return 'bg-red-500/70 hover:bg-red-400 border-red-300/40';
		return 'bg-amber-500/60 hover:bg-amber-400 border-amber-300/40';
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

<div class="max-w-5xl">
	<a href="/admin/evaluations" class="text-sm text-sky-300 hover:text-sky-200">← All evaluations</a>

	{#if message}<div class="mt-4 text-red-300 text-sm">{message}</div>{/if}

	{#if loading}
		<div class="mt-6 p-6 border-2 border-sky-500/20 bg-sky-500/5 text-sky-200 font-game tracking-wider text-center animate-pulse">
			Loading entry...
		</div>
	{:else if !ev}
		<div class="mt-6 p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center">
			Entry not found.
		</div>
	{:else}
		<div class="mt-3 flex flex-wrap items-start justify-between gap-4">
			<div>
				<h1 class="font-game text-3xl text-star-white" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
					{ev.username}
				</h1>
				<p class="text-sm text-text-muted mt-2">
					{(ev.level_id ?? 'farp').toUpperCase()} · {ev.placements?.length ?? 0} defenders · {ev.trials} trials · evaluated {dateLabel}
				</p>
				<p class="text-[11px] text-text-muted/60 font-mono mt-1 break-all">{ev.id}</p>
			</div>
			<div class="flex gap-2">
				<a
					href={apiUrl(`/api/evaluations/${ev.id}/export`)}
					class="px-4 py-2 border-2 border-sky-400/40 text-sky-200 font-game text-sm tracking-wider hover:bg-sky-500/15 transition-colors"
				>
					DOWNLOAD ZIP
				</a>
				<button
					onclick={remove}
					class="px-4 py-2 border-2 border-red-500/30 text-red-300 font-game text-sm tracking-wider hover:bg-red-500/15 transition-colors"
				>
					DELETE
				</button>
			</div>
		</div>

		<div class="h-px my-6" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

		{#if ev.status === 'failed'}
			<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 text-center">
				Evaluation failed{ev.error ? `: ${ev.error}` : '.'}
			</div>
		{:else if ev.status === 'queued' || ev.status === 'running'}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				Benchmark still running ({Math.round((ev.progress ?? 0) * 100)}%). Results appear when the headless run finishes.
			</div>
		{:else if outcomes.length === 0}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No benchmark data available.
			</div>
		{:else}
			<div class="mb-8 flex flex-wrap gap-6 items-end">
				<div>
					<div class="font-sim text-[clamp(2rem,6vw,3.2rem)] text-green-300 leading-none" style="text-shadow: 0 0 24px rgba(74,222,128,0.35)">
						{successRate}%
					</div>
					<div class="text-xs text-text-muted mt-1 tracking-wider font-sim">DETECTION RATE</div>
				</div>
				<div class="flex gap-4 text-sm">
					<span class="text-green-300">{counts.win} intercepts</span>
					<span class="text-red-300">{counts.lose} planet hits</span>
					{#if counts.timeout > 0}<span class="text-amber-300">{counts.timeout} timeouts</span>{/if}
				</div>
			</div>

			<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
				<button type="button" onclick={() => (zoomed = 'line')} class="p-2 border-2 border-sky-500/20 bg-white cursor-zoom-in hover:border-sky-400/50 transition-colors">
					<img src={chartUrl('line')} alt="Cumulative detection rate" class="w-full" />
				</button>
				<button type="button" onclick={() => (zoomed = 'bar')} class="p-2 border-2 border-sky-500/20 bg-white cursor-zoom-in hover:border-sky-400/50 transition-colors">
					<img src={chartUrl('bar')} alt="Outcome breakdown" class="w-full" />
				</button>
			</div>

			<div class="mt-10">
				<h2 class="font-sim text-xl text-star-white mb-2">Runs ({outcomes.length})</h2>
				<p class="text-xs text-text-muted mb-4">Each cell is one trial — green intercepted, red reached the planet. Click a run to replay it.</p>
				<div class="flex flex-wrap gap-2 mb-4" style="max-width: 760px">
					{#each outcomes as o, i}
						<button
							type="button"
							title={`Trial ${i + 1}: ${o}`}
							onclick={() => loadReplay(i)}
							class="w-11 h-11 px-1 border text-[11px] text-white/90 flex items-center justify-center transition-colors {cellColor(o)} {selectedTrial === i ? 'ring-2 ring-sky-300' : ''}"
							aria-label={`Trial ${i + 1} ${o}`}
						>{i + 1}</button>
					{/each}
				</div>
				{#if selectedReplay}
					<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 max-w-[860px]">
						<div class="text-xs text-text-muted mb-2 font-sim tracking-wider">
							TRIAL {(selectedTrial ?? 0) + 1}
						</div>
						<FarpReplay replay={selectedReplay} />
					</div>
				{/if}
			</div>

			<div class="mt-10">
				<h2 class="font-sim text-xl text-star-white mb-4">Defender Algorithm</h2>
				<div class="p-5 border-2 border-sky-500/20 bg-page-bg/60 overflow-x-auto">
					<AlgorithmView scripts={ev.algorithm} />
				</div>
			</div>
		{/if}
	{/if}
</div>

{#if zoomed && ev}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4 cursor-zoom-out"
		role="button"
		tabindex="-1"
		aria-label="Close zoomed chart"
		onclick={() => (zoomed = null)}
		onkeydown={onKey}
	>
		<div class="bg-white rounded-md p-2 max-w-[95vw] max-h-[92vh] overflow-auto" role="presentation" onclick={(e) => e.stopPropagation()}>
			<img src={chartUrl(zoomed)} alt="Chart" class="max-w-full" style="width: 1000px" />
		</div>
	</div>
{/if}
