<script lang="ts">
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
			(p) => search.trim() === '' || (p.username ?? '').toLowerCase().includes(search.toLowerCase())
		)
	);

	function medal(rank: number): string {
		if (rank === 1) return 'text-amber-300 border-amber-400/50 bg-amber-400/10';
		if (rank === 2) return 'text-zinc-200 border-zinc-300/40 bg-zinc-300/10';
		if (rank === 3) return 'text-orange-300 border-orange-400/40 bg-orange-400/10';
		return 'text-sky-200 border-sky-400/30 bg-sky-400/5';
	}
</script>

<svelte:head>
	<title>XP Leaderboard — AstroSwarm</title>
	<meta name="description" content="Top commanders ranked by XP earned across every AstroSwarm level." />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-6">
		<h1 class="font-game text-[clamp(1.8rem,4vw,3rem)] font-bold text-star-white leading-tight mb-3" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			XP Leaderboard
		</h1>
		<p class="font-sim text-base text-text-muted leading-relaxed max-w-150">
			Commanders ranked by total XP earned across every level. Click a commander to see their profile,
			entries, and per-level ranks.
		</p>
	</div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pb-6">
		<input
			type="text"
			placeholder="Search commander..."
			bind:value={search}
			class="w-full max-w-100 px-4 py-2 bg-sky-500/5 border-2 border-sky-500/20 text-star-white font-game text-sm placeholder:text-text-muted focus:outline-none focus:border-sky-400/50 transition-colors"
		/>
	</div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 py-6">
		{#if data.apiError}
			<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center">
				COMMUNICATION ERROR. UNABLE TO LOAD LEADERBOARD DATA.
			</div>
		{:else if shown.length === 0}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No commanders yet. Complete a level in-game and claim your XP to appear here.
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				{#each shown as p}
					<a
						href={`/leaderboard/${p.player_id}`}
						class="flex items-center gap-4 p-4 border-2 border-sky-500/20 bg-sky-500/5 hover:border-sky-400/50 hover:bg-sky-500/10 transition-colors"
					>
						<span class="w-12 text-center font-game text-lg px-2 py-1 border {medal(p.rank)}">
							#{p.rank}
						</span>
						<span class="flex-1 min-w-0 font-game text-lg text-star-white truncate">{p.username}</span>
						<span class="text-right">
							<span class="block font-game text-xl text-sky-200">{p.total_xp} XP</span>
							<span class="block text-[11px] text-text-muted">
								{p.overall_success != null ? `${p.overall_success}% avg` : '—'} · {p.entries} entries
							</span>
						</span>
					</a>
				{/each}
			</div>
		{/if}
	</div>
</div>
