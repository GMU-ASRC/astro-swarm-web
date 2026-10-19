<script lang="ts">
	import Icon from '@iconify/svelte';
	import Spaceship from '$lib/components/Spaceship.svelte';
	import TrainingLeaderboard from '$lib/components/TrainingLeaderboard.svelte';
	import { TRAINING_LEVEL } from '$lib/ts/levels';
	import type { TrainingScoreRow } from '$lib/ts/training';

	interface PageData {
		leaderboardPromise: Promise<{ rows: TrainingScoreRow[]; apiError: boolean }>;
	}

	let { data }: { data: PageData } = $props();

	const level = TRAINING_LEVEL;

	let rows = $state<TrainingScoreRow[]>([]);
	let apiError = $state(false);
	let loading = $state(true);

	$effect(() => {
		let active = true;
		loading = true;
		data.leaderboardPromise.then((result) => {
			if (!active) return;
			rows = result.rows;
			apiError = result.apiError;
			loading = false;
		});
		return () => {
			active = false;
		};
	});
</script>

<svelte:head>
	<title>{level.name} · {level.subtitle} — AstroSwarm</title>
	<meta name="description" content={level.summary} />
</svelte:head>

<div class="page">
	<div class="shell shell-wide page-head">
		<a href="/gamemodes/levels" class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			Levels
		</a>

		<div class="head-row">
			<div>
				<h1 class="page-title level-heading">{level.name} · {level.subtitle}</h1>
				<p class="page-lede">{level.summary}</p>
			</div>
			<Spaceship variant={level.variant} size="80" />
		</div>

		<h2 class="section-heading">
			<Icon icon="ph:trophy-bold" width="16" />
			Circliness Leaderboard
		</h2>
	</div>

	<div class="shell shell-wide panel">
		{#if loading}
			<div class="notice">Loading leaderboard...</div>
		{:else if apiError}
			<div class="notice notice-error">Communication error. Unable to load the leaderboard.</div>
		{:else}
			<TrainingLeaderboard {rows} />
		{/if}
	</div>
</div>

<style>
	.panel {
		padding-bottom: 6rem;
	}

	.head-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 2rem;
	}

	.level-heading {
		margin-top: 1.25rem;
	}

	.section-heading {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-top: 1.5rem;
		font-size: 1rem;
		font-weight: 600;
		color: var(--color-heading);
	}

	@media (max-width: 800px) {
		.head-row :global(.spaceship) {
			display: none;
		}
	}
</style>
