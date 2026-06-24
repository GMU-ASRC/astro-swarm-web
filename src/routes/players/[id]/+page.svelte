<script lang="ts">
	import { onMount } from 'svelte';
	import LineChart from '$lib/components/LineChart.svelte';
	import BarChart from '$lib/components/BarChart.svelte';
	import { apiUrl } from '$lib/ts/api';
	import type { PlayerEvaluation, BaselineResult } from '$lib/ts/evaluation';

	interface PageData {
		evaluation: PlayerEvaluation;
		baseline: BaselineResult;
	}

	let { data }: { data: PageData } = $props();

	let ev: PlayerEvaluation = $state(data.evaluation);
	let zoomed: null | 'line' | 'bar' = $state(null);

	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape') zoomed = null;
	}

	let dateLabel = $derived((ev.completed_at ?? ev.created_at).slice(0, 10));
	let pending = $derived(ev.status === 'queued' || ev.status === 'running');
	let percent = $derived(Math.round((ev.progress ?? 0) * 100));

	let lineSeries = $derived([
		{ label: ev.username, color: '#1f77b4', points: ev.results },
		...(data.baseline.results.length > 0
			? [{ label: 'Community avg', color: '#ff7f0e', points: data.baseline.results }]
			: [])
	]);

	onMount(() => {
		if (!pending) return;
		console.log('Benchmark running. Check back shortly — results appear when the headless run finishes.');
		const interval = setInterval(async () => {
			try {
				const res = await fetch(apiUrl(`/api/evaluations/${ev.player_id}`));
				if (!res.ok) return;
				const next: PlayerEvaluation = await res.json();
				ev = next;
				console.log(
					`Benchmark ${next.status} — ${Math.round((next.progress ?? 0) * 100)}% (N ${Math.round((next.progress ?? 0) * next.n_max)}/${next.n_max})`
				);
				if (next.status === 'done' || next.status === 'failed') {
					clearInterval(interval);
				}
			} catch (err) {
				console.error('Error polling evaluation:', err);
			}
		}, 2000);
		return () => clearInterval(interval);
	});
</script>

<svelte:head>
	<title>{ev.username} — AstroSwarm</title>
	<meta name="description" content={`FARP defender benchmark for ${ev.username}.`} />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-6">
		<a href="/players" class="text-sm text-sky-300 hover:text-sky-200">← All players</a>
		<h1 class="font-game text-[clamp(1.6rem,4vw,2.6rem)] font-bold text-star-white leading-tight mt-3 mb-2" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			{ev.username}
		</h1>
		<p class="font-sim text-sm text-text-muted">
			FARP defender algorithm · {ev.trials} trials per swarm size · evaluated {dateLabel}
		</p>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 py-10">
		{#if ev.status === 'failed'}
			<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 text-center">
				Evaluation failed{ev.error ? `: ${ev.error}` : '.'}
			</div>
		{:else if pending}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5">
				<p class="text-text-muted text-center mb-4">
					Benchmark running. Check back shortly — results appear when the headless run finishes.
				</p>
				<div class="h-3 w-full bg-sky-500/10 border border-sky-500/30 overflow-hidden">
					<div class="h-full bg-sky-400 transition-all duration-500" style={`width: ${percent}%`}></div>
				</div>
				<p class="font-game text-xs text-sky-300 text-center mt-2 tracking-wider">
					{percent}% · N {Math.round((ev.progress ?? 0) * ev.n_max)} / {ev.n_max}
				</p>
			</div>
		{:else if ev.results.length === 0}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No benchmark data available.
			</div>
		{:else}
			<div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
				<button
					type="button"
					onclick={() => (zoomed = 'line')}
					class="p-4 border-2 border-sky-500/20 bg-sky-500/5 cursor-zoom-in hover:border-sky-400/50 transition-colors"
					aria-label="Zoom Capture Rate chart"
				>
					<LineChart series={lineSeries} title="Capture Rate" subtitle={dateLabel} />
				</button>
				<button
					type="button"
					onclick={() => (zoomed = 'bar')}
					class="p-4 border-2 border-sky-500/20 bg-sky-500/5 cursor-zoom-in hover:border-sky-400/50 transition-colors"
					aria-label="Zoom Success Rate chart"
				>
					<BarChart points={ev.results} title={`(${ev.username}) Success Rate vs. N`} />
				</button>
			</div>
		{/if}
	</div>
</div>

{#if zoomed}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4 cursor-zoom-out"
		role="button"
		tabindex="-1"
		aria-label="Close zoomed chart"
		onclick={() => (zoomed = null)}
		onkeydown={onKey}
	>
		<div
			class="bg-white rounded-md p-2 max-w-[95vw] max-h-[92vh] overflow-auto"
			role="presentation"
			onclick={(e) => e.stopPropagation()}
		>
			{#if zoomed === 'line'}
				<LineChart series={lineSeries} title="Capture Rate" subtitle={dateLabel} height={560} maxWidth="1100px" />
			{:else}
				<BarChart points={ev.results} title={`(${ev.username}) Success Rate vs. N`} height={560} maxWidth="1100px" />
			{/if}
		</div>
	</div>
{/if}
