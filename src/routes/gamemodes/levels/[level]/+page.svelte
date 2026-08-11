<script lang="ts">
	import Icon from '@iconify/svelte';
	import Spaceship from '$lib/components/Spaceship.svelte';
	import type { PlayerListItem } from '$lib/ts/evaluation';
	import { canonicalLevelId, type LevelInfo } from '$lib/ts/levels';

	interface PageData {
		level: LevelInfo;
		playersPromise: Promise<{ players: PlayerListItem[]; apiError: boolean }>;
	}

	let { data }: { data: PageData } = $props();

	let level = $derived(data.level);

	let players = $state<PlayerListItem[]>([]);
	let apiError = $state(false);
	let loading = $state(true);

	$effect(() => {
		let active = true;
		loading = true;
		data.playersPromise.then((result) => {
			if (!active) return;
			players = result.players;
			apiError = result.apiError;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	let searchQuery = $state('');
	let sortOrder = $state('date_desc');
	let minRate = $state(0);
	let startDate = $state('');
	let endDate = $state('');

	let entries = $derived(
		players.filter(
			(player) => canonicalLevelId(player.level_id) === level.id && player.status !== 'cancelled'
		)
	);

	let shown = $derived(
		entries
			.filter((player) => {
				if (searchQuery.trim() !== '') {
					const query = searchQuery.toLowerCase();
					if (
						!player.username.toLowerCase().includes(query) &&
						!player.id.toLowerCase().includes(query)
					) {
						return false;
					}
				}

				if (minRate > 0) {
					if (player.success_rate === null || player.success_rate === undefined) return false;
					if (player.success_rate < minRate) return false;
				}

				if (startDate) {
					const created = new Date(player.created_at);
					const from = new Date(startDate);
					from.setHours(0, 0, 0, 0);
					if (created.getTime() < from.getTime()) return false;
				}

				if (endDate) {
					const created = new Date(player.created_at);
					const until = new Date(endDate);
					until.setHours(23, 59, 59, 999);
					if (created.getTime() > until.getTime()) return false;
				}

				return true;
			})
			.sort((a, b) => {
				if (sortOrder === 'date_asc')
					return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
				if (sortOrder === 'rate_desc') return (b.success_rate ?? -1) - (a.success_rate ?? -1);
				if (sortOrder === 'rate_asc') return (a.success_rate ?? -1) - (b.success_rate ?? -1);
				return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
			})
	);

	const PAGE_SIZE = 12;
	let page = $state(1);
	let pageCount = $derived(Math.max(1, Math.ceil(shown.length / PAGE_SIZE)));
	let paged = $derived(shown.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE));

	$effect(() => {
		level.id;
		searchQuery;
		sortOrder;
		minRate;
		startDate;
		endDate;
		page = 1;
	});

	$effect(() => {
		if (page > pageCount) page = pageCount;
	});

	function statusClass(status: string): string {
		if (status === 'done') return 'badge-win';
		if (status === 'running' || status === 'queued') return 'badge-brand';
		return 'badge-loss';
	}

	function when(iso: string): string {
		const date = new Date(iso);
		const month = String(date.getMonth() + 1).padStart(2, '0');
		const day = String(date.getDate()).padStart(2, '0');
		return `${month}/${day}/${date.getFullYear()}`;
	}
</script>

<svelte:head>
	<title>{level.name} · {level.subtitle} — AstroSwarm</title>
	<meta name="description" content={level.summary} />
</svelte:head>

<div class="page">
	<div class="shell shell-wide page-head">
		<a href="/gamemodes/levels" class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			Levels
		</a>

		<div class="head-row">
			<div>
				<h1 class="page-title level-heading">{level.name} · {level.subtitle}</h1>
				<p class="page-lede">{level.summary}</p>
			</div>
			<Spaceship variant={level.variant} size="80" />
		</div>
	</div>

	<div class="shell shell-wide layout">
		<aside class="filters">
			<div>
				<label class="field-label" for="search">Search</label>
				<input
					id="search"
					type="text"
					placeholder="Username or ID"
					bind:value={searchQuery}
					class="field"
				/>
			</div>

			<div>
				<span class="field-label">Min {level.rateLabel}</span>
				<div class="slider-row">
					<input type="range" min="0" max="100" step="1" bind:value={minRate} class="slider" />
					<span class="slider-value">{minRate}%</span>
				</div>
			</div>

			<div>
				<span class="field-label">Date range</span>
				<div class="date-inputs">
					<input type="date" bind:value={startDate} class="field" />
					<input type="date" bind:value={endDate} class="field" />
				</div>
			</div>

			<div>
				<label class="field-label" for="sort">Sort by</label>
				<select id="sort" bind:value={sortOrder} class="field">
					<option value="date_desc">Date (newest)</option>
					<option value="date_asc">Date (oldest)</option>
					<option value="rate_desc">Success rate (high to low)</option>
					<option value="rate_asc">Success rate (low to high)</option>
				</select>
			</div>
		</aside>

		<div class="results">
			{#if loading}
				<div class="notice">Loading level data...</div>
			{:else if apiError}
				<div class="notice notice-error">Communication error. Unable to load level data.</div>
			{:else if shown.length === 0}
				<div class="notice">
					{entries.length === 0
						? 'No entries yet. Complete this level in game to appear here.'
						: 'No entries match these filters.'}
				</div>
			{:else}
				<div class="entry-grid">
					{#each paged as player}
						<a href={`/levels/${player.id}`} class="card-link entry">
							<div class="entry-head">
								<span class="entry-name">{player.username}</span>
								<span class="badge {statusClass(player.status)}">{player.status}</span>
							</div>

							<div class="entry-id">{player.id}</div>

							<div class="entry-result">
								{#if player.status === 'running' || player.status === 'queued'}
									{level.piloted ? 'Rendering' : 'Benchmarking'} · {Math.round(
										(player.progress ?? 0) * 100
									)}%
								{:else if level.piloted && level.number === 4}
									{player.success_rate ? 'Swarm delivered' : 'Not delivered'} · piloted run
								{:else if level.piloted}
									{player.success_rate ? 'Planet reached' : 'No goal'} · piloted run
								{:else if player.success_rate !== null && player.success_rate !== undefined}
									{player.success_rate}% {level.rateLabel} · {player.trials} trials
								{:else}
									{player.trials} trials
								{/if}
							</div>

							<div class="entry-foot">
								<span>{when(player.created_at)}</span>
								<span>{player.game_version ?? 'v0.0.4'}</span>
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
	.head-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 2rem;
	}

	.level-heading {
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

	.slider-row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.slider {
		flex: 1;
		accent-color: var(--color-brand);
		cursor: pointer;
	}

	.slider-value {
		width: 2.75rem;
		text-align: right;
		font-size: 0.85rem;
		color: var(--color-heading);
	}

	.date-inputs {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.results {
		flex: 1;
		min-width: 0;
	}

	.entry-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(19rem, 1fr));
		gap: 0.75rem;
	}

	.entry {
		padding: 1.25rem;
	}

	.entry-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
	}

	.entry-name {
		font-size: 1.05rem;
		font-weight: 600;
		color: var(--color-heading);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.entry-id {
		margin-top: 0.6rem;
		font-family: ui-monospace, monospace;
		font-size: 0.7rem;
		color: var(--color-faint);
		overflow-wrap: anywhere;
	}

	.entry-result {
		margin-top: 0.5rem;
		font-size: 0.85rem;
		color: var(--color-dim);
	}

	.entry-foot {
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

		.head-row :global(.spaceship) {
			display: none;
		}
	}
</style>
