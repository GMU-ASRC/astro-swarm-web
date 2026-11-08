<script lang="ts">
	import type { DailyCount } from '$lib/ts/adminStats';

	let { days }: { days: DailyCount[] } = $props();

	let peak = $derived(Math.max(1, ...days.map((day) => day.submitted)));
	let total = $derived(days.reduce((sum, day) => sum + day.submitted, 0));

	function dayLabel(date: string): string {
		return new Date(`${date}T00:00:00Z`).toLocaleDateString(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' });
	}
</script>

<div class="daily">
	<div class="daily-head">
		<span>{total.toLocaleString()} submissions in the last {days.length} days</span>
		<span class="daily-legend">
			<span class="swatch submitted"></span> submitted
			<span class="swatch done"></span> finished
		</span>
	</div>
	<div class="daily-bars">
		{#each days as day, index}
			<div class="daily-column" title={`${dayLabel(day.date)}: ${day.submitted} submitted, ${day.done} finished, ${day.players} players`}>
				<div class="daily-value">{day.submitted || ''}</div>
				<div class="daily-track">
					<div class="bar submitted" style:height={`${(100 * day.submitted) / peak}%`}>
						<div class="bar done" style:height={day.submitted ? `${(100 * day.done) / day.submitted}%` : '0'}></div>
					</div>
				</div>
				<div class="daily-label">{index % 2 === days.length % 2 ? '' : dayLabel(day.date)}</div>
			</div>
		{/each}
	</div>
</div>

<style>
	.daily-head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: 0.5rem;
		margin-bottom: 0.9rem;
		color: var(--color-dim);
		font-size: 0.8rem;
	}

	.daily-legend {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}

	.swatch {
		display: inline-block;
		width: 0.6rem;
		height: 0.6rem;
		margin-left: 0.6rem;
		border-radius: 2px;
	}

	.daily-bars {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: minmax(0, 1fr);
		gap: 0.4rem;
		height: 11rem;
	}

	.daily-column {
		display: flex;
		flex-direction: column;
		align-items: stretch;
		min-width: 0;
	}

	.daily-value {
		height: 1rem;
		color: var(--color-faint);
		font-size: 0.65rem;
		text-align: center;
		font-variant-numeric: tabular-nums;
	}

	.daily-track {
		display: flex;
		flex: 1;
		align-items: flex-end;
	}

	.bar {
		width: 100%;
		border-radius: 3px 3px 0 0;
	}

	.submitted {
		display: flex;
		align-items: flex-end;
		min-height: 2px;
		background: color-mix(in srgb, var(--color-brand) 45%, transparent);
	}

	.done {
		background: var(--color-win);
	}

	.daily-label {
		height: 1.2rem;
		margin-top: 0.35rem;
		overflow: hidden;
		color: var(--color-faint);
		font-size: 0.65rem;
		text-align: center;
		white-space: nowrap;
	}
</style>
