<script lang="ts">
	import { onMount } from 'svelte';
	import LineChart from '$lib/components/LineChart.svelte';
	import BarChart from '$lib/components/BarChart.svelte';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import { apiUrl } from '$lib/ts/api';
	import type { PlayerEvaluation, BaselineResult, Replay } from '$lib/ts/evaluation';

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

	let outcomes = $derived(ev.results?.outcomes ?? []);
	let successRate = $derived(ev.results?.success_rate ?? 0);

	let counts = $derived.by(() => {
		const c = { win: 0, lose: 0, timeout: 0 };
		for (const o of outcomes) {
			if (o === 'win') c.win++;
			else if (o === 'lose') c.lose++;
			else c.timeout++;
		}
		return c;
	});

	let total = $derived(Math.max(1, outcomes.length));

	let barPoints = $derived([
		{ n: 'Intercept', success_rate: (counts.win / total) * 100 },
		{ n: 'Planet hit', success_rate: (counts.lose / total) * 100 },
		{ n: 'Timeout', success_rate: (counts.timeout / total) * 100 }
	] as any);

	let cumulativeSeries = $derived.by(() => {
		let wins = 0;
		const points = outcomes.map((o, i) => {
			if (o === 'win') wins++;
			return { n: i + 1, success_rate: (wins / (i + 1)) * 100 };
		});
		return [{ label: ev.username, color: '#1f77b4', points }];
	});

	let selectedTrial: number | null = $state(null);
	let selectedReplay: Replay | null = $state(null);
	let loadedReplays = false;

	function cellColor(o: string): string {
		if (o === 'win') return 'bg-green-500/70 hover:bg-green-400 border-green-300/40';
		if (o === 'lose') return 'bg-red-500/70 hover:bg-red-400 border-red-300/40';
		return 'bg-amber-500/60 hover:bg-amber-400 border-amber-300/40';
	}

	async function loadReplay(trial: number) {
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
		if (ev.status === 'done' && outcomes.length > 0 && !loadedReplays) {
			loadedReplays = true;
			loadReplay(0);
		}
	});

	onMount(() => {
		if (!pending) return;
		console.log('Benchmark running. Check back shortly — results appear when the headless run finishes.');
		const interval = setInterval(async () => {
			try {
				const res = await fetch(apiUrl(`/api/evaluations/${ev.id}`));
				if (!res.ok) return;
				const next: PlayerEvaluation = await res.json();
				ev = next;
				console.log(`Benchmark ${next.status} — ${Math.round((next.progress ?? 0) * 100)}%`);
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
	<div class="max-w-275 mx-auto px-8 max-sm:px-5 pt-12 pb-6">
		<a href="/levels" class="text-sm text-sky-300 hover:text-sky-200">← All levels</a>
		<h1 class="font-sim text-[clamp(1.6rem,4vw,2.6rem)] font-bold text-star-white leading-tight mt-3 mb-2" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			{ev.username}
		</h1>
		<p class="font-sim text-sm text-text-muted">
			FARP · {ev.placements?.length ?? 0} defenders · {ev.trials} trials · evaluated {dateLabel}
		</p>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-275 mx-auto px-8 max-sm:px-5 py-10">
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
				<p class="font-sim text-xs text-sky-300 text-center mt-2 tracking-wider">{percent}%</p>
			</div>
		{:else if outcomes.length === 0}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No benchmark data available.
			</div>
		{:else}
			<div class="mb-8 flex flex-wrap gap-6 items-end">
				<div>
					<div class="font-sim text-[clamp(2.2rem,6vw,3.6rem)] text-green-300 leading-none" style="text-shadow: 0 0 24px rgba(74,222,128,0.35)">
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
				<div
					role="button"
					tabindex="0"
					onclick={() => (zoomed = 'line')}
					onkeydown={(e) => e.key === 'Enter' && (zoomed = 'line')}
					class="p-4 border-2 border-sky-500/20 bg-sky-500/5 cursor-zoom-in hover:border-sky-400/50 transition-colors"
				>
					<LineChart series={cumulativeSeries} title="Cumulative Detection Rate" subtitle={dateLabel} xLabel="Trial" />
				</div>
				<div
					role="button"
					tabindex="0"
					onclick={() => (zoomed = 'bar')}
					onkeydown={(e) => e.key === 'Enter' && (zoomed = 'bar')}
					class="p-4 border-2 border-sky-500/20 bg-sky-500/5 cursor-zoom-in hover:border-sky-400/50 transition-colors"
				>
					<BarChart points={barPoints} title="Outcome Breakdown" xLabel="Outcome" yLabel="% of trials" />
				</div>
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
					<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 max-w-[680px]">
						<div class="text-xs text-text-muted mb-2 font-sim tracking-wider">
							TRIAL {(selectedTrial ?? 0) + 1} · {selectedReplay.outcome.toUpperCase()}
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
				<LineChart series={cumulativeSeries} title="Cumulative Detection Rate" subtitle={dateLabel} xLabel="Trial" height={560} maxWidth="1100px" />
			{:else}
				<BarChart points={barPoints} title="Outcome Breakdown" xLabel="Outcome" yLabel="% of trials" height={560} maxWidth="1100px" />
			{/if}
		</div>
	</div>
{/if}
