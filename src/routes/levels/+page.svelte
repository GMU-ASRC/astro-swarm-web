<script lang="ts">
	import type { PlayerListItem } from '$lib/ts/evaluation';

	interface PageData {
		playersPromise: Promise<{ players: PlayerListItem[]; apiError: boolean }>;
	}

	let { data }: { data: PageData } = $props();

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

	const levels = [{ id: 'farp', label: 'FARP', enabled: true }];
	let selectedLevel = $state('farp');
	let searchQuery = $state('');
	let sortOrder = $state('date_desc');
	let minRate = $state(0);
	let startDate = $state('');
	let endDate = $state('');

	let shown = $derived(players.filter((p) => {
		if ((p.level_id ?? 'farp') !== selectedLevel) return false;
		
		if (searchQuery.trim() !== '') {
			const q = searchQuery.toLowerCase();
			if (!p.username.toLowerCase().includes(q) && !p.id.toLowerCase().includes(q)) return false;
		}

		if (minRate > 0) {
			if (p.success_rate === null || p.success_rate === undefined || p.success_rate < minRate) return false;
		}

		if (startDate) {
			const d = new Date(p.created_at);
			const s = new Date(startDate);
			s.setHours(0, 0, 0, 0);
			if (d.getTime() < s.getTime()) return false;
		}

		if (endDate) {
			const d = new Date(p.created_at);
			const e = new Date(endDate);
			e.setHours(23, 59, 59, 999);
			if (d.getTime() > e.getTime()) return false;
		}

		return true;
	}).sort((a, b) => {
		if (sortOrder === 'date_desc') return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
		if (sortOrder === 'date_asc') return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
		if (sortOrder === 'rate_desc') return (b.success_rate ?? -1) - (a.success_rate ?? -1);
		if (sortOrder === 'rate_asc') return (a.success_rate ?? -1) - (b.success_rate ?? -1);
		return 0;
	}));

	const PAGE_SIZE = 12;
	let page = $state(1);
	let pageCount = $derived(Math.max(1, Math.ceil(shown.length / PAGE_SIZE)));
	let paged = $derived(shown.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE));

	$effect(() => {
		selectedLevel;
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

	function statusColor(status: string): string {
		if (status === 'done') return 'text-green-300 border-green-400/40 bg-green-400/10';
		if (status === 'running' || status === 'queued') return 'text-sky-300 border-sky-400/40 bg-sky-400/10';
		return 'text-red-300 border-red-400/40 bg-red-400/10';
	}

	function when(iso: string): string {
		const d = new Date(iso);
		const mm = String(d.getMonth() + 1).padStart(2, '0');
		const dd = String(d.getDate()).padStart(2, '0');
		const yyyy = d.getFullYear();
		return `${mm}/${dd}/${yyyy}`;
	}
</script>

<svelte:head>
	<title>Levels — AstroSwarm</title>
	<meta name="description" content="Per-level benchmark data for player algorithms in AstroSwarm." />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-6">
		<h1 class="font-game text-[clamp(1.8rem,4vw,3rem)] font-bold text-star-white leading-tight mb-3" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			Levels
		</h1>
		<p class="font-sim text-base text-text-muted leading-relaxed max-w-150">
			Pick a level to see how each commander's algorithm performs. Every completed run is benchmarked headlessly over 100 trials.
		</p>
	</div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pb-6 flex flex-wrap gap-2">
		{#each levels as level}
			<button
				type="button"
				disabled={!level.enabled}
				onclick={() => (selectedLevel = level.id)}
				class="px-4 py-2 border-2 font-game text-sm tracking-wider transition-colors disabled:opacity-40 disabled:cursor-not-allowed {selectedLevel === level.id
					? 'border-sky-400 bg-sky-500/15 text-sky-200'
					: 'border-sky-500/20 text-text-muted hover:border-sky-400/50'}"
			>
				{level.label}
			</button>
		{/each}
	</div>



	<div class="max-w-225 mx-auto px-8 max-sm:px-5 py-12 max-md:py-8 flex flex-col md:flex-row gap-8 items-start">
		<aside class="w-full md:w-64 shrink-0 flex flex-col gap-8">
			<div>
				<h3 class="font-game text-sky-400 text-sm tracking-widest mb-3">SEARCH</h3>
				<input
					type="text"
					placeholder="Username or ID..."
					bind:value={searchQuery}
					class="w-full px-4 py-2 bg-sky-500/5 border-2 border-sky-500/20 text-star-white font-game text-sm placeholder:text-text-muted focus:outline-none focus:border-sky-400/50 transition-colors"
				/>
			</div>

			<div>
				<h3 class="font-game text-sky-400 text-sm tracking-widest mb-3">MIN DETECTION RATE</h3>
				<div class="flex items-center gap-3">
					<input
						type="range"
						min="0"
						max="100"
						step="1"
						bind:value={minRate}
						class="flex-1 accent-sky-400 cursor-pointer"
					/>
					<span class="font-game text-sky-200 text-sm w-10 text-right">{minRate}%</span>
				</div>
			</div>

			<div>
				<h3 class="font-game text-sky-400 text-sm tracking-widest mb-3">DATE RANGE</h3>
				<div class="flex flex-col gap-2">
					<input
						type="date"
						bind:value={startDate}
						class="w-full px-4 py-2 bg-sky-500/5 border-2 border-sky-500/20 text-star-white font-game text-sm focus:outline-none focus:border-sky-400/50 transition-colors"
					/>
					<input
						type="date"
						bind:value={endDate}
						class="w-full px-4 py-2 bg-sky-500/5 border-2 border-sky-500/20 text-star-white font-game text-sm focus:outline-none focus:border-sky-400/50 transition-colors"
					/>
				</div>
			</div>

			<div>
				<h3 class="font-game text-sky-400 text-sm tracking-widest mb-3">SORT BY</h3>
				<select
					bind:value={sortOrder}
					class="w-full px-4 py-2 bg-sky-500/5 border-2 border-sky-500/20 text-star-white font-game text-sm focus:outline-none focus:border-sky-400/50 transition-colors appearance-none cursor-pointer"
				>
					<option value="date_desc">Date (Newest)</option>
					<option value="date_asc">Date (Oldest)</option>
					<option value="rate_desc">Detection Rate (High to Low)</option>
					<option value="rate_asc">Detection Rate (Low to High)</option>
				</select>
			</div>
		</aside>

		<div class="flex-1 min-w-0">
		{#if loading}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-sky-200 font-game tracking-wider text-center animate-pulse">
				LOADING LEVEL DATA...
			</div>
		{:else if apiError}
			<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center">
				COMMUNICATION ERROR. UNABLE TO LOAD LEVEL DATA.
			</div>
		{:else if shown.length === 0}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No evaluations yet. Complete the FARP scenario in-game to appear here.
			</div>
		{:else}
			<div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
				{#each paged as player}
					<a
						href={`/levels/${player.id}`}
						class="block p-5 border-2 border-sky-500/20 bg-sky-500/5 hover:border-sky-400/50 hover:bg-sky-500/10 transition-colors"
					>
						<div class="flex items-center justify-between gap-3">
							<span class="font-game text-lg text-star-white tracking-wide">{player.username}</span>
							<span class="text-[10px] font-game tracking-wider px-2 py-1 border {statusColor(player.status)}">
								{player.status.toUpperCase()}
							</span>
						</div>
						<div class="mt-3 text-xs text-text-muted">
							<div class="mb-1 text-[10px] text-text-muted/60 font-mono break-all">{player.id}</div>
							{#if player.status === 'running' || player.status === 'queued'}
								Benchmarking · {Math.round((player.progress ?? 0) * 100)}%
							{:else if player.success_rate !== null && player.success_rate !== undefined}
								FARP · {player.success_rate}% detection rate · {player.trials} trials
							{:else}
								FARP · {player.trials} trials
							{/if}
						</div>
						<div class="mt-1 text-[11px] text-text-muted/70">{when(player.created_at)}</div>
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
					<span class="text-xs text-text-muted font-game tracking-wider">PAGE {page} / {pageCount}</span>
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
