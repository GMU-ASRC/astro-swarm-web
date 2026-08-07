<script lang="ts">
	import Icon from '@iconify/svelte';
	import ChartCard from '$lib/components/ChartCard.svelte';
	import type { SurviveMatch, SurvivePlayer } from '$lib/ts/survive';
	import { PLAYER_COLORS, apmChartConfig, clockLabel } from '$lib/ts/survive';

	interface PageData {
		match: SurviveMatch;
	}

	let { data }: { data: PageData } = $props();

	let match: SurviveMatch = $derived(data.match);

	let players = $derived(match.players ?? []);
	let hasSeries = $derived((match.apm_times?.length ?? 0) > 1);
	let chartConfig = $derived(apmChartConfig(match));
	let dateLabel = $derived(match.created_at.slice(0, 10));

	let headline = $derived(match.outcome === 'tie' ? 'Tie match' : `${match.winner ?? 'Winner'} wins`);

	const statRows: { key: keyof SurvivePlayer; label: string; hint: string }[] = [
		{
			key: 'evaders_through',
			label: 'Evaders through',
			hint: 'Red ships that reached this planet — fewest wins'
		},
		{
			key: 'defenders_at_end',
			label: 'Defenders at the buzzer',
			hint: 'Blue ships still closer to this planet than the rival one'
		},
		{ key: 'herded_total', label: 'Ships herded', hint: 'Purple ships turned blue by this player' },
		{ key: 'freezes_used', label: 'Freezes used', hint: 'Of the two charges held for the match' },
		{ key: 'actions', label: 'Actions', hint: 'Input state changes plus power up activations' },
		{ key: 'apm_average', label: 'Average APM', hint: 'Mean actions per minute across the match' },
		{ key: 'apm_peak', label: 'Peak APM', hint: 'Highest single sample' }
	];

	function colorFor(slot: number): string {
		return PLAYER_COLORS[Math.min(Math.max(slot - 1, 0), PLAYER_COLORS.length - 1)];
	}
</script>

<svelte:head>
	<title>{match.player1} vs {match.player2} — Survive — AstroSwarm</title>
	<meta
		name="description"
		content={`Survive match report: ${match.player1} vs ${match.player2}, ${headline}.`}
	/>
</svelte:head>

<div class="page">
	<div class="shell shell-wide page-head">
		<a href="/gamemodes/survive" class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			All matches
		</a>
		<h1 class="page-title match-heading">{match.player1} vs {match.player2}</h1>
		<p class="match-id">{match.id}</p>
		<p class="page-lede">
			{dateLabel} · {clockLabel(match.duration)} played · {match.evaders_spawned} evaders launched ·
			{match.wild_remaining} purple ships never herded · {match.game_version}
		</p>
	</div>

	<div class="shell shell-wide report">
		<div class="headline">
			<div class="headline-value" class:tie={match.outcome === 'tie'}>{headline}</div>
			<div class="headline-label">Fewest evaders on the planet wins</div>
			<div class="headline-score">
				{match.player1}
				{match.evaders_player1} — {match.evaders_player2}
				{match.player2}
			</div>
		</div>

		<div class="player-grid">
			{#each players as player}
				<div class="card player">
					<h2 class="player-name">
						<span class="player-swatch" style={`background:${colorFor(player.slot)}`}></span>
						P{player.slot} · {player.name}
					</h2>
					<dl class="player-stats">
						{#each statRows as row}
							<dt title={row.hint}>{row.label}</dt>
							<dd>{player[row.key]}</dd>
						{/each}
					</dl>
				</div>
			{/each}
		</div>

		<h2 class="section-title chart-heading">Actions per minute</h2>
		{#if hasSeries}
			<ChartCard config={chartConfig} />

			<details class="card table-toggle">
				<summary>View as table</summary>
				<div class="table-scroll">
					<table class="data-table">
						<thead>
							<tr>
								<th>Match time</th>
								<th class="right">{match.player1} APM</th>
								<th class="right">{match.player2} APM</th>
							</tr>
						</thead>
						<tbody>
							{#each match.apm_times as time, index}
								<tr>
									<td>{clockLabel(time)}</td>
									<td class="right">{match.apm_series_player1[index] ?? 0}</td>
									<td class="right">{match.apm_series_player2[index] ?? 0}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</details>
		{:else}
			<div class="notice">This match is too short to chart.</div>
		{/if}

		<p class="footnote">
			An action is a movement input changing state — a key going down or up, or a controller stick
			crossing into a new direction — plus every power up activation. Samples are taken every
			{match.apm_bucket_seconds} seconds and scaled to a per-minute rate.
		</p>
	</div>
</div>

<style>
	.match-heading {
		margin-top: 1.25rem;
	}

	.match-id {
		margin-top: 0.6rem;
		font-family: ui-monospace, monospace;
		font-size: 0.72rem;
		color: var(--color-faint);
		overflow-wrap: anywhere;
	}

	.report {
		padding-bottom: 6rem;
	}

	.headline {
		margin-bottom: 2.5rem;
	}

	.headline-value {
		font-size: clamp(1.6rem, 4.5vw, 2.6rem);
		font-weight: 600;
		line-height: 1.1;
		color: var(--color-win);
	}

	.headline-value.tie {
		color: var(--color-warn);
	}

	.headline-label {
		margin-top: 0.5rem;
		font-size: 0.75rem;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--color-faint);
	}

	.headline-score {
		margin-top: 0.9rem;
		font-size: 0.95rem;
		color: var(--color-heading);
	}

	.player-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(19rem, 1fr));
		gap: 0.75rem;
	}

	.player {
		padding: 1.4rem;
	}

	.player-name {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.95rem;
		font-weight: 600;
	}

	.player-swatch {
		width: 0.75rem;
		height: 0.75rem;
		border-radius: 2px;
		flex-shrink: 0;
	}

	.player-stats {
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 0.5rem 1rem;
		margin-top: 1.1rem;
		font-size: 0.875rem;
	}

	.player-stats dt {
		color: var(--color-dim);
	}

	.player-stats dd {
		margin: 0;
		text-align: right;
		color: var(--color-heading);
		font-variant-numeric: tabular-nums;
	}

	.chart-heading {
		margin-top: 3rem;
		margin-bottom: 1rem;
	}

	.table-toggle {
		margin-top: 1rem;
		overflow: hidden;
	}

	.table-toggle summary {
		padding: 0.85rem 1.25rem;
		font-size: 0.85rem;
		font-weight: 500;
		color: var(--color-dim);
		cursor: pointer;
	}

	.table-toggle summary:hover {
		color: var(--color-heading);
	}

	.table-scroll {
		max-height: 24rem;
		overflow: auto;
	}

	.right {
		text-align: right;
	}

	.footnote {
		margin-top: 1.5rem;
		max-width: 48rem;
		font-size: 0.78rem;
		line-height: 1.6;
		color: var(--color-faint);
	}
</style>
