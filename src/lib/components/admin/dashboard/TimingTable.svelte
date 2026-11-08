<script lang="ts">
	import type { TimingSummary } from '$lib/ts/adminStats';
	import { formatAgo, formatCount, formatMs } from '$lib/ts/adminFormat';

	let {
		rows,
		limit = 12,
		label,
		empty,
		slowMs = 1000,
		warnMs = 300
	}: { rows: TimingSummary[]; limit?: number; label: string; empty: string; slowMs?: number; warnMs?: number } = $props();

	let showAll = $state(false);
	let shown = $derived(showAll ? rows : rows.slice(0, limit));

	function tone(ms: number): string {
		if (ms >= slowMs) return 'slow';
		if (ms >= warnMs) return 'warn';
		return 'fast';
	}
</script>

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>{label}</th>
				<th>Count</th>
				<th>Average</th>
				<th>Median</th>
				<th>p95</th>
				<th>Slowest</th>
				<th>Last</th>
			</tr>
		</thead>
		<tbody>
			{#each shown as row}
				<tr>
					<td><code>{row.key}</code></td>
					<td>{formatCount(row.count)}</td>
					<td class={tone(row.average_ms)}>{formatMs(row.average_ms)}</td>
					<td class={tone(row.p50_ms)}>{formatMs(row.p50_ms)}</td>
					<td class={tone(row.p95_ms)}>{formatMs(row.p95_ms)}</td>
					<td class={tone(row.max_ms)}>{formatMs(row.max_ms)}</td>
					<td>{formatAgo(row.last_at)}</td>
				</tr>
			{:else}
				<tr><td colspan="7">{empty}</td></tr>
			{/each}
		</tbody>
	</table>
</div>
{#if rows.length > limit}
	<button class="admin-btn more" onclick={() => (showAll = !showAll)}>
		{showAll ? 'Show fewer' : `Show all ${rows.length}`}
	</button>
{/if}

<style>
	td {
		font-variant-numeric: tabular-nums;
	}

	.fast {
		color: var(--color-win);
	}

	.warn {
		color: var(--color-warn);
	}

	.slow {
		color: var(--color-loss);
	}

	.more {
		margin-top: 0.6rem;
	}
</style>
