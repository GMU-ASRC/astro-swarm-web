<script lang="ts">
	import { goto } from '$app/navigation';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import ReplayWorkspace from '$lib/components/ReplayWorkspace.svelte';
	import { toneOf, type ReplayGroup } from '$lib/ts/replay';
	import ChartCard from '$lib/components/ChartCard.svelte';
	import { apiUrl } from '$lib/ts/api';
	import { barConfig, lineConfig, headlineRatesConfig, detectionRateConfig, captureRateConfig, combinedRatesConfig, timesConfig } from '$lib/ts/charts';
	import type { PlayerEvaluation, Replay } from '$lib/ts/evaluation';
	import { isPilot as pilotLevel, isSwarm as swarmLevel, isWave as waveLevel } from '$lib/ts/levels';

	let { data } = $props();

	let ev = $state<PlayerEvaluation | null>(null);
	let loading = $state(true);
	let message = $state('');

	let selectedTrial: number | null = $state(null);
	let selectedReplay: Replay | null = $state(null);
	let loadedReplays = false;

	let sweepRuns = $state<{ n: number; outcome: string; detection_time?: number; capture_time?: number; detection_rate?: number; capture_rate?: number; trial_count?: number }[]>([]);
	let sweepTrials = $state<{ trial: number; outcome: string }[]>([]);
	let selectedN: number | null = $state(null);
	let selectedSweepTrial: number | null = $state(null);
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
	let levelNumber = $derived(ev?.level_number ?? 1);
	let isPilot = $derived(pilotLevel(levelNumber));
	let isSwarm = $derived(swarmLevel(levelNumber));
	let isWave = $derived(waveLevel(levelNumber));

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
	let detectionRate = $derived(ev?.results?.detection_rate ?? rateOf(detectionTimes));
	let captureRate = $derived(ev?.results?.capture_rate ?? rateOf(captureTimes));

	function rateOf(times: number[]): number {
		if (times.length === 0) return 0;
		const hits = times.filter((time) => time >= 0).length;
		return Math.round((1000 * hits) / times.length) / 10;
	}

	let pilotTimes = $derived({
		detection: detectionTimes[0] ?? -1,
		capture: captureTimes[0] ?? -1,
		goal: (ev?.results?.goal_times ?? [])[0] ?? -1
	});

	let pilotOutcomeLabel = $derived.by(() => {
		const outcome = outcomes[0] ?? 'timeout';
		if (isSwarm) return outcome === 'win' ? 'Swarm delivered' : 'Out of time';
		if (outcome === 'win') return 'Planet reached';
		return outcome === 'lose' ? 'Caught' : 'Out of time';
	});

	let placementGroups = $derived<ReplayGroup[]>([
		{
			label: isPilot ? 'Flight' : 'Trials',
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

	let placementMeta = $derived.by(() => {
		if (!selectedReplay) return '';
		const label = isPilot ? 'Recorded flight' : `Trial ${(selectedTrial ?? 0) + 1}`;
		const times = `detected: ${fmtTime(selectedReplay.detection_time)} · captured: ${fmtTime(selectedReplay.capture_time)}`;
		if (!isWave) return `${label} · outcome: ${selectedReplay.outcome} · ${times}`;
		const stats = selectedReplay.stats ?? {};
		const evaders = stats.evaders ?? 0;
		return `${label} · ${evaders} ${evaders === 1 ? 'evader' : 'evaders'} per wave · destroyed ${stats.destroyed ?? 0} of ${evaders * 2} · through ${stats.breaches ?? 0} · ${times}`;
	});

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
			? `N = ${selectedN} defenders${selectedSweepTrial !== null ? ` · trial ${selectedSweepTrial + 1}` : ''} · outcome: ${selectedSweepReplay.outcome} · detected: ${fmtTime(selectedSweepReplay.detection_time)} · captured: ${fmtTime(selectedSweepReplay.capture_time)}`
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

	function fmtTime(t: number | undefined): string {
		return t != null && t >= 0 ? `${t.toFixed(2)}s` : '—';
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

	async function loadSweepTrialIndex(n: number) {
		if (!ev) return;
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
		if (!ev) return;
		selectedSweepTrial = trial;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${ev.id}/sweep-replay/${n}/trial/${trial}`));
			if (!res.ok) return;
			selectedSweepReplay = await res.json();
		} catch (err) {
			console.error('Error loading sweep trial replay:', err);
		}
	}

	async function loadSweepReplay(n: number) {
		if (!ev) return;
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

	$effect(() => {
		if (ev && ev.status === 'done' && !isPilot && !loadedSweep) {
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
			sweepTrials = [];
			selectedSweepReplay = null;
			selectedN = null;
			selectedSweepTrial = null;
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
		<p>Benchmark {ev.status === 'queued' ? 'queued' : 'running'} ({Math.round((ev.progress ?? 0) * 100)}%). Results appear when the headless run finishes.</p>
		{#if ev.stage}<p class="meta">{ev.stage}</p>{/if}
	{:else if outcomes.length === 0}
		<p>No benchmark data available.</p>
	{:else}
		{#if isPilot}
			<div class="stat-grid">
				<div class="stat">
					<div class="label">Outcome</div>
					<div>{pilotOutcomeLabel}</div>
				</div>
				<div class="stat">
					<div class="label">Detected</div>
					<div>{fmtTime(pilotTimes.detection)}</div>
				</div>
				<div class="stat">
					<div class="label">Captured</div>
					<div>{fmtTime(pilotTimes.capture)}</div>
				</div>
				<div class="stat">
					<div class="label">Reached planet</div>
					<div>{fmtTime(pilotTimes.goal)}</div>
				</div>
			</div>
		{:else}
			<div class="stat-grid">
				<div class="stat">
					<div class="label">{isWave ? 'Both waves held' : 'Detection rate'}</div>
					<div>{successRate}%</div>
				</div>
				<div class="stat">
					<div class="label">{isWave ? 'Trials held' : 'Intercepts'}</div>
					<div>{counts.win}</div>
				</div>
				<div class="stat">
					<div class="label">{isWave ? 'Trials breached' : 'Planet hits'}</div>
					<div>{counts.lose}</div>
				</div>
				<div class="stat">
					<div class="label">Timeouts</div>
					<div>{counts.timeout}</div>
				</div>
			</div>
			{#if isWave}
				<div class="stat-grid">
					<div class="stat">
						<div class="label">First wave held</div>
						<div>{ev.results?.sequential_rate ?? 0}%</div>
					</div>
					<div class="stat">
						<div class="label">All-at-once wave held</div>
						<div>{ev.results?.simultaneous_rate ?? 0}%</div>
					</div>
					<div class="stat">
						<div class="label">Evaders destroyed</div>
						<div>{ev.results?.evaders_destroyed ?? 0} / {ev.results?.evaders_total ?? 0}</div>
					</div>
					<div class="stat">
						<div class="label">Evader kill rate</div>
						<div>{ev.results?.evader_destroyed_rate ?? 0}%</div>
					</div>
				</div>
			{/if}
		{/if}

		{#if !isPilot}
			<div class="charts">
				<ChartCard config={headlineRatesConfig(detectionRate, captureRate, successRate, outcomes.length)} />
				<ChartCard config={lineConfig(outcomes)} downloadUrl={apiUrl(`/api/evaluations/${ev.id}/chart/line.png`)} />
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
		{/if}

		{#if isPilot}
			<h2>Recorded Flight</h2>
			<p class="meta">The player's own piloted run, rendered from the movement recorded in game.</p>
		{:else if isWave}
			<h2>Run Data ({outcomes.length})</h2>
			<p class="meta">{outcomes.length} trials against the submitted defender scatter. Each one plays both waves back to back — the evaders one after another, then the arena resets and the same evaders come all at once — and is green only if both waves held. Each trial varies the scatter, the spawn angles and the evader count. Click a run to replay it.</p>
		{:else}
			<h2>Run Data ({outcomes.length})</h2>
			<p class="meta">The player's own defender placements against {outcomes.length} random enemy spawns — green intercepted, red reached the planet, yellow timed out. Click a run to replay it.</p>
		{/if}
		<ReplayWorkspace groups={placementGroups} replay={selectedReplay} meta={placementMeta} />

		{#if !isPilot}
			<h2>Ring Sweep Runs ({sweepRuns.length})</h2>
			<p class="meta">Each ring size is simulated repeatedly — n defenders spread around the target, each one dropped at a random angle inside its own slice of the ring and facing a random way, against a per-trial evader approach. Click an n to replay it.</p>
			{#if sweepRuns.length > 0}
				<ReplayWorkspace
					groups={sweepGroups}
					replay={selectedSweepReplay}
					meta={sweepMeta}
					empty="Pick a ring size to replay it."
				/>
			{:else}
				<div class="message">No ring sweep data for this entry. Re-simulate it with the latest simulator build to generate the sweep replays.</div>
			{/if}
		{/if}

		<h2>{isSwarm ? 'Swarm Agent Algorithm' : isPilot ? 'Opponent Algorithm' : 'Defender Algorithm'}</h2>
		<div class="admin-table-wrap">
			<AlgorithmView scripts={ev.algorithm} />
		</div>
	{/if}
{/if}

<style>
	hr {
		border: none;
		border-top: 1px solid var(--color-line);
		margin: 1.75rem 0;
	}
</style>
