<script lang="ts">
	import Icon from '@iconify/svelte';
	import type { SurviveMatchListItem } from '$lib/ts/survive';
	import { clockLabel } from '$lib/ts/survive';

	interface PageData {
		matchesPromise: Promise<{ matches: SurviveMatchListItem[]; apiError: boolean }>;
	}

	let { data }: { data: PageData } = $props();

	let matches = $state<SurviveMatchListItem[]>([]);
	let apiError = $state(false);
	let loading = $state(true);

	$effect(() => {
		let active = true;
		loading = true;
		data.matchesPromise.then((result) => {
			if (!active) return;
			matches = result.matches;
			apiError = result.apiError;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	let searchQuery = $state('');
	let sortOrder = $state('date_desc');

	let shown = $derived(
		matches
			.filter((match) => {
				if (searchQuery.trim() === '') return true;
				const query = searchQuery.toLowerCase();
				return (
					match.player1.toLowerCase().includes(query) ||
					match.player2.toLowerCase().includes(query) ||
					match.id.toLowerCase().includes(query)
				);
			})
			.sort((a, b) => {
				if (sortOrder === 'date_asc')
					return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
				if (sortOrder === 'apm_desc')
					return Math.max(b.apm_player1, b.apm_player2) - Math.max(a.apm_player1, a.apm_player2);
				return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
			})
	);

	const PAGE_SIZE = 12;
	let page = $state(1);
	let pageCount = $derived(Math.max(1, Math.ceil(shown.length / PAGE_SIZE)));
	let paged = $derived(shown.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE));

	$effect(() => {
		searchQuery;
		sortOrder;
		page = 1;
	});

	$effect(() => {
		if (page > pageCount) page = pageCount;
	});

	function when(iso: string): string {
		const date = new Date(iso);
		const month = String(date.getMonth() + 1).padStart(2, '0');
		const day = String(date.getDate()).padStart(2, '0');
		return `${month}/${day}/${date.getFullYear()}`;
	}

	function outcomeLabel(match: SurviveMatchListItem): string {
		if (match.outcome === 'tie') return 'Tie';
		return `${match.winner ?? 'Winner'} wins`;
	}
</script>

<svelte:head>
	<title>Survive — AstroSwarm</title>
	<meta
		name="description"
		content="Two-player Survive match reports with actions-per-minute telemetry."
	/>
</svelte:head>

<div class="page">
	<div class="shell shell-wide page-head">
		<a href="/gamemodes" class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			All game modes
		</a>
		<h1 class="page-title mode-heading">Survive</h1>
		<p class="page-lede">
			Two commanders, one swarm, three minutes. Each match herds twenty four wandering ships into
			planetary defenses while fourteen evaders come at the bases in waves, seven at each planet.
			Every match uploads its result and the actions-per-minute traces for both players.
		</p>
	</div>

	<div class="shell shell-wide layout">
		<aside class="filters">
			<div>
				<label class="field-label" for="search">Search</label>
				<input
					id="search"
					type="text"
					placeholder="Commander or match ID"
					bind:value={searchQuery}
					class="field"
				/>
			</div>

			<div>
				<label class="field-label" for="sort">Sort by</label>
				<select id="sort" bind:value={sortOrder} class="field">
					<option value="date_desc">Date (newest)</option>
					<option value="date_asc">Date (oldest)</option>
					<option value="apm_desc">Peak average APM</option>
				</select>
			</div>
		</aside>

		<div class="results">
			{#if loading}
				<div class="notice">Loading match data...</div>
			{:else if apiError}
				<div class="notice notice-error">Communication error. Unable to load match data.</div>
			{:else if shown.length === 0}
				<div class="notice">
					{matches.length === 0
						? 'No matches yet. Play the Survive game mode in game to appear here.'
						: 'No matches match this search.'}
				</div>
			{:else}
				<div class="match-grid">
					{#each paged as match}
						<a href={`/survive/${match.id}`} class="card-link match">
							<div class="match-head">
								<span class="match-name">{match.player1} vs {match.player2}</span>
								<span class="badge" class:badge-warn={match.outcome === 'tie'} class:badge-win={match.outcome !== 'tie'}>
									{outcomeLabel(match)}
								</span>
							</div>

							<div class="match-id">{match.id}</div>

							<div class="match-result">
								Evaders through {match.evaders_player1} - {match.evaders_player2} · avg APM
								{Math.round(match.apm_player1)} / {Math.round(match.apm_player2)}
							</div>

							<div class="match-foot">
								<span>{when(match.created_at)} · {clockLabel(match.duration)}</span>
								<span>{match.game_version}</span>
							</div>
						</a>
					{/each}
				</div>

				{#if pageCount > 1}
					<div class="pager">
						<button
							type="button"
							class="btn btn-ghost btn-sm"
							disabled={page <= 1}
							onclick={() => (page = Math.max(1, page - 1))}
						>
							<Icon icon="ph:caret-left-bold" width="13" />
							Prev
						</button>
						<span class="pager-label">Page {page} of {pageCount}</span>
						<button
							type="button"
							class="btn btn-ghost btn-sm"
							disabled={page >= pageCount}
							onclick={() => (page = Math.min(pageCount, page + 1))}
						>
							Next
							<Icon icon="ph:caret-right-bold" width="13" />
						</button>
					</div>
				{/if}
			{/if}
		</div>
	</div>
</div>

<style>
	.mode-heading {
		margin-top: 1.25rem;
	}

	.layout {
		display: flex;
		align-items: flex-start;
		gap: 2.5rem;
		padding-bottom: 6rem;
	}

	.filters {
		display: flex;
		flex-direction: column;
		gap: 1.75rem;
		width: 16rem;
		flex-shrink: 0;
	}

	.results {
		flex: 1;
		min-width: 0;
	}

	.match-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(19rem, 1fr));
		gap: 0.75rem;
	}

	.match {
		padding: 1.25rem;
	}

	.match-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
	}

	.match-name {
		font-size: 1rem;
		font-weight: 600;
		color: var(--color-heading);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.match-id {
		margin-top: 0.6rem;
		font-family: ui-monospace, monospace;
		font-size: 0.7rem;
		color: var(--color-faint);
		overflow-wrap: anywhere;
	}

	.match-result {
		margin-top: 0.5rem;
		font-size: 0.85rem;
		color: var(--color-dim);
	}

	.match-foot {
		display: flex;
		justify-content: space-between;
		margin-top: 0.9rem;
		padding-top: 0.9rem;
		border-top: 1px solid var(--color-line);
		font-size: 0.75rem;
		color: var(--color-faint);
	}

	.pager {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		margin-top: 2rem;
	}

	.pager-label {
		font-size: 0.8rem;
		color: var(--color-faint);
	}

	@media (max-width: 900px) {
		.layout {
			flex-direction: column;
			gap: 2rem;
		}

		.filters {
			width: 100%;
		}
	}
</style>
