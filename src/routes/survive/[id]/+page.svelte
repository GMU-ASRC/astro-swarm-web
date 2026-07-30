<script lang="ts">
	import ChartCard from '$lib/components/ChartCard.svelte';
	import type { SurviveMatch, SurvivePlayer } from '$lib/ts/survive';
	import { PLAYER_COLORS, apmChartConfig, clockLabel } from '$lib/ts/survive';

	interface PageData {
		match: SurviveMatch;
	}

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let match: SurviveMatch = $state(data.match);

	let players = $derived(match.players ?? []);
	let hasSeries = $derived((match.apm_times?.length ?? 0) > 1);
	let chartConfig = $derived(apmChartConfig(match));
	let dateLabel = $derived(match.created_at.slice(0, 10));

	let headline = $derived(
		match.outcome === 'tie' ? 'TIE MATCH' : `${match.winner ?? 'WINNER'} WINS`
	);

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

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-320 mx-auto px-8 max-sm:px-5 py-12">
		<a href="/survive" class="font-game text-sm text-sky-300 hover:text-sky-200 tracking-wider"
			>&larr; ALL MATCHES</a
		>

		<h1
			class="font-game text-[clamp(1.6rem,3.6vw,2.6rem)] font-bold text-star-white leading-tight mt-4"
			style="text-shadow: 0 0 20px rgba(56,189,248,0.4)"
		>
			{match.player1} vs {match.player2}
		</h1>
		<p class="text-xs text-text-muted mt-2 font-mono break-all">{match.id}</p>
		<p class="text-sm text-text-muted mt-1">
			{dateLabel} · {clockLabel(match.duration)} played · {match.evaders_spawned} evaders launched · {match.wild_remaining}
			purple ships never herded · {match.game_version}
		</p>

		<div class="mt-8 mb-8 flex flex-wrap gap-6 items-end">
			<div>
				<div
					class="font-sim text-[clamp(1.8rem,5vw,3rem)] leading-none {match.outcome === 'tie'
						? 'text-amber-200'
						: 'text-green-300'}"
					style="text-shadow: 0 0 24px rgba(74,222,128,0.25)"
				>
					{headline}
				</div>
				<div class="text-xs text-text-muted mt-1 tracking-wider font-sim">
					FEWEST EVADERS ON THE PLANET WINS
				</div>
			</div>
			<div class="flex gap-4 text-sm">
				<span class="text-star-white"
					>{match.player1} {match.evaders_player1} — {match.evaders_player2} {match.player2}</span
				>
			</div>
		</div>

		<div class="mb-10 grid grid-cols-1 md:grid-cols-2 gap-6">
			{#each players as player}
				<div class="p-5 border-2 border-sky-500/20 bg-page-bg/60">
					<h2 class="font-sim text-sm mb-3 tracking-wider flex items-center gap-2">
						<span
							class="inline-block w-3 h-3 shrink-0"
							style={`background:${colorFor(player.slot)}`}
						></span>
						<span class="text-star-white">P{player.slot} · {player.name}</span>
					</h2>
					<dl class="grid grid-cols-2 gap-y-2 text-sm">
						{#each statRows as row}
							<dt class="text-text-muted" title={row.hint}>{row.label}</dt>
							<dd class="text-star-white text-right font-sim">{player[row.key]}</dd>
						{/each}
					</dl>
				</div>
			{/each}
		</div>

		<h2 class="font-game text-sky-400 text-sm tracking-widest mb-3">ACTIONS PER MINUTE</h2>
		{#if hasSeries}
			<ChartCard config={chartConfig} />

			<details class="mt-4 border-2 border-sky-500/20 bg-sky-500/5">
				<summary class="px-4 py-3 cursor-pointer font-game text-xs text-sky-200 tracking-wider">
					VIEW AS TABLE
				</summary>
				<div class="max-h-96 overflow-auto">
					<table class="w-full text-sm">
						<thead class="sticky top-0 bg-page-bg">
							<tr class="text-text-muted text-left">
								<th class="px-4 py-2 font-sim font-normal">Match time</th>
								<th class="px-4 py-2 font-sim font-normal text-right">{match.player1} APM</th>
								<th class="px-4 py-2 font-sim font-normal text-right">{match.player2} APM</th>
							</tr>
						</thead>
						<tbody>
							{#each match.apm_times as time, index}
								<tr class="border-t border-sky-500/10">
									<td class="px-4 py-1.5 text-text-muted">{clockLabel(time)}</td>
									<td class="px-4 py-1.5 text-star-white text-right font-sim"
										>{match.apm_series_player1[index] ?? 0}</td
									>
									<td class="px-4 py-1.5 text-star-white text-right font-sim"
										>{match.apm_series_player2[index] ?? 0}</td
									>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</details>
		{:else}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				This match is too short to chart.
			</div>
		{/if}

		<p class="text-[11px] text-text-muted mt-4 max-w-200">
			An action is a movement input changing state — a key going down or up, or a controller stick
			crossing into a new direction — plus every power up activation. Samples are taken every {match.apm_bucket_seconds}
			seconds and scaled to a per-minute rate.
		</p>
	</div>
</div>
