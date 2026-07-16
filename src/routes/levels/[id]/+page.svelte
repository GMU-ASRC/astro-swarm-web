<script lang="ts">
	import { onMount } from 'svelte';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import ChartCard from '$lib/components/ChartCard.svelte';
	import { apiUrl } from '$lib/ts/api';
	import { barConfig, lineConfig, detectionRateConfig, captureRateConfig, timesConfig } from '$lib/ts/charts';
	import type { PlayerEvaluation, Replay } from '$lib/ts/evaluation';
	import { EVADER_CONFIG, PILOT_EVADER_CONFIG, configRows, defenderConfig } from '$lib/ts/shipConfig';

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
	let isPilot = $derived((ev.level_number ?? 1) === 3);

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
	let goalTimes = $derived(ev.results?.goal_times ?? []);
	let detectionRate = $derived(ev.results?.detection_rate ?? rateOf(detectionTimes));
	let captureRate = $derived(ev.results?.capture_rate ?? rateOf(captureTimes));
	let meanGoalTime = $derived(mean(goalTimes));

	let defenderRows = $derived(configRows(defenderConfig(ev.algorithm)));
	let evaderRows = $derived(configRows(isPilot ? PILOT_EVADER_CONFIG : EVADER_CONFIG));
	let pilotOutcome = $derived(outcomes[0] ?? 'timeout');
	let pilotDetect = $derived(detectionTimes[0] ?? -1);
	let pilotCapture = $derived(captureTimes[0] ?? -1);
	let pilotGoal = $derived(goalTimes[0] ?? -1);

	function rateOf(times: number[]): number {
		if (times.length === 0) return 0;
		const hits = times.filter((t) => t >= 0).length;
		return Math.round((1000 * hits) / times.length) / 10;
	}

	function mean(times: number[]): number | null {
		const hits = times.filter((t) => t >= 0);
		if (hits.length === 0) return null;
		return Math.round((10 * hits.reduce((a, b) => a + b, 0)) / hits.length) / 10;
	}

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

	let sweepRuns = $state<{ n: number; outcome: string; detection_time?: number; capture_time?: number; detection_rate?: number; capture_rate?: number; trial_count?: number }[]>([]);
	let sweepTrials = $state<{ trial: number; outcome: string }[]>([]);
	let selectedN: number | null = $state(null);
	let selectedTrial: number | null = $state(null);
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
		selectedTrial = null;
		sweepTrials = [];
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${ev.id}/sweep-replay/${n}`));
			if (!res.ok) return;
			selectedSweepReplay = await res.json();
		} catch (err) {
			console.error('Error loading sweep replay:', err);
		}
		loadSweepTrialIndex(n);
	}

	async function loadSweepTrialIndex(n: number) {
		if (!sweepRuns.find((run) => run.n === n)?.trial_count) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${ev.id}/sweep-replay/${n}/trials`));
			if (!res.ok) return;
			sweepTrials = await res.json();
		} catch (err) {
			console.error('Error loading sweep trial index:', err);
		}
	}

	async function loadSweepTrialReplay(n: number, trial: number) {
		selectedTrial = trial;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${ev.id}/sweep-replay/${n}/trial/${trial}`));
			if (!res.ok) return;
			selectedSweepReplay = await res.json();
		} catch (err) {
			console.error('Error loading sweep trial replay:', err);
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
			{#if isPilot}
			<div class="mb-8 flex flex-wrap gap-6 items-end">
				<div>
					<div class="font-sim text-[clamp(1.8rem,5vw,3rem)] leading-none {pilotOutcome === 'win' ? 'text-green-300' : 'text-red-300'}" style="text-shadow: 0 0 24px rgba(74,222,128,0.25)">
						{pilotOutcome === 'win' ? 'PLANET REACHED' : pilotOutcome === 'lose' ? 'CAUGHT' : 'OUT OF TIME'}
					</div>
					<div class="text-xs text-text-muted mt-1 tracking-wider font-sim">PILOTED RUN</div>
				</div>
			</div>

			<div class="mb-8 grid grid-cols-2 md:grid-cols-4 gap-4">
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<div class="font-sim text-2xl text-sky-300">{fmtTime(pilotDetect)}</div>
					<div class="text-[11px] text-text-muted mt-1">Detected — a defender first saw the pilot</div>
				</div>
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<div class="font-sim text-2xl text-green-300">{fmtTime(pilotCapture)}</div>
					<div class="text-[11px] text-text-muted mt-1">Captured — a defender touched the pilot</div>
				</div>
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<div class="font-sim text-2xl text-red-300">{fmtTime(pilotGoal)}</div>
					<div class="text-[11px] text-text-muted mt-1">Reached planet — the pilot got to the goal</div>
				</div>
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<div class="font-sim text-2xl text-star-white">{ev.defender_count ?? ev.placements?.length ?? 0}</div>
					<div class="text-[11px] text-text-muted mt-1">Defenders faced</div>
				</div>
			</div>
			{:else}
			<div class="mb-8 flex flex-wrap gap-6 items-end">
				<div>
					<div class="font-sim text-[clamp(2.2rem,6vw,3.6rem)] text-green-300 leading-none" style="text-shadow: 0 0 24px rgba(74,222,128,0.35)">
						{successRate}%
					</div>
					<div class="text-xs text-text-muted mt-1 tracking-wider font-sim">SUCCESS RATE</div>
				</div>
				<div class="flex gap-4 text-sm">
					<span class="text-green-300">{counts.win} captured</span>
					<span class="text-red-300">{counts.lose} reached the planet</span>
					{#if counts.timeout > 0}<span class="text-amber-300">{counts.timeout} timeouts</span>{/if}
				</div>
			</div>

			<div class="mb-8 grid grid-cols-2 md:grid-cols-4 gap-4">
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<div class="font-sim text-2xl text-sky-300">{detectionRate}%</div>
					<div class="text-[11px] text-text-muted mt-1">Detection rate — a defender saw the evader</div>
				</div>
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<div class="font-sim text-2xl text-green-300">{captureRate}%</div>
					<div class="text-[11px] text-text-muted mt-1">Capture rate — a defender touched the evader</div>
				</div>
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<div class="font-sim text-2xl text-red-300">{meanGoalTime != null ? `${meanGoalTime}s` : '—'}</div>
					<div class="text-[11px] text-text-muted mt-1">Mean time for the evader to reach the planet</div>
				</div>
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
					<div class="font-sim text-2xl text-star-white">{ev.defender_count ?? ev.placements?.length ?? 0}</div>
					<div class="text-[11px] text-text-muted mt-1">Defenders placed</div>
				</div>
			</div>
			{/if}

			<div class="mb-10 grid grid-cols-1 md:grid-cols-2 gap-6">
				<div class="p-5 border-2 border-sky-500/20 bg-page-bg/60">
					<h2 class="font-sim text-sm text-sky-300 mb-3 tracking-wider">DEFENDER CONFIG</h2>
					<dl class="grid grid-cols-2 gap-y-2 text-sm">
						{#each defenderRows as row}
							<dt class="text-text-muted">{row.label}</dt>
							<dd class="text-star-white text-right font-sim">{row.value}</dd>
						{/each}
					</dl>
				</div>
				<div class="p-5 border-2 border-red-500/20 bg-page-bg/60">
					<h2 class="font-sim text-sm text-red-300 mb-3 tracking-wider">EVADER CONFIG</h2>
					<dl class="grid grid-cols-2 gap-y-2 text-sm">
						{#each evaderRows as row}
							<dt class="text-text-muted">{row.label}</dt>
							<dd class="text-star-white text-right font-sim">{row.value}</dd>
						{/each}
					</dl>
					<p class="text-[11px] text-text-muted mt-3">
						{isPilot
							? 'The evader is flown by the player from a chosen point on the outer ring.'
							: 'The evader drives straight at the planet from a random point on the outer ring.'}
					</p>
				</div>
			</div>

			{#if !isPilot}
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
							TRIAL {(selectedTrial ?? 0) + 1} · {selectedReplay.outcome.toUpperCase()} · DETECTED {fmtTime(selectedReplay.detection_time)} · CAPTURED {fmtTime(selectedReplay.capture_time)} · REACHED PLANET {fmtTime(selectedReplay.goal_time)}
						</div>
						<FarpReplay replay={selectedReplay} />
					</div>
				{/if}
			</div>
			{:else if selectedReplay}
			<div class="mt-10">
				<h2 class="font-sim text-xl text-star-white mb-2">Recorded Flight</h2>
				<p class="text-xs text-text-muted mb-4">The player's own flight, rendered from the movement recorded in game.</p>
				<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 max-w-[860px]">
					<div class="text-xs text-text-muted mb-2 font-sim tracking-wider">
						{selectedReplay.outcome.toUpperCase()} · DETECTED {fmtTime(selectedReplay.detection_time)} · CAPTURED {fmtTime(selectedReplay.capture_time)} · REACHED PLANET {fmtTime(selectedReplay.goal_time)}
					</div>
					<FarpReplay replay={selectedReplay} />
				</div>
			</div>
			{/if}

			{#if sweepRuns.length > 0}
				<div class="mt-10">
					<h2 class="font-sim text-xl text-star-white mb-2">Ring Sweep Runs ({sweepRuns.length})</h2>
					<p class="text-xs text-text-muted mb-4">Each ring size is simulated repeatedly — n defenders placed in a circle around the target, the ring rotated to a seeded random angle each trial, against a fixed enemy spawn. Click an n to replay it.</p>
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
					{#if sweepTrials.length > 0}
						<p class="text-xs text-text-muted mb-2">Each n is simulated multiple times with the ring rotated to a different angle. Pick a trial to replay it.</p>
						<div class="flex flex-wrap gap-2 mb-4" style="max-width: 760px">
							{#each sweepTrials as trial}
								<button
									type="button"
									title={`n=${selectedN} trial ${trial.trial + 1}: ${trial.outcome}`}
									onclick={() => loadSweepTrialReplay(selectedN as number, trial.trial)}
									class="w-11 h-11 px-1 border text-[11px] text-white/90 flex items-center justify-center transition-colors {cellColor(trial.outcome)} {selectedTrial === trial.trial ? 'ring-2 ring-sky-300' : ''}"
									aria-label={`trial ${trial.trial + 1} ${trial.outcome}`}
								>{trial.trial + 1}</button>
							{/each}
						</div>
					{/if}
					{#if selectedSweepReplay}
						<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 max-w-[860px]">
							<div class="text-xs text-text-muted mb-2 font-sim tracking-wider">
								N = {selectedN} DEFENDERS{selectedTrial !== null ? ` · TRIAL ${selectedTrial + 1}` : ''} · {selectedSweepReplay.outcome.toUpperCase()} · DETECTED {fmtTime(selectedSweepReplay.detection_time)} · CAPTURED {fmtTime(selectedSweepReplay.capture_time)} · REACHED PLANET {fmtTime(selectedSweepReplay.goal_time)}
							</div>
							<FarpReplay replay={selectedSweepReplay} />
						</div>
					{/if}
				</div>
			{/if}

			<div class="mt-10">
				<h2 class="font-sim text-xl text-star-white mb-4">{isPilot ? 'Opponent Algorithm' : 'Defender Algorithm'}</h2>
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
