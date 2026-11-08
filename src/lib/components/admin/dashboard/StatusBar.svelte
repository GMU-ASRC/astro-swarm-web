<script lang="ts">
	import { STATUS_COLORS, STATUS_ORDER } from '$lib/ts/adminStats';
	import { formatCount } from '$lib/ts/adminFormat';

	let { counts }: { counts: Record<string, number> } = $props();

	let total = $derived(Object.values(counts).reduce((sum, count) => sum + count, 0));
	let segments = $derived(
		Object.entries(counts)
			.filter(([, count]) => count > 0)
			.sort(([left], [right]) => rank(left) - rank(right))
			.map(([status, count]) => ({ status, count, share: total ? (100 * count) / total : 0 }))
	);

	function rank(status: string): number {
		const index = STATUS_ORDER.indexOf(status);
		return index === -1 ? STATUS_ORDER.length : index;
	}
</script>

<div class="status">
	<div class="status-bar" role="img" aria-label="Entries by status">
		{#each segments as segment}
			<div
				class="status-segment"
				style:width={`${segment.share}%`}
				style:background={STATUS_COLORS[segment.status] ?? 'var(--color-warn)'}
				title={`${segment.status}: ${segment.count}`}
			></div>
		{/each}
	</div>
	<div class="status-legend">
		{#each segments as segment}
			<span class="legend-item">
				<span class="swatch" style:background={STATUS_COLORS[segment.status] ?? 'var(--color-warn)'}></span>
				{segment.status}
				<strong>{formatCount(segment.count)}</strong>
				<span class="share">{segment.share.toFixed(1)}%</span>
			</span>
		{/each}
	</div>
</div>

<style>
	.status-bar {
		display: flex;
		height: 0.8rem;
		overflow: hidden;
		border-radius: 999px;
		background: var(--color-surface-raised);
	}

	.status-segment {
		min-width: 3px;
	}

	.status-legend {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem 1.25rem;
		margin-top: 0.85rem;
		font-size: 0.8rem;
		color: var(--color-dim);
		text-transform: capitalize;
	}

	.legend-item {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}

	.legend-item strong {
		color: var(--color-heading);
		font-weight: 600;
	}

	.swatch {
		width: 0.6rem;
		height: 0.6rem;
		border-radius: 2px;
	}

	.share {
		color: var(--color-faint);
	}
</style>
