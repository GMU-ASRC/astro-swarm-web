<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '@iconify/svelte';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import ReplayWorkspace from '$lib/components/ReplayWorkspace.svelte';
	import { toneOf, type ReplayGroup } from '$lib/ts/replay';
	import ChartCard from '$lib/components/ChartCard.svelte';
	import { isPilot as pilotLevel, isSwarm as swarmLevel, isWave as waveLevel } from '$lib/ts/levels';
	import { apiUrl } from '$lib/ts/api';
	import { barConfig, lineConfig, headlineRatesConfig, detectionRateConfig, captureRateConfig, combinedRatesConfig, timesConfig, waveCaptureConfig, waveDetectionConfig } from '$lib/ts/charts';
	import type { PlayerEvaluation, Replay } from '$lib/ts/evaluation';
	import { EVADER_CONFIG, PILOT_EVADER_CONFIG, LEADER_CONFIG, configRows, defenderConfig } from '$lib/ts/shipConfig';
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
	let levelNumber = $derived(ev.level_number ?? 1);
	let isSwarm = $derived(swarmLevel(levelNumber));
	let isPilot = $derived(pilotLevel(levelNumber));
	let isWave = $derived(waveLevel(levelNumber));

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
	let evaderRows = $derived(
		configRows(isSwarm ? LEADER_CONFIG : isPilot ? PILOT_EVADER_CONFIG : EVADER_CONFIG)
	);
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

	let stats = $derived(ev.results?.stats ?? selectedReplay?.stats ?? {});

	let placementGroups = $derived<ReplayGroup[]>([
		{
			label: 'Trials',
			cells: outcomes.map((outcome, index) => ({
				id: index,
				label: index + 1,
				tone: toneOf(outcome),
				title: `Trial ${index + 1}: ${outcome}`,
				selected: selectedTrial === index,
				select: () => loadReplay(index)
			}))
		}
	]);

	let placementMeta = $derived(
		selectedReplay
			? `Trial ${(selectedTrial ?? 0) + 1} · ${selectedReplay.outcome} · detected ${fmtTime(selectedReplay.detection_time)} · captured ${fmtTime(selectedReplay.capture_time)} · reached planet ${fmtTime(selectedReplay.goal_time)}`
			: ''
	);

	let sweepGroups = $derived<ReplayGroup[]>([
		{
			label: 'Ring size (n)',
			cells: sweepRuns.map((run) => ({
				id: `n${run.n}`,
				label: run.n,
				tone: sweepTone(run),
				title: `n=${run.n}: ${sweepLabel(run)}`,
				selected: selectedN === run.n,
				select: () => loadSweepReplay(run.n)
			}))
		},
		{
			label: selectedN != null ? `Trials for n = ${selectedN}` : 'Trials',
			cells: sweepTrials.map((trial) => ({
				id: `t${trial.trial}`,
				label: trial.trial + 1,
				tone: toneOf(trial.outcome),
				title: `n=${selectedN} trial ${trial.trial + 1}: ${trial.outcome}`,
				selected: selectedSweepTrial === trial.trial,
				select: () => loadSweepTrialReplay(selectedN as number, trial.trial)
			}))
		}
	]);

	let sweepMeta = $derived(
		selectedSweepReplay
			? `n = ${selectedN} defenders${selectedSweepTrial !== null ? ` · trial ${selectedSweepTrial + 1}` : ''} · ${selectedSweepReplay.outcome} · detected ${fmtTime(selectedSweepReplay.detection_time)} · captured ${fmtTime(selectedSweepReplay.capture_time)} · reached planet ${fmtTime(selectedSweepReplay.goal_time)}`
			: ''
	);

	function sweepTone(run: { outcome: string; capture_rate?: number }): 'win' | 'loss' | 'timeout' {
		// Colour an n by how its trials went overall, not by the one trial kept
		// for replay.
		if (run.capture_rate != null) return run.capture_rate > 50 ? 'win' : 'loss';
		return toneOf(run.outcome);
	}

	function sweepLabel(run: { outcome: string; capture_rate?: number }): string {
		if (run.capture_rate != null) return `${run.capture_rate}% of trials captured`;
		return run.outcome;
	}

	function fmtTime(time: number | undefined): string {
		return time != null && time >= 0 ? `${time.toFixed(2)}s` : '—';
	}

	function fmtNumber(value: number | undefined, digits = 3): string {
		return value != null && value >= 0 ? value.toFixed(digits) : '—';
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
			{(ev.level_id ?? 'farp').toUpperCase()} · {ev.defender_count ?? ev.placements?.length ?? 0}
			{isSwarm ? 'agents' : 'defenders'}
			{#if !isPilot}· {ev.trials} trials{/if}
			· {ev.game_version ?? 'v0.0.4'} · recorded {dateLabel}
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
				{#if isSwarm}
					<div class="headline">
						<div class="headline-value" class:won={pilotOutcome === 'win'} class:lost={pilotOutcome !== 'win'}>
							{pilotOutcome === 'win' ? 'Swarm delivered' : 'Out of time'}
						</div>
						<div class="headline-label">Piloted swarm run</div>
					</div>

					<div class="stat-grid">
						<div class="stat">
							<div class="stat-value">{fmtTime(stats.merge_time)}</div>
							<div class="stat-label">Merged — the two groups became one swarm</div>
						</div>
						<div class="stat">
							<div class="stat-value">{fmtTime(stats.deliver_time)}</div>
							<div class="stat-label">Delivered — the mill reached the planet</div>
						</div>
						<div class="stat">
							<div class="stat-value">{fmtTime(stats.escape_time)}</div>
							<div class="stat-label">Escaped — the leader left the mill intact</div>
						</div>
						<div class="stat">
							<div class="stat-value">{fmtNumber(stats.minimum_loss)}</div>
							<div class="stat-label">Minimum loss — goal distance plus milling error</div>
						</div>
						<div class="stat">
							<div class="stat-value">{fmtNumber(stats.circliness)}</div>
							<div class="stat-label">Circliness at the end — 1.0 is a clean mill</div>
						</div>
						<div class="stat">
							<div class="stat-value">{stats.agents ?? ev.defender_count ?? 0}</div>
							<div class="stat-label">Agents herded across {stats.groups ?? 2} groups</div>
						</div>
					</div>
				{:else if isPilot}
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
						<h2 class="config-title">{isSwarm ? 'Swarm agent config' : 'Defender config'}</h2>
						<dl class="config-rows">
							{#each defenderRows as row}
								<dt>{row.label}</dt>
								<dd>{row.value}</dd>
							{/each}
						</dl>
					</div>
					<div class="card config">
						<h2 class="config-title config-title-enemy">{isSwarm ? 'Leader config' : 'Evader config'}</h2>
						<dl class="config-rows">
							{#each evaderRows as row}
								<dt>{row.label}</dt>
								<dd>{row.value}</dd>
							{/each}
						</dl>
						<p class="config-note">
							{isSwarm
								? 'The leader is flown by the player. The agents cannot tell it apart from one of their own, which is what lets it steer the mill.'
								: isPilot
									? 'The evader is flown by the player from a chosen point on the outer ring.'
									: 'The evader drives straight at the planet from a random point on the outer ring.'}
						</p>
					</div>
				</div>

				{#if !isPilot}
					<div class="chart-grid">
						<ChartCard config={headlineRatesConfig(detectionRate, captureRate, successRate, outcomes.length)} />
						<ChartCard config={lineConfig(outcomes)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/line.png`)} />
						{#if isWave && (ev.results?.trial_destroyed ?? []).length > 0}
							<ChartCard
								config={waveCaptureConfig(ev.results?.trial_destroyed ?? [], ev.results?.trial_evaders ?? [])}
							/>
						{/if}
						{#if isWave && (ev.results?.trial_detected_first ?? []).length > 0}
							<ChartCard
								config={waveDetectionConfig(
									ev.results?.trial_detected_first ?? [],
									ev.results?.trial_detected_second ?? []
								)}
							/>
						{/if}
						<ChartCard config={barConfig(outcomes)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/bar.png`)} />
						{#if sweepRuns.length > 0}
							<ChartCard config={detectionRateConfig(sweepRuns)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/sweep.png`)} />
							<ChartCard config={captureRateConfig(sweepRuns)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/capture.png`)} />
							<ChartCard config={combinedRatesConfig(sweepRuns)} />
						{/if}
						{#if detectionTimes.length > 0}
							<ChartCard config={timesConfig(detectionTimes, captureTimes)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/times.png`)} />
						{/if}
					</div>

					<section class="block">
						<h2 class="section-title">Runs ({outcomes.length})</h2>
						<p class="block-note">
							{#if isWave}
								Each cell is one trial: both waves back to back, the evaders one after another
								and then all at once. Green held both, red let one through.
							{:else}
								Each cell is one trial — green intercepted, red reached the planet. Pick a run to
								replay it.
							{/if}
						</p>
						<ReplayWorkspace
							groups={placementGroups}
							replay={selectedReplay}
							meta={placementMeta}
						/>
					</section>
				{:else if selectedReplay}
					<section class="block">
						<h2 class="section-title">Recorded flight</h2>
						<p class="block-note">
							The player's own flight, rendered from the movement recorded in game.
						</p>
						<div class="card replay">
							<div class="replay-meta">
								{#if isSwarm}
									{selectedReplay.outcome} · merged {fmtTime(stats.merge_time)} · delivered
									{fmtTime(stats.deliver_time)} · escaped {fmtTime(stats.escape_time)}
								{:else}
									{selectedReplay.outcome} · detected {fmtTime(selectedReplay.detection_time)} · captured
									{fmtTime(selectedReplay.capture_time)} · reached planet {fmtTime(selectedReplay.goal_time)}
								{/if}
							</div>
							<FarpReplay replay={selectedReplay} mode={isSwarm ? 'swarm' : 'defence'} />
						</div>
					</section>
				{/if}

				{#if sweepRuns.length > 0}
					<section class="block">
						<h2 class="section-title">Ring sweep runs ({sweepRuns.length})</h2>
						<p class="block-note">
							Each ring size is simulated repeatedly — n defenders spread around the target, each
							one dropped at a random angle inside its own slice of the ring and facing a random
							way. A green n means most of its trials captured the evader. Pick an n to replay it.
						</p>
						<ReplayWorkspace
							groups={sweepGroups}
							replay={selectedSweepReplay}
							meta={sweepMeta}
							empty="Pick a ring size to replay it."
						/>
					</section>
				{/if}

				<section class="block">
					<h2 class="section-title">
						{isSwarm ? 'Swarm agent algorithm' : isPilot ? 'Opponent algorithm' : 'Defender algorithm'}
					</h2>
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
