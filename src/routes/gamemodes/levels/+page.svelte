<script lang="ts">
	import Icon from '@iconify/svelte';
	import Spaceship from '$lib/components/Spaceship.svelte';
	import type { PlayerListItem } from '$lib/ts/evaluation';
	import { FARP_LEVELS, canonicalLevelId, type LevelInfo } from '$lib/ts/levels';

	interface PageData {
		playersPromise: Promise<{ players: PlayerListItem[]; apiError: boolean }>;
	}

	let { data }: { data: PageData } = $props();

	let players = $state<PlayerListItem[]>([]);
	let apiError = $state(false);
	let loading = $state(true);

	$effect(() => {
		let active = true;
		loading = true;
		data.playersPromise.then((result) => {
			if (!active) return;
			players = result.players;
			apiError = result.apiError;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	function entriesFor(level: LevelInfo): PlayerListItem[] {
		return players.filter(
			(player) => canonicalLevelId(player.level_id) === level.id && player.status !== 'cancelled'
		);
	}

	function bestRate(entries: PlayerListItem[]): number | null {
		const rates = entries
			.map((entry) => entry.success_rate)
			.filter((rate): rate is number => rate !== null && rate !== undefined);
		return rates.length > 0 ? Math.max(...rates) : null;
	}
</script>

<svelte:head>
	<title>Levels — AstroSwarm</title>
	<meta name="description" content="Per-level benchmark data for player algorithms in AstroSwarm." />
</svelte:head>

<div class="page">
	<div class="shell page-head">
		<a href="/gamemodes" class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			All game modes
		</a>
		<h1 class="page-title level-heading">Levels</h1>
		<p class="page-lede">
			Defenders hold a planet against an evader breaking in from the outer rim. A detection is a
			sighting, a capture is a collision, and reaching the planet ends the run. Pick a level to see
			every commander's entry.
		</p>
	</div>

	<div class="shell levels">
		{#if apiError}
			<div class="notice notice-error">Communication error. Unable to load level data.</div>
		{/if}

		{#each FARP_LEVELS as level}
			{@const entries = entriesFor(level)}
			{@const best = bestRate(entries)}
			<a href={`/gamemodes/levels/${level.slug}`} class="card-link level">
				<div class="level-art">
					<Spaceship variant={level.variant} size="64" />
				</div>

				<div class="level-body">
					<div class="level-title">
						<h2 class="level-name">{level.name}</h2>
						<span class="badge" class:badge-warn={level.piloted}>{level.subtitle}</span>
					</div>
					<p class="level-summary">{level.summary}</p>
				</div>

				<div class="level-numbers">
					<div>
						<div class="level-value">{loading ? '—' : entries.length}</div>
						<div class="level-caption">entries</div>
					</div>
					<div>
						<div class="level-value">{loading || best === null ? '—' : `${best}%`}</div>
						<div class="level-caption">best {level.rateLabel}</div>
					</div>
					<Icon icon="ph:arrow-right-bold" width="16" color="var(--color-brand)" />
				</div>
			</a>
		{/each}
	</div>
</div>

<style>
	.level-heading {
		margin-top: 1.25rem;
	}

	.levels {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding-bottom: 6rem;
	}

	.level {
		display: flex;
		align-items: center;
		gap: 1.5rem;
		padding: 1.5rem;
	}

	.level-art {
		flex-shrink: 0;
	}

	.level-body {
		flex: 1;
		min-width: 0;
	}

	.level-title {
		display: flex;
		align-items: center;
		gap: 0.6rem;
	}

	.level-name {
		font-size: 1.2rem;
		font-weight: 600;
	}

	.level-summary {
		margin-top: 0.6rem;
		font-size: 0.875rem;
		line-height: 1.6;
		color: var(--color-dim);
	}

	.level-numbers {
		display: flex;
		align-items: center;
		gap: 2rem;
		flex-shrink: 0;
		text-align: right;
	}

	.level-value {
		font-size: 1.35rem;
		font-weight: 600;
		color: var(--color-heading);
	}

	.level-caption {
		margin-top: 0.2rem;
		font-size: 0.72rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--color-faint);
	}

	@media (max-width: 800px) {
		.level {
			flex-wrap: wrap;
			gap: 1.25rem;
		}

		.level-numbers {
			width: 100%;
			justify-content: flex-start;
			text-align: left;
			padding-top: 1.25rem;
			border-top: 1px solid var(--color-line);
		}
	}
</style>
