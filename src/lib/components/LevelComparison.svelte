<script lang="ts">
	import ChartCard from '$lib/components/ChartCard.svelte';
	import type { LevelSweepEntry } from '$lib/ts/evaluation';
	import { BEST_SERIES_ID, comparisonConfig } from '$lib/ts/levelCharts';

	let { entries, attrition }: { entries: LevelSweepEntry[]; attrition: boolean } = $props();

	// The best rate reached at each ring size across every entry, so a reader can
	// see the ceiling the level has been pushed to as well as who got there.
	let best = $derived.by(() => {
		const byN = new Map<number, number>();
		for (const entry of entries) {
			for (const point of entry.sweep) {
				const current = byN.get(point.n);
				if (current == null || point.capture_rate > current) byN.set(point.n, point.capture_rate);
			}
		}
		return [...byN.entries()]
			.sort((a, b) => a[0] - b[0])
			.map(([n, rate]) => ({ n, capture_rate: rate, risk: Math.round((100 - rate) * 10) / 10 }));
	});

	let series = $derived(
		best.length > 0
			? [
					{
						id: BEST_SERIES_ID,
						username: 'Best of every entry',
						success_rate: best[0].capture_rate,
						sweep: best
					},
					...entries
				]
			: entries
	);
</script>

{#if entries.length === 0}
	<div class="notice">
		No finished entries with a defender sweep yet. Submit this level in game to draw the first
		curve.
	</div>
{:else}
	<div class="chart-grid">
		<ChartCard
			config={comparisonConfig(
				series,
				'capture_rate',
				'Capture Success Rate vs Number of Defenders',
				'Capture success rate (%)'
			)}
		/>
		<ChartCard
			config={comparisonConfig(
				series,
				'risk',
				'Risk vs Number of Defenders',
				'Risk = 1 - capture success rate (%)'
			)}
		/>
	</div>
	<p class="note">
		Every line is one submitted entry, drawn from its own defender sweep: n ships ringed around the
		planet, each dropped at a random angle inside its own slice. The heavy dashed line is the best rate
		any entry reached at each n.
		{#if attrition}
			A capture on this level destroys the defender that made it, so an entry's own page also charts
			the risk it carried as its line thinned.
		{/if}
	</p>
{/if}

<style>
	.chart-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(24rem, 1fr));
		gap: 1rem;
	}

	.note {
		margin-top: 1.25rem;
		font-size: 0.8rem;
		color: var(--color-faint);
	}
</style>
