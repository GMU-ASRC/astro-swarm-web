<script lang="ts">
	import Icon from '@iconify/svelte';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import {
		circlinessText,
		fetchTrainingScore,
		type TrainingScoreDetail,
		type TrainingScoreRow
	} from '$lib/ts/training';

	let { rows }: { rows: TrainingScoreRow[] } = $props();

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

	function dateText(value: string | null): string {
		return value ? new Date(value).toLocaleDateString() : 'unknown date';
	}

	let openPlayerId = $state<string | null>(null);
	let details = $state<Record<string, TrainingScoreDetail | null>>({});
	let loadingPlayerId = $state<string | null>(null);

	async function toggleRow(playerId: string) {
		if (openPlayerId === playerId) {
			openPlayerId = null;
			return;
		}
		openPlayerId = playerId;
		if (playerId in details) return;
		loadingPlayerId = playerId;
		details[playerId] = await fetchTrainingScore(playerId);
		if (loadingPlayerId === playerId) loadingPlayerId = null;
	}
</script>

<input type="text" placeholder="Search commander" bind:value={search} class="field search" />

{#if shown.length === 0}
	<div class="notice">
		{rows.length === 0
			? 'No training scores yet. Run the sandbox in game for 10 seconds to appear here.'
			: 'No commanders match that search.'}
	</div>
{:else}
	<div class="rows">
		{#each shown as row}
			{@const isOpen = openPlayerId === row.player_id}
			<div class="entry">
				<button type="button" class="card-link row" onclick={() => toggleRow(row.player_id)}>
					<span class="badge rank {rankClass(row.rank)}">#{row.rank}</span>
					<span class="name">{row.username}</span>
					<span class="numbers">
						<span class="rate">{circlinessText(row.circliness)}</span>
						<span class="detail">
							best circliness · {row.defenders} defenders · {dateText(row.updated_at)}
						</span>
					</span>
					<Icon
						icon={isOpen ? 'ph:caret-down-bold' : 'ph:caret-right-bold'}
						width="14"
						color="var(--color-faint)"
					/>
				</button>
				{#if isOpen}
					<div class="card algorithm-panel">
						<div class="algorithm-head">
							<span class="algorithm-title">Defender algorithm</span>
							<a href={`/leaderboard/${row.player_id}`} class="profile-link">
								View commander
								<Icon icon="ph:arrow-right-bold" width="12" />
							</a>
						</div>
						{#if loadingPlayerId === row.player_id}
							<div class="notice">Loading algorithm...</div>
						{:else if details[row.player_id]}
							<AlgorithmView scripts={details[row.player_id]?.algorithm ?? []} />
						{:else}
							<div class="notice notice-error">Unable to load this algorithm.</div>
						{/if}
					</div>
				{/if}
			</div>
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

	.entry {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.row {
		display: flex;
		align-items: center;
		gap: 1.25rem;
		width: 100%;
		padding: 1rem 1.25rem;
		font: inherit;
		text-align: left;
		cursor: pointer;
	}

	.algorithm-panel {
		padding: 1rem 1.25rem;
	}

	.algorithm-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 0.75rem;
	}

	.algorithm-title {
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--color-heading);
	}

	.profile-link {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.75rem;
		color: var(--color-brand-hover);
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
