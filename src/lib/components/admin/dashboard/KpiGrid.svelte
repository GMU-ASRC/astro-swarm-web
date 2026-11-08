<script lang="ts">
	import type { AdminStats } from '$lib/ts/adminStats';
	import { formatBytes, formatCount, formatPercent, formatSeconds } from '$lib/ts/adminFormat';

	let { stats, totalBytes = null }: { stats: AdminStats; totalBytes?: number | null } = $props();

	let entries = $derived(stats.entries);
	let queue = $derived((entries.by_status.queued ?? 0) + (entries.by_status.running ?? 0));
	let cards = $derived([
		{ label: 'Entries', value: formatCount(entries.total), hint: `+${formatCount(entries.last_24h)} today · +${formatCount(entries.last_7d)} this week` },
		{ label: 'Players', value: formatCount(entries.players), hint: `${formatCount(entries.active_players)} active this week` },
		{ label: 'Job queue', value: formatCount(queue), hint: `${entries.by_status.running ?? 0} running · ${entries.by_status.queued ?? 0} waiting`, tone: queue > 0 ? 'brand' : '' },
		{ label: 'Workers online', value: `${stats.workers.online} / ${stats.workers.total}`, hint: `${stats.workers.busy} busy · ${stats.workers.enabled} enabled`, tone: stats.workers.online === 0 ? 'loss' : 'win' },
		{ label: 'Average success', value: formatPercent(entries.average_success), hint: 'across finished entries' },
		{ label: 'Turnaround', value: formatSeconds(entries.turnaround_s_7d), hint: 'submit to result, last 7 days' },
		{ label: 'Shared results', value: formatCount(entries.shared_results), hint: `${formatCount(entries.benchmarks)} unique benchmarks` },
		{ label: 'Total stored', value: formatBytes(totalBytes), hint: `${formatCount(entries.benchmark_runs)} recorded runs` }
	]);
</script>

<div class="kpi-grid">
	{#each cards as card}
		<div class="kpi" data-tone={card.tone ?? ''}>
			<div class="kpi-label">{card.label}</div>
			<div class="kpi-value">{card.value}</div>
			<div class="kpi-hint">{card.hint}</div>
		</div>
	{/each}
</div>

<style>
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
		gap: 0.75rem;
	}

	.kpi {
		padding: 1rem 1.1rem;
		border: 1px solid var(--color-line);
		border-top: 2px solid var(--color-line-strong);
		background: var(--color-surface);
	}

	.kpi[data-tone='brand'] {
		border-top-color: var(--color-brand);
	}

	.kpi[data-tone='win'] {
		border-top-color: var(--color-win);
	}

	.kpi[data-tone='loss'] {
		border-top-color: var(--color-loss);
	}

	.kpi-label {
		color: var(--color-faint);
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}

	.kpi-value {
		margin-top: 0.45rem;
		color: var(--color-heading);
		font-size: 1.55rem;
		font-weight: 600;
		line-height: 1.1;
		font-variant-numeric: tabular-nums;
	}

	.kpi-hint {
		margin-top: 0.4rem;
		color: var(--color-dim);
		font-size: 0.75rem;
	}
</style>
