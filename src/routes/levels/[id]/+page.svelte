<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '@iconify/svelte';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import ChartCard from '$lib/components/ChartCard.svelte';
	import { apiUrl } from '$lib/ts/api';
	import { barConfig, lineConfig, detectionRateConfig, captureRateConfig, timesConfig } from '$lib/ts/charts';
	import type { PlayerEvaluation, Replay } from '$lib/ts/evaluation';
	import { EVADER_CONFIG, PILOT_EVADER_CONFIG, configRows, defenderConfig } from '$lib/ts/shipConfig';
	import { FARP_LEVELS, levelById } from '$lib/ts/levels';

	interface PageData {
		evaluation: PlayerEvaluation;
	}

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let ev: PlayerEvaluation = $state(data.evaluation);

	let level = $derived(levelById(ev.level_id) ?? FARP_LEVELS[0]);
	let dateLabel = $derived((ev.completed_at ?? ev.created_at).slice(0, 10));
	let pending = $derived(ev.status === 'queued' || ev.status === 'running');
	let percent = $derived(Math.round((ev.progress ?? 0) * 100));

	let outcomes = $derived(ev.results?.outcomes ?? []);
	let successRate = $derived(ev.results?.success_rate ?? 0);
	let isPilot = $derived((ev.level_number ?? 1) === 3);

	let counts = $derived.by(() => {
		const tally = { win: 0, lose: 0, timeout: 0 };
		for (const outcome of outcomes) {
			if (outcome === 'win') tally.win++;
			else if (outcome === 'lose') tally.lose++;
			else tally.timeout++;
		}
		return tally;
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
		const hits = times.filter((time) => time >= 0).length;
		return Math.round((1000 * hits) / times.length) / 10;
	}

	function mean(times: number[]): number | null {
		const hits = times.filter((time) => time >= 0);
		if (hits.length === 0) return null;
		return Math.round((10 * hits.reduce((a, b) => a + b, 0)) / hits.length) / 10;
	}

	let selectedTrial: number | null = $state(null);
	let selectedReplay: Replay | null = $state(null);
	let loadedReplays = false;

	function cellClass(outcome: string): string {
		if (outcome === 'win') return 'cell-win';
		if (outcome === 'lose') return 'cell-loss';
		return 'cell-timeout';
	}

	function fmtTime(time: number | undefined): string {
		return time != null && time >= 0 ? `${time}s` : '—';
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
	let selectedSweepTrial: number | null = $state(null);
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
		selectedSweepTrial = null;
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
		selectedSweepTrial = trial;
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
		const interval = setInterval(async () => {
			try {
				const res = await fetch(apiUrl(`/api/evaluations/${ev.id}`));
				if (!res.ok) return;
				const next: PlayerEvaluation = await res.json();
				ev = next;
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

<div class="page">
	<div class="shell page-head">
		<a href={`/gamemodes/levels/${level.slug}`} class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			{level.name} · {level.subtitle}
		</a>
		<h1 class="page-title name-heading">{ev.username}</h1>
		<p class="page-lede">
			{(ev.level_id ?? 'farp').toUpperCase()} · {ev.defender_count ?? ev.placements?.length ?? 0} defenders
			· {ev.trials} trials · {ev.game_version ?? 'v0.0.4'} · evaluated {dateLabel}
		</p>
	</div>

	<div class="shell detail">
		{#if ev.status === 'failed'}
			<div class="notice notice-error">
				Evaluation failed{ev.error ? `: ${ev.error}` : '.'}
			</div>
		{:else}
			{#if pending}
				<div class="card progress-card">
					<p class="progress-text">
						Benchmark running — showing partial results. Full data appears when the headless run
						finishes.
					</p>
					<div class="progress-track">
						<div class="progress-fill" style={`width: ${percent}%`}></div>
					</div>
					<p class="progress-percent">{percent}%</p>
				</div>
			{/if}

			{#if outcomes.length > 0}
				{#if isPilot}
					<div class="headline">
						<div class="headline-value" class:won={pilotOutcome === 'win'} class:lost={pilotOutcome !== 'win'}>
							{pilotOutcome === 'win' ? 'Planet reached' : pilotOutcome === 'lose' ? 'Caught' : 'Out of time'}
						</div>
						<div class="headline-label">Piloted run</div>
					</div>

					<div class="stat-grid">
						<div class="stat">
							<div class="stat-value">{fmtTime(pilotDetect)}</div>
							<div class="stat-label">Detected — a defender first saw the pilot</div>
						</div>
						<div class="stat">
							<div class="stat-value">{fmtTime(pilotCapture)}</div>
							<div class="stat-label">Captured — a defender touched the pilot</div>
						</div>
						<div class="stat">
							<div class="stat-value">{fmtTime(pilotGoal)}</div>
							<div class="stat-label">Reached planet — the pilot got to the goal</div>
						</div>
						<div class="stat">
							<div class="stat-value">{ev.defender_count ?? ev.placements?.length ?? 0}</div>
							<div class="stat-label">Defenders faced</div>
						</div>
					</div>
				{:else}
					<div class="headline">
						<div class="headline-value won">{successRate}%</div>
						<div class="headline-label">Success rate</div>
						<div class="headline-breakdown">
							<span class="tally-win">{counts.win} captured</span>
							<span class="tally-loss">{counts.lose} reached the planet</span>
							{#if counts.timeout > 0}
								<span class="tally-warn">{counts.timeout} timeouts</span>
							{/if}
						</div>
					</div>

					<div class="stat-grid">
						<div class="stat">
							<div class="stat-value">{detectionRate}%</div>
							<div class="stat-label">Detection rate — a defender saw the evader</div>
						</div>
						<div class="stat">
							<div class="stat-value">{captureRate}%</div>
							<div class="stat-label">Capture rate — a defender touched the evader</div>
						</div>
						<div class="stat">
							<div class="stat-value">{meanGoalTime != null ? `${meanGoalTime}s` : '—'}</div>
							<div class="stat-label">Mean time for the evader to reach the planet</div>
						</div>
						<div class="stat">
							<div class="stat-value">{ev.defender_count ?? ev.placements?.length ?? 0}</div>
							<div class="stat-label">Defenders placed</div>
						</div>
					</div>
				{/if}

				<div class="config-grid">
					<div class="card config">
						<h2 class="config-title">Defender config</h2>
						<dl class="config-rows">
							{#each defenderRows as row}
								<dt>{row.label}</dt>
								<dd>{row.value}</dd>
							{/each}
						</dl>
					</div>
					<div class="card config">
						<h2 class="config-title config-title-enemy">Evader config</h2>
						<dl class="config-rows">
							{#each evaderRows as row}
								<dt>{row.label}</dt>
								<dd>{row.value}</dd>
							{/each}
						</dl>
						<p class="config-note">
							{isPilot
								? 'The evader is flown by the player from a chosen point on the outer ring.'
								: 'The evader drives straight at the planet from a random point on the outer ring.'}
						</p>
					</div>
				</div>

				{#if !isPilot}
					<div class="chart-grid">
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

					<section class="block">
						<h2 class="section-title">Runs ({outcomes.length})</h2>
						<p class="block-note">
							Each cell is one trial — green intercepted, red reached the planet. Pick a run to replay
							it.
						</p>
						<div class="cell-grid">
							{#each outcomes as outcome, index}
								<button
									type="button"
									title={`Trial ${index + 1}: ${outcome}`}
									onclick={() => loadReplay(index)}
									class="cell {cellClass(outcome)}"
									class:selected={selectedTrial === index}
									aria-label={`Trial ${index + 1} ${outcome}`}
								>
									{index + 1}
								</button>
							{/each}
						</div>

						{#if selectedReplay}
							<div class="card replay">
								<div class="replay-meta">
									Trial {(selectedTrial ?? 0) + 1} · {selectedReplay.outcome} · detected {fmtTime(selectedReplay.detection_time)}
									· captured {fmtTime(selectedReplay.capture_time)} · reached planet {fmtTime(selectedReplay.goal_time)}
								</div>
								<FarpReplay replay={selectedReplay} />
							</div>
						{/if}
					</section>
				{:else if selectedReplay}
					<section class="block">
						<h2 class="section-title">Recorded flight</h2>
						<p class="block-note">
							The player's own flight, rendered from the movement recorded in game.
						</p>
						<div class="card replay">
							<div class="replay-meta">
								{selectedReplay.outcome} · detected {fmtTime(selectedReplay.detection_time)} · captured
								{fmtTime(selectedReplay.capture_time)} · reached planet {fmtTime(selectedReplay.goal_time)}
							</div>
							<FarpReplay replay={selectedReplay} />
						</div>
					</section>
				{/if}

				{#if sweepRuns.length > 0}
					<section class="block">
						<h2 class="section-title">Ring sweep runs ({sweepRuns.length})</h2>
						<p class="block-note">
							Each ring size is simulated repeatedly — n defenders placed in a circle around the
							target, the ring rotated to a seeded random angle each trial, against a fixed enemy
							spawn. Pick an n to replay it.
						</p>
						<div class="cell-grid">
							{#each sweepRuns as run}
								<button
									type="button"
									title={`n=${run.n}: ${run.outcome}`}
									onclick={() => loadSweepReplay(run.n)}
									class="cell {cellClass(run.outcome)}"
									class:selected={selectedN === run.n}
									aria-label={`n ${run.n} ${run.outcome}`}
								>
									{run.n}
								</button>
							{/each}
						</div>

						{#if sweepTrials.length > 0}
							<p class="block-note">
								Each n is simulated multiple times with the ring rotated to a different angle. Pick a
								trial to replay it.
							</p>
							<div class="cell-grid">
								{#each sweepTrials as trial}
									<button
										type="button"
										title={`n=${selectedN} trial ${trial.trial + 1}: ${trial.outcome}`}
										onclick={() => loadSweepTrialReplay(selectedN as number, trial.trial)}
										class="cell {cellClass(trial.outcome)}"
										class:selected={selectedSweepTrial === trial.trial}
										aria-label={`trial ${trial.trial + 1} ${trial.outcome}`}
									>
										{trial.trial + 1}
									</button>
								{/each}
							</div>
						{/if}

						{#if selectedSweepReplay}
							<div class="card replay">
								<div class="replay-meta">
									n = {selectedN} defenders{selectedSweepTrial !== null
										? ` · trial ${selectedSweepTrial + 1}`
										: ''} · {selectedSweepReplay.outcome} · detected {fmtTime(selectedSweepReplay.detection_time)}
									· captured {fmtTime(selectedSweepReplay.capture_time)} · reached planet {fmtTime(selectedSweepReplay.goal_time)}
								</div>
								<FarpReplay replay={selectedSweepReplay} />
							</div>
						{/if}
					</section>
				{/if}

				<section class="block">
					<h2 class="section-title">{isPilot ? 'Opponent algorithm' : 'Defender algorithm'}</h2>
					<div class="card algorithm">
						<AlgorithmView scripts={ev.algorithm} />
					</div>
				</section>
			{:else if !pending}
				<div class="notice">No benchmark data available.</div>
			{/if}
		{/if}
	</div>
</div>

<style>
	.name-heading {
		margin-top: 1.25rem;
	}

	.detail {
		padding-bottom: 6rem;
	}

	.progress-card {
		padding: 1.5rem;
		margin-bottom: 2rem;
	}

	.progress-text {
		font-size: 0.9rem;
		color: var(--color-dim);
		text-align: center;
	}

	.progress-track {
		height: 8px;
		margin-top: 1rem;
		background: var(--color-ink);
		border: 1px solid var(--color-line);
		border-radius: 4px;
		overflow: hidden;
	}

	.progress-fill {
		height: 100%;
		background: var(--color-brand);
		transition: width 0.4s;
	}

	.progress-percent {
		margin-top: 0.5rem;
		text-align: center;
		font-size: 0.8rem;
		color: var(--color-faint);
	}

	.headline {
		margin-bottom: 2rem;
	}

	.headline-value {
		font-size: clamp(2rem, 6vw, 3.2rem);
		font-weight: 600;
		line-height: 1;
	}

	.headline-value.won {
		color: var(--color-win);
	}

	.headline-value.lost {
		color: var(--color-loss);
	}

	.headline-label {
		margin-top: 0.5rem;
		font-size: 0.75rem;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--color-faint);
	}

	.headline-breakdown {
		display: flex;
		flex-wrap: wrap;
		gap: 1.25rem;
		margin-top: 1rem;
		font-size: 0.875rem;
	}

	.tally-win {
		color: var(--color-win);
	}

	.tally-loss {
		color: var(--color-loss);
	}

	.tally-warn {
		color: var(--color-warn);
	}

	.stat-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr));
		gap: 0.75rem;
	}

	.config-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
		gap: 0.75rem;
		margin-top: 2.5rem;
	}

	.config {
		padding: 1.4rem;
	}

	.config-title {
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--color-brand);
	}

	.config-title-enemy {
		color: var(--color-loss);
	}

	.config-rows {
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 0.5rem 1rem;
		margin-top: 1rem;
		font-size: 0.875rem;
	}

	.config-rows dt {
		color: var(--color-dim);
	}

	.config-rows dd {
		text-align: right;
		color: var(--color-heading);
	}

	.config-note {
		margin-top: 1rem;
		font-size: 0.78rem;
		line-height: 1.55;
		color: var(--color-faint);
	}

	.chart-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(24rem, 1fr));
		gap: 1rem;
		margin-top: 2.5rem;
	}

	.block {
		margin-top: 3rem;
	}

	.block-note {
		margin-top: 0.5rem;
		max-width: 48rem;
		font-size: 0.85rem;
		line-height: 1.6;
		color: var(--color-dim);
	}

	.cell-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
		max-width: 48rem;
		margin-top: 1.25rem;
	}

	.cell {
		width: 2.5rem;
		height: 2.5rem;
		border: 2px solid transparent;
		border-radius: 0;
		color: #ffffff;
		font-size: 0.72rem;
		cursor: pointer;
		transition:
			filter 0.15s,
			border-color 0.15s;
	}

	.cell:hover {
		filter: brightness(1.25);
	}

	.cell.selected {
		border-color: var(--color-heading);
	}

	.cell-win {
		background: color-mix(in srgb, var(--color-win) 70%, #000000);
	}

	.cell-loss {
		background: color-mix(in srgb, var(--color-loss) 70%, #000000);
	}

	.cell-timeout {
		background: color-mix(in srgb, var(--color-warn) 70%, #000000);
	}

	.replay {
		max-width: 54rem;
		margin-top: 1.25rem;
		padding: 1.25rem;
	}

	.replay-meta {
		margin-bottom: 0.85rem;
		font-size: 0.78rem;
		color: var(--color-faint);
	}

	.algorithm {
		margin-top: 1.25rem;
		padding: 1.4rem;
		overflow-x: auto;
	}
</style>
