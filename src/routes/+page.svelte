<script lang="ts">
	import Icon from '@iconify/svelte';
	import LiveBattle from '$lib/components/LiveBattle.svelte';
	import DownloadButton from '$lib/components/DownloadButton.svelte';
	import Spaceship from '$lib/components/Spaceship.svelte';
	import { REPO } from '$lib/ts/github';
	import type { ShipVariant } from '$lib/ts/ships';

	const modes: { href: string; variant: ShipVariant; name: string; tagline: string; blurb: string }[] = [
		{
			href: '/gamemodes/levels',
			variant: 'blue',
			name: 'Levels',
			tagline: 'Three levels · benchmarked',
			blurb:
				'Place a ring of defenders around the planet and program how they hunt. Every submission is replayed headlessly over many trials and scored on detection and capture.'
		},
		{
			href: '/gamemodes/survive',
			variant: 'green',
			name: 'Survive',
			tagline: 'Two players · three minutes',
			blurb:
				'Two commanders herd wandering ships into planetary defences while waves of evaders close in. Fewest evaders through the line wins.'
		}
	];

	const fleet: ShipVariant[] = ['blue', 'green', 'gold', 'purple', 'red'];
</script>

<svelte:head>
	<title>AstroSwarm</title>
</svelte:head>

<section class="hero">
	<LiveBattle />

	<div class="hero-content">
		<h1 class="hero-title">ASTROSWARM</h1>
		<p class="hero-tagline">
			Design your fleet, program its behaviour, and send it against the swarm.
		</p>

		<div class="hero-actions">
			<DownloadButton size="large" />
			<a
				href={`https://github.com/${REPO}`}
				target="_blank"
				rel="noreferrer"
				class="btn btn-lg"
			>
				<Icon icon="ph:github-logo-fill" width="18" />
				View source
			</a>
		</div>
	</div>
</section>

<section class="shell shell-wide section">
	<h2 class="page-title">Pick your fight</h2>

	<div class="mode-grid">
		{#each modes as mode}
			<a href={mode.href} class="card-link mode-card">
				<div class="mode-art">
					<Spaceship variant={mode.variant} size="72" />
				</div>
				<div>
					<h3 class="mode-name">{mode.name}</h3>
					<p class="mode-tagline">{mode.tagline}</p>
					<p class="mode-blurb">{mode.blurb}</p>
					<span class="mode-cta">
						View data
						<Icon icon="ph:arrow-right-bold" width="14" />
					</span>
				</div>
			</a>
		{/each}
	</div>
</section>

<style>
	.hero {
		position: relative;
		z-index: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
		padding: 6rem 2rem 4rem;
		text-align: center;
		overflow: hidden;
		border-bottom: 1px solid var(--color-line);
	}

	.hero-content {
		position: relative;
		z-index: 2;
		width: 100%;
		max-width: 60rem;
	}

	.hero-title {
		font-family: var(--font-display);
		font-size: clamp(2.9rem, 10vw, 6.5rem);
		line-height: 1.05;
		letter-spacing: 0.03em;
		padding-left: 0.03em;
		color: var(--color-heading);
	}

	.hero-tagline {
		margin: 1.5rem auto 0;
		max-width: 32rem;
		font-size: 1.05rem;
		line-height: 1.6;
		color: var(--color-dim);
	}

	.hero-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
		justify-content: center;
		margin-top: 2.5rem;
	}

	.hero-fleet {
		position: relative;
		z-index: 2;
		display: flex;
		gap: 1.75rem;
		margin-top: 4.5rem;
		opacity: 0.75;
	}

	.section {
		padding-top: 5rem;
		padding-bottom: 1rem;
		position: relative;
		z-index: 1;
	}

	.section:last-of-type {
		padding-bottom: 6rem;
	}

	.mode-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(20rem, 1fr));
		gap: 1rem;
		margin-top: 2rem;
	}

	.mode-card {
		display: flex;
		gap: 1.5rem;
		padding: 1.75rem;
	}

	.mode-art {
		flex-shrink: 0;
	}

	.mode-name {
		font-size: 1.25rem;
		font-weight: 600;
	}

	.mode-tagline {
		margin-top: 0.25rem;
		font-size: 0.8rem;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--color-faint);
	}

	.mode-blurb {
		margin-top: 0.9rem;
		font-size: 0.9rem;
		line-height: 1.6;
		color: var(--color-dim);
	}

	.mode-cta {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin-top: 1.1rem;
		font-size: 0.85rem;
		font-weight: 500;
		color: var(--color-brand-hover);
	}

	@media (max-width: 640px) {
		.mode-card {
			flex-direction: column;
			gap: 1rem;
		}

		.hero-fleet {
			gap: 1rem;
			margin-top: 3rem;
		}
	}
</style>
