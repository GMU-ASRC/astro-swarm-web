<script lang="ts">
	import Icon from '@iconify/svelte';
	import Spaceship from '$lib/components/Spaceship.svelte';
	import type { ShipVariant } from '$lib/ts/ships';

	const modes: {
		href: string;
		variant: ShipVariant;
		name: string;
		tagline: string;
		blurb: string;
		facts: string[];
	}[] = [
		{
			href: '/gamemodes/levels',
			variant: 'blue',
			name: 'Levels',
			tagline: 'Single player · three levels',
			blurb:
				'Place a ring of defenders around the planet and program how they hunt. Submissions are replayed on dedicated workers and scored on how often the evader is spotted and caught.',
			facts: ['3 levels', 'Headless benchmarks', 'Replays and charts']
		},
		{
			href: '/gamemodes/survive',
			variant: 'green',
			name: 'Survive',
			tagline: 'Two players · three minutes',
			blurb:
				'Two commanders herd wandering ships into planetary defences while waves of evaders close in on both bases. Whoever lets fewest evaders through wins the match.',
			facts: ['Head to head', 'Match reports', 'APM telemetry']
		}
	];
</script>

<svelte:head>
	<title>Game Modes — AstroSwarm</title>
	<meta name="description" content="Browse AstroSwarm game modes and the data behind every recorded run." />
</svelte:head>

<div class="page">
	<div class="shell page-head">
		<h1 class="page-title">Pick a mode</h1>
		<p class="page-lede">
			Every mode records what happened in game and publishes it here. Choose a mode to browse its
			levels, entries, and match reports.
		</p>
	</div>

	<div class="shell modes">
		{#each modes as mode}
			<a href={mode.href} class="card-link mode">
				<div class="mode-art">
					<Spaceship variant={mode.variant} size="88" />
				</div>

				<div class="mode-body">
					<h2 class="mode-name">{mode.name}</h2>
					<p class="mode-tagline">{mode.tagline}</p>
					<p class="mode-blurb">{mode.blurb}</p>

					<div class="mode-facts">
						{#each mode.facts as fact}
							<span class="badge">{fact}</span>
						{/each}
					</div>

					<span class="mode-cta">
						Open mode
						<Icon icon="ph:arrow-right-bold" width="14" />
					</span>
				</div>
			</a>
		{/each}
	</div>
</div>

<style>
	.modes {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(22rem, 1fr));
		gap: 1rem;
		padding-bottom: 6rem;
	}

	.mode {
		display: flex;
		gap: 1.75rem;
		padding: 2rem;
	}

	.mode-art {
		flex-shrink: 0;
	}

	.mode-body {
		min-width: 0;
	}

	.mode-name {
		font-size: 1.4rem;
		font-weight: 600;
	}

	.mode-tagline {
		margin-top: 0.3rem;
		font-size: 0.78rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--color-faint);
	}

	.mode-blurb {
		margin-top: 1rem;
		font-size: 0.9rem;
		line-height: 1.65;
		color: var(--color-dim);
	}

	.mode-facts {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		margin-top: 1.25rem;
	}

	.mode-cta {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin-top: 1.5rem;
		font-size: 0.85rem;
		font-weight: 500;
		color: var(--color-brand-hover);
	}

	@media (max-width: 640px) {
		.mode {
			flex-direction: column;
			gap: 1.25rem;
			padding: 1.5rem;
		}
	}
</style>
