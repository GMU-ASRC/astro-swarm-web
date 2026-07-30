<script lang="ts">
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
			.filter((m) => {
				if (searchQuery.trim() === '') return true;
				const q = searchQuery.toLowerCase();
				return (
					m.player1.toLowerCase().includes(q) ||
					m.player2.toLowerCase().includes(q) ||
					m.id.toLowerCase().includes(q)
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
		const d = new Date(iso);
		const mm = String(d.getMonth() + 1).padStart(2, '0');
		const dd = String(d.getDate()).padStart(2, '0');
		return `${mm}/${dd}/${d.getFullYear()}`;
	}

	function outcomeLabel(m: SurviveMatchListItem): string {
		if (m.outcome === 'tie') return 'TIE';
		return `${m.winner ?? 'WINNER'} WINS`;
	}

	function outcomeColor(outcome: string): string {
		if (outcome === 'tie') return 'text-amber-200 border-amber-400/40 bg-amber-400/10';
		return 'text-green-300 border-green-400/40 bg-green-400/10';
	}
</script>

<svelte:head>
	<title>Survive Matches — AstroSwarm</title>
	<meta
		name="description"
		content="Two-player Survive match reports with actions-per-minute telemetry."
	/>
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-360 mx-auto px-8 max-sm:px-5 pt-12 pb-6">
		<h1
			class="font-game text-[clamp(1.8rem,4vw,3rem)] font-bold text-star-white leading-tight mb-3"
			style="text-shadow: 0 0 20px rgba(56,189,248,0.4)"
		>
			Survive
		</h1>
		<p class="font-sim text-base text-text-muted leading-relaxed max-w-150">
			Two commanders, one swarm, three minutes. Each match herds twenty four wandering ships into
			planetary defences while fourteen evaders come at the bases in waves, seven at each planet.
			Every match uploads its result and the actions-per-minute traces for both players.
		</p>
	</div>

	<div class="max-w-360 mx-auto px-8 max-sm:px-5 py-8 flex flex-col md:flex-row gap-8 items-start">
		<aside class="w-full md:w-64 shrink-0 flex flex-col gap-8">
			<div>
				<h3 class="font-game text-sky-400 text-sm tracking-widest mb-3">SEARCH</h3>
				<input
					type="text"
					placeholder="Commander or match ID..."
					bind:value={searchQuery}
					class="w-full px-4 py-2 bg-sky-500/5 border-2 border-sky-500/20 text-star-white font-game text-sm placeholder:text-text-muted focus:outline-none focus:border-sky-400/50 transition-colors"
				/>
			</div>

			<div>
				<h3 class="font-game text-sky-400 text-sm tracking-widest mb-3">SORT BY</h3>
				<select
					bind:value={sortOrder}
					class="w-full px-4 py-2 bg-sky-500/5 border-2 border-sky-500/20 text-star-white font-game text-sm focus:outline-none focus:border-sky-400/50 transition-colors appearance-none cursor-pointer"
				>
					<option value="date_desc">Date (Newest)</option>
					<option value="date_asc">Date (Oldest)</option>
					<option value="apm_desc">Peak average APM</option>
				</select>
			</div>
		</aside>

		<div class="flex-1 min-w-0 w-full">
			{#if loading}
				<div
					class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-sky-200 font-game tracking-wider text-center animate-pulse"
				>
					LOADING MATCH DATA...
				</div>
			{:else if apiError}
				<div
					class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center"
				>
					COMMUNICATION ERROR. UNABLE TO LOAD MATCH DATA.
				</div>
			{:else if shown.length === 0}
				<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
					No matches yet. Play the Survive game mode in-game to appear here.
				</div>
			{:else}
				<div class="grid grid-cols-1 md:grid-cols-2 2xl:grid-cols-3 gap-4">
					{#each paged as match}
						<a
							href={`/survive/${match.id}`}
							class="block p-5 border-2 border-sky-500/20 bg-sky-500/5 hover:border-sky-400/50 hover:bg-sky-500/10 transition-colors"
						>
							<div class="flex items-center justify-between gap-3">
								<span class="font-game text-lg text-star-white tracking-wide"
									>{match.player1} vs {match.player2}</span
								>
								<span
									class="text-[10px] font-game tracking-wider px-2 py-1 border shrink-0 {outcomeColor(
										match.outcome
									)}"
								>
									{outcomeLabel(match)}
								</span>
							</div>
							<div class="mt-3 text-xs text-text-muted">
								<div class="mb-1 text-[10px] text-text-muted/60 font-mono break-all">{match.id}</div>
								Evaders through: {match.evaders_player1} - {match.evaders_player2} · avg APM {Math.round(
									match.apm_player1
								)} / {Math.round(match.apm_player2)}
							</div>
							<div class="mt-1 flex items-center justify-between text-[11px] text-text-muted/70">
								<span>{when(match.created_at)} · {clockLabel(match.duration)}</span>
								<span class="font-mono">{match.game_version}</span>
							</div>
						</a>
					{/each}
				</div>

				{#if pageCount > 1}
					<div class="flex items-center justify-center gap-3 mt-8">
						<button
							type="button"
							disabled={page <= 1}
							onclick={() => (page = Math.max(1, page - 1))}
							class="px-4 py-2 border-2 border-sky-500/20 text-sky-200 text-sm font-game tracking-wider hover:border-sky-400/50 disabled:opacity-30 disabled:cursor-not-allowed"
						>
							PREV
						</button>
						<span class="text-xs text-text-muted font-game tracking-wider"
							>PAGE {page} / {pageCount}</span
						>
						<button
							type="button"
							disabled={page >= pageCount}
							onclick={() => (page = Math.min(pageCount, page + 1))}
							class="px-4 py-2 border-2 border-sky-500/20 text-sky-200 text-sm font-game tracking-wider hover:border-sky-400/50 disabled:opacity-30 disabled:cursor-not-allowed"
						>
							NEXT
						</button>
					</div>
				{/if}
			{/if}
		</div>
	</div>
</div>
