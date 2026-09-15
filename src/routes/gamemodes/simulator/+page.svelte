<script lang="ts">
	import Icon from '@iconify/svelte';
	import type { SimulatorEntryListItem } from '$lib/ts/simulator';
	import { durationLabel, shortDate } from '$lib/ts/simulator';

	interface PageData {
		entriesPromise: Promise<{ entries: SimulatorEntryListItem[]; apiError: boolean }>;
	}

	let { data }: { data: PageData } = $props();

	let entries = $state<SimulatorEntryListItem[]>([]);
	let apiError = $state(false);
	let loading = $state(true);

	$effect(() => {
		let active = true;
		loading = true;
		data.entriesPromise.then((result) => {
			if (!active) return;
			entries = result.entries;
			apiError = result.apiError;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	let searchQuery = $state('');
	let sortOrder = $state('date_desc');

	function matchesSearch(entry: SimulatorEntryListItem, query: string): boolean {
		return (
			entry.title.toLowerCase().includes(query) ||
			entry.username.toLowerCase().includes(query) ||
			entry.id.toLowerCase().includes(query) ||
			entry.species.some((species) => species.name.toLowerCase().includes(query))
		);
	}

	function compareEntries(a: SimulatorEntryListItem, b: SimulatorEntryListItem): number {
		if (sortOrder === 'date_asc') return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
		if (sortOrder === 'robots_desc') return b.peak_robot_count - a.peak_robot_count;
		if (sortOrder === 'duration_desc') return b.duration_seconds - a.duration_seconds;
		return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
	}

	let shown = $derived(
		entries
			.filter((entry) => searchQuery.trim() === '' || matchesSearch(entry, searchQuery.trim().toLowerCase()))
			.sort(compareEntries)
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
</script>

<svelte:head>
	<title>Simulator — AstroSwarm</title>
	<meta
		name="description"
		content="Swarm simulator runs shared by players, with replays and the logic of every species."
	/>
</svelte:head>

<div class="page">
	<div class="shell shell-wide page-head">
		<a href="/gamemodes" class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			All game modes
		</a>
		<h1 class="page-title mode-heading">Simulator</h1>
		<p class="page-lede">
			The sandbox, separate from the game. Players design species, program them with blocks, drop
			them in an arena with walls, obstacles and spawn zones, and record what happens. Every entry
			here carries the full recording and the logic of each species.
		</p>
	</div>

	<div class="shell shell-wide layout">
		<aside class="filters">
			<div>
				<label class="field-label" for="search">Search</label>
				<input
					id="search"
					type="text"
					placeholder="Title, player, species or ID"
					bind:value={searchQuery}
					class="field"
				/>
			</div>

			<div>
				<label class="field-label" for="sort">Sort by</label>
				<select id="sort" bind:value={sortOrder} class="field">
					<option value="date_desc">Date (newest)</option>
					<option value="date_asc">Date (oldest)</option>
					<option value="robots_desc">Most robots</option>
					<option value="duration_desc">Longest run</option>
				</select>
			</div>

			<p class="filters-note">
				Upload a recorded run from the simulator's Manage Setups screen to share it here.
			</p>
		</aside>

		<div class="results">
			{#if loading}
				<div class="notice">Loading simulator entries...</div>
			{:else if apiError}
				<div class="notice notice-error">Communication error. Unable to load simulator entries.</div>
			{:else if shown.length === 0}
				<div class="notice">
					{entries.length === 0
						? 'No simulator entries yet. Record a run in the simulator and upload it to appear here.'
						: 'No entries match this search.'}
				</div>
			{:else}
				<div class="entry-grid">
					{#each paged as entry}
						<a href={`/simulator/${entry.id}`} class="card-link entry">
							<div class="entry-head">
								<span class="entry-title">{entry.title}</span>
								<span class="badge">{durationLabel(entry.duration_seconds)}</span>
							</div>
							<div class="entry-author">by {entry.username}</div>

							{#if entry.description}
								<p class="entry-description">{entry.description}</p>
							{/if}

							<div class="species-row">
								{#each entry.species as species}
									<span class="species-chip">
										<span class="swatch" style={`background:${species.color}`}></span>
										{species.name}
									</span>
								{/each}
							</div>

							<div class="entry-foot">
								<span>
									{entry.peak_robot_count} robots peak
									{#if entry.spawn_zone_count > 0}
										· {entry.spawn_zone_count} spawn {entry.spawn_zone_count === 1 ? 'zone' : 'zones'}
									{/if}
								</span>
								<span>{shortDate(entry.created_at)}</span>
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

	.filters-note {
		font-size: 0.78rem;
		line-height: 1.55;
		color: var(--color-faint);
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
		display: flex;
		flex-direction: column;
		padding: 1.25rem;
	}

	.entry-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
	}

	.entry-title {
		font-size: 1rem;
		font-weight: 600;
		color: var(--color-heading);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.entry-author {
		margin-top: 0.3rem;
		font-size: 0.8rem;
		color: var(--color-dim);
	}

	.entry-description {
		margin-top: 0.6rem;
		font-size: 0.82rem;
		line-height: 1.5;
		color: var(--color-dim);
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.species-row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem 0.8rem;
		margin-top: 0.8rem;
		font-size: 0.75rem;
		color: var(--color-dim);
	}

	.species-chip {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
	}

	.swatch {
		width: 0.65rem;
		height: 0.65rem;
		border-radius: 50%;
	}

	.entry-foot {
		display: flex;
		justify-content: space-between;
		gap: 0.75rem;
		margin-top: auto;
		padding-top: 0.9rem;
		border-top: 1px solid var(--color-line);
		font-size: 0.75rem;
		color: var(--color-faint);
	}

	.species-row + .entry-foot {
		margin-top: 0.9rem;
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
