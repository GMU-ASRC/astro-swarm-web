<script lang="ts">
	import Icon from '@iconify/svelte';

	interface PlayerRow {
		player_id: string;
		username: string;
		total_xp: number;
		entries: number;
		overall_success: number | null;
		rank: number;
	}

	interface PageData {
		players: PlayerRow[];
		apiError: boolean;
	}

	let { data }: { data: PageData } = $props();

	let search = $state('');
	let shown = $derived(
		(data.players ?? []).filter(
			(player) =>
				search.trim() === '' || (player.username ?? '').toLowerCase().includes(search.toLowerCase())
		)
	);

	function rankClass(rank: number): string {
		if (rank <= 3) return 'badge-gold';
		return '';
	}
</script>

<svelte:head>
	<title>XP Leaderboard — AstroSwarm</title>
	<meta name="description" content="Top commanders ranked by XP earned across every AstroSwarm level." />
</svelte:head>

<div class="page">
	<div class="shell page-head">
		<p class="eyebrow">STANDINGS</p>
		<h1 class="page-title">XP Leaderboard</h1>
		<p class="page-lede">
			Commanders ranked by total XP earned across every level. Open a commander to see their profile,
			entries, and per-level ranks.
		</p>
	</div>

	<div class="shell board">
		<input type="text" placeholder="Search commander" bind:value={search} class="field search" />

		{#if data.apiError}
			<div class="notice notice-error">Communication error. Unable to load leaderboard data.</div>
		{:else if shown.length === 0}
			<div class="notice">
				No commanders yet. Complete a level in game and claim your XP to appear here.
			</div>
		{:else}
			<div class="rows">
				{#each shown as player}
					<a href={`/leaderboard/${player.player_id}`} class="card-link row">
						<span class="badge rank {rankClass(player.rank)}">#{player.rank}</span>
						<span class="name">{player.username}</span>
						<span class="numbers">
							<span class="xp">{player.total_xp} XP</span>
							<span class="detail">
								{player.overall_success != null ? `${player.overall_success}% avg` : 'no data'} ·
								{player.entries} entries
							</span>
						</span>
						<Icon icon="ph:caret-right-bold" width="14" color="var(--color-faint)" />
					</a>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.board {
		padding-bottom: 6rem;
	}

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

	.xp {
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
