<script lang="ts">
	import LineChart from '$lib/components/LineChart.svelte';
	import BarChart from '$lib/components/BarChart.svelte';
	import type { PlayerEvaluation, BaselineResult } from '$lib/ts/evaluation';

	interface PageData {
		evaluation: PlayerEvaluation;
		baseline: BaselineResult;
	}

	let { data }: { data: PageData } = $props();

	let ev = $derived(data.evaluation);
	let dateLabel = $derived(
		(ev.completed_at ?? ev.created_at).slice(0, 10)
	);

	let lineSeries = $derived(
		[
			{ label: ev.username, color: '#1f77b4', points: ev.results },
			...(data.baseline.results.length > 0
				? [{ label: 'Community avg', color: '#ff7f0e', points: data.baseline.results }]
				: [])
		]
	);

	let pending = $derived(ev.status === 'queued' || ev.status === 'running');
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
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				Benchmark {ev.status}. Check back shortly — results appear when the headless run finishes.
			</div>
		{:else if ev.results.length === 0}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No benchmark data available.
			</div>
		{:else}
			<div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<LineChart
						series={lineSeries}
						title="Capture Rate"
						subtitle={dateLabel}
					/>
				</div>
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<BarChart
						points={ev.results}
						title={`(${ev.username}) Success Rate vs. N`}
					/>
				</div>
			</div>
		{/if}
	</div>
</div>
