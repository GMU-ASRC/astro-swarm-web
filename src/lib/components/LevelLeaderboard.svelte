<script lang="ts">
	import Icon from '@iconify/svelte';
	import type { LevelLeaderboardRow } from '$lib/ts/evaluation';

	let {
		rows,
		rateLabel
	}: { rows: LevelLeaderboardRow[]; rateLabel: string } = $props();

	let search = $state('');
	let shown = $derived(
		rows.filter(
			(row) =>
				search.trim() === '' || (row.username ?? '').toLowerCase().includes(search.toLowerCase())
		)
	);

	function rankClass(rank: number): string {
		return rank <= 3 ? 'badge-gold' : '';
	}
</script>

<input type="text" placeholder="Search commander" bind:value={search} class="field search" />

{#if shown.length === 0}
	<div class="notice">
		{rows.length === 0
			? 'No finished entries on this level yet.'
			: 'No commanders match that search.'}
	</div>
{:else}
	<div class="rows">
		{#each shown as row}
			<a href={`/levels/${row.best_entry_id}`} class="card-link row">
				<span class="badge rank {rankClass(row.rank)}">#{row.rank}</span>
				<span class="name">{row.username}</span>
				<span class="numbers">
					<span class="rate">{row.best_rate}%</span>
					<span class="detail">
						best {rateLabel} · {row.average_rate}% average · {row.entries}
						{row.entries === 1 ? 'entry' : 'entries'} · {row.best_defenders} defenders
					</span>
				</span>
				<Icon icon="ph:caret-right-bold" width="14" color="var(--color-faint)" />
			</a>
		{/each}
	</div>
{/if}

<style>
	.search {
		max-width: 24rem;
		margin-bottom: 1.5rem;
	}

	.rows {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.row {
		display: flex;
		align-items: center;
		gap: 1.25rem;
		padding: 1rem 1.25rem;
	}

	.rank {
		min-width: 3rem;
		justify-content: center;
		font-size: 0.85rem;
		font-weight: 600;
	}

	.name {
		flex: 1;
		min-width: 0;
		font-size: 1.05rem;
		font-weight: 600;
		color: var(--color-heading);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.numbers {
		text-align: right;
	}

	.rate {
		display: block;
		font-size: 1.05rem;
		font-weight: 600;
		color: var(--color-brand-hover);
	}

	.detail {
		display: block;
		margin-top: 0.15rem;
		font-size: 0.75rem;
		color: var(--color-faint);
	}
</style>
