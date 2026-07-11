<script lang="ts">
	import { onMount } from 'svelte';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import ChartCard from '$lib/components/ChartCard.svelte';
	import { apiUrl } from '$lib/ts/api';
	import { barConfig, lineConfig, detectionRateConfig, captureRateConfig, timesConfig } from '$lib/ts/charts';
	import type { PlayerEvaluation, Replay } from '$lib/ts/evaluation';

	interface PageData {
		evaluation: PlayerEvaluation;
	}

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let ev: PlayerEvaluation = $state(data.evaluation);

	let dateLabel = $derived((ev.completed_at ?? ev.created_at).slice(0, 10));
	let pending = $derived(ev.status === 'queued' || ev.status === 'running');
	let percent = $derived(Math.round((ev.progress ?? 0) * 100));

	let outcomes = $derived(ev.results?.outcomes ?? []);
	let successRate = $derived(ev.results?.success_rate ?? 0);
	let isAttack = $derived(ev.is_attack ?? ((ev.level_number ?? 1) >= 3));
	let rateLabel = $derived(isAttack ? 'EVASION RATE' : 'DETECTION RATE');

	let counts = $derived.by(() => {
		const c = { win: 0, lose: 0, timeout: 0 };
		for (const o of outcomes) {
			if (o === 'win') c.win++;
			else if (o === 'lose') c.lose++;
			else c.timeout++;
		}
		return c;
	});

	let detectionTimes = $derived(ev.results?.detection_times ?? []);
	let captureTimes = $derived(ev.results?.capture_times ?? []);

	let selectedTrial: number | null = $state(null);
	let selectedReplay: Replay | null = $state(null);
	let loadedReplays = false;

	function cellColor(o: string): string {
		if (o === 'win') return 'bg-green-500/70 hover:bg-green-400 border-green-300/40';
		if (o === 'lose') return 'bg-red-500/70 hover:bg-red-400 border-red-300/40';
		return 'bg-amber-500/60 hover:bg-amber-400 border-amber-300/40';
	}

	function fmtTime(t: number | undefined): string {
		return t != null && t >= 0 ? `${t}s` : '—';
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

	let sweepRuns = $state<{ n: number; outcome: string; detection_time?: number; capture_time?: number; detection_rate?: number; capture_rate?: number }[]>([]);
	let selectedN: number | null = $state(null);
	let selectedSweepReplay: Replay | null = $state(null);
	let loadedSweep = false;

	async function loadSweepIndex() {
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
		if (ev.status === 'done' && !loadedSweep) {
			loadedSweep = true;
			loadSweepIndex();
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
			{(ev.level_id ?? 'farp').toUpperCase()} · {ev.defender_count ?? ev.placements?.length ?? 0} defenders · {ev.trials} trials · {ev.game_version ?? 'v0.0.4'} · evaluated {dateLabel}
		</p>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-275 mx-auto px-8 max-sm:px-5 py-10">
		{#if ev.status === 'failed'}
			<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 text-center">
				Evaluation failed{ev.error ? `: ${ev.error}` : '.'}
			</div>
		{:else}
			{#if pending}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 mb-6">
				<p class="text-text-muted text-center mb-4">
					Benchmark running — showing partial results ({percent}% complete). Full data appears when the headless run finishes.
				</p>
				<div class="h-3 w-full bg-sky-500/10 border border-sky-500/30 overflow-hidden">
					<div class="h-full bg-sky-400 transition-all duration-500" style={`width: ${percent}%`}></div>
				</div>
				<p class="font-sim text-xs text-sky-300 text-center mt-2 tracking-wider">{percent}%</p>
			</div>
			{/if}
			{#if outcomes.length > 0}
			<div class="mb-8 flex flex-wrap gap-6 items-end">
				<div>
					<div class="font-sim text-[clamp(2.2rem,6vw,3.6rem)] text-green-300 leading-none" style="text-shadow: 0 0 24px rgba(74,222,128,0.35)">
						{successRate}%
					</div>
					<div class="text-xs text-text-muted mt-1 tracking-wider font-sim">{rateLabel}</div>
				</div>
				<div class="flex gap-4 text-sm">
					{#if isAttack}
						<span class="text-green-300">{counts.win} breaches</span>
						<span class="text-red-300">{counts.lose} intercepted</span>
					{:else}
						<span class="text-green-300">{counts.win} intercepts</span>
						<span class="text-red-300">{counts.lose} planet hits</span>
					{/if}
					{#if counts.timeout > 0}<span class="text-amber-300">{counts.timeout} timeouts</span>{/if}
				</div>
			</div>

			<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
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
							TRIAL {(selectedTrial ?? 0) + 1} · {selectedReplay.outcome.toUpperCase()} · DETECTED {fmtTime(selectedReplay.detection_time)} · CAPTURED {fmtTime(selectedReplay.capture_time)}
						</div>
						<FarpReplay replay={selectedReplay} />
					</div>
				{/if}
			</div>

			{#if sweepRuns.length > 0}
				<div class="mt-10">
					<h2 class="font-sim text-xl text-star-white mb-2">Ring Sweep Runs ({sweepRuns.length})</h2>
					<p class="text-xs text-text-muted mb-4">One sim per ring size — n defenders placed in a circle around the target at random orientations, against a fixed enemy spawn. Click an n to replay it.</p>
					<div class="flex flex-wrap gap-2 mb-4" style="max-width: 760px">
						{#each sweepRuns as run}
							<button
								type="button"
								title={`n=${run.n}: ${run.outcome}`}
								onclick={() => loadSweepReplay(run.n)}
								class="w-11 h-11 px-1 border text-[11px] text-white/90 flex items-center justify-center transition-colors {cellColor(run.outcome)} {selectedN === run.n ? 'ring-2 ring-sky-300' : ''}"
								aria-label={`n ${run.n} ${run.outcome}`}
							>{run.n}</button>
						{/each}
					</div>
					{#if selectedSweepReplay}
						<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 max-w-[860px]">
							<div class="text-xs text-text-muted mb-2 font-sim tracking-wider">
								N = {selectedN} DEFENDERS · {selectedSweepReplay.outcome.toUpperCase()} · DETECTED {fmtTime(selectedSweepReplay.detection_time)} · CAPTURED {fmtTime(selectedSweepReplay.capture_time)}
							</div>
							<FarpReplay replay={selectedSweepReplay} />
						</div>
					{/if}
				</div>
			{/if}

			<div class="mt-10">
				<h2 class="font-sim text-xl text-star-white mb-4">{isAttack ? 'Evader Algorithm' : 'Defender Algorithm'}</h2>
				<div class="p-5 border-2 border-sky-500/20 bg-page-bg/60 overflow-x-auto">
					<AlgorithmView scripts={ev.algorithm} />
				</div>
			</div>
			{:else if !pending}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No benchmark data available.
			</div>
			{/if}
		{/if}
	</div>
</div>
