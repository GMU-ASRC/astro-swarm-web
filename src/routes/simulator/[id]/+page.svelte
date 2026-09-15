<script lang="ts">
	import Icon from '@iconify/svelte';
	import SimulatorLogic from '$lib/components/SimulatorLogic.svelte';
	import SimulatorReplay from '$lib/components/SimulatorReplay.svelte';
	import { apiUrl } from '$lib/ts/api';
	import type { SimulatorEntry, SimulatorReplay as SimulatorReplayData } from '$lib/ts/simulator';
	import { PIXELS_PER_METER, durationLabel, shortDate } from '$lib/ts/simulator';

	interface PageData {
		entry: SimulatorEntry;
	}

	let { data }: { data: PageData } = $props();

	let entry: SimulatorEntry = $derived(data.entry);

	let replay = $state<SimulatorReplayData | null>(null);
	let replayError = $state(false);

	$effect(() => {
		const entryId = entry.id;
		let active = true;
		replay = null;
		replayError = false;
		fetch(apiUrl(`/api/simulator/entries/${entryId}/replay`))
			.then((response) => {
				if (!response.ok) throw new Error(`HTTP ${response.status}`);
				return response.json() as Promise<SimulatorReplayData>;
			})
			.then((loaded) => {
				if (active) replay = loaded;
			})
			.catch((err) => {
				console.error('Error fetching simulator replay:', err);
				if (active) replayError = true;
			});
		return () => {
			active = false;
		};
	});

	const stats = $derived([
		{ label: 'Duration', value: durationLabel(entry.duration_seconds) },
		{ label: 'Species', value: String(entry.species.length) },
		{ label: 'Robots placed', value: String(entry.placement_count) },
		{ label: 'Peak robots', value: String(entry.peak_robot_count) },
		{ label: 'Spawn zones', value: String(entry.spawn_zones.length) },
		{ label: 'Obstacles', value: String(entry.obstacles.length) },
		{
			label: 'Arena',
			value: `${(entry.arena_width / PIXELS_PER_METER).toFixed(0)} × ${(entry.arena_height / PIXELS_PER_METER).toFixed(0)} m`
		}
	]);
</script>

<svelte:head>
	<title>{entry.title} by {entry.username} — Simulator — AstroSwarm</title>
	<meta
		name="description"
		content={entry.description || `A swarm simulator run by ${entry.username} with ${entry.species.length} species.`}
	/>
</svelte:head>

<div class="page">
	<div class="shell shell-wide page-head">
		<a href="/gamemodes/simulator" class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			All simulator entries
		</a>
		<h1 class="page-title entry-heading">{entry.title}</h1>
		<p class="entry-id">{entry.id}</p>
		<p class="page-lede">
			by {entry.username} · {shortDate(entry.created_at)}
			{#if entry.game_version}
				· {entry.game_version}
			{/if}
		</p>
		{#if entry.description}
			<p class="entry-description">{entry.description}</p>
		{/if}
	</div>

	<div class="shell shell-wide report">
		<div class="stat-grid">
			{#each stats as stat}
				<div class="stat">
					<div class="stat-value">{stat.value}</div>
					<div class="stat-label">{stat.label}</div>
				</div>
			{/each}
		</div>

		<section class="block">
			<h2 class="section-title">Replay</h2>
			{#if replay}
				<SimulatorReplay {entry} {replay} />
			{:else if replayError}
				<div class="notice notice-error">Unable to load the recording for this entry.</div>
			{:else}
				<div class="notice">Loading recording...</div>
			{/if}
		</section>

		<section class="block">
			<h2 class="section-title">Logic</h2>
			<p class="block-note">
				Every species in the run with its settings and blocks, plus the arena program that controls
				the spawn zones.
			</p>
			<SimulatorLogic {entry} />
		</section>
	</div>
</div>

<style>
	.entry-heading {
		margin-top: 1.25rem;
	}

	.entry-id {
		margin-top: 0.6rem;
		font-family: ui-monospace, monospace;
		font-size: 0.72rem;
		color: var(--color-faint);
		overflow-wrap: anywhere;
	}

	.entry-description {
		margin-top: 0.9rem;
		max-width: 48rem;
		font-size: 0.9rem;
		line-height: 1.6;
		color: var(--color-dim);
		white-space: pre-line;
	}

	.report {
		padding-bottom: 6rem;
	}

	.stat-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(9.5rem, 1fr));
		gap: 0.75rem;
	}

	.block {
		margin-top: 2.75rem;
	}

	.block .section-title {
		margin-bottom: 1rem;
	}

	.block-note {
		margin: -0.5rem 0 1rem;
		max-width: 48rem;
		font-size: 0.82rem;
		color: var(--color-faint);
	}
</style>
