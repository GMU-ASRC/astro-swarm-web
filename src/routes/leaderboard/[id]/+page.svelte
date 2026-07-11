<script lang="ts">
	interface LevelStat {
		level_number: number;
		success_rate: number | null;
		xp: number;
		rank: number | null;
		players: number;
	}

	interface EntryRow {
		id: string;
		level_id: string;
		level_number?: number;
		is_attack?: boolean;
		status: string;
		success_rate: number | null;
		xp_awarded: number | null;
		game_version?: string;
		created_at: string;
	}

	interface Profile {
		player_id: string;
		username: string;
		total_xp: number;
		overall_success: number | null;
		overall_rank: number | null;
		total_players: number;
		entries: number;
		levels: LevelStat[];
		recent_entries: EntryRow[];
	}

	interface PageData {
		profile: Profile;
	}

	let { data }: { data: PageData } = $props();
	let p = $derived(data.profile);

	function when(iso: string): string {
		const d = new Date(iso);
		const mm = String(d.getMonth() + 1).padStart(2, '0');
		const dd = String(d.getDate()).padStart(2, '0');
		return `${mm}/${dd}/${d.getFullYear()}`;
	}

	function levelName(n: number): string {
		if (n >= 3) return `Level ${n} · Evasion`;
		return `Level ${n} · Defense`;
	}

	function statusColor(status: string): string {
		if (status === 'done') return 'text-green-300 border-green-400/40 bg-green-400/10';
		if (status === 'running' || status === 'queued') return 'text-sky-300 border-sky-400/40 bg-sky-400/10';
		return 'text-red-300 border-red-400/40 bg-red-400/10';
	}
</script>

<svelte:head>
	<title>{p.username} — AstroSwarm</title>
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-260 mx-auto px-8 max-sm:px-5 pt-12 pb-6">
		<a href="/leaderboard" class="font-game text-sm text-text-muted hover:text-accent-cyan transition-colors no-underline mb-6 inline-block">
			← BACK TO LEADERBOARD
		</a>
		<h1 class="font-game text-[clamp(1.8rem,4vw,3rem)] font-bold text-star-white leading-tight mb-3" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			{p.username}
		</h1>
		<p class="font-sim text-base text-text-muted">
			Rank #{p.overall_rank ?? '—'} of {p.total_players} · {p.total_xp} XP total
		</p>
	</div>

	<div class="max-w-260 mx-auto px-8 max-sm:px-5 py-6 grid grid-cols-2 md:grid-cols-4 gap-4">
		<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
			<div class="text-[11px] text-text-muted">TOTAL XP</div>
			<div class="font-game text-2xl text-sky-200">{p.total_xp}</div>
		</div>
		<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
			<div class="text-[11px] text-text-muted">OVERALL SUCCESS</div>
			<div class="font-game text-2xl text-star-white">{p.overall_success != null ? `${p.overall_success}%` : '—'}</div>
		</div>
		<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
			<div class="text-[11px] text-text-muted">OVERALL RANK</div>
			<div class="font-game text-2xl text-star-white">#{p.overall_rank ?? '—'}</div>
		</div>
		<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
			<div class="text-[11px] text-text-muted">ENTRIES</div>
			<div class="font-game text-2xl text-star-white">{p.entries}</div>
		</div>
	</div>

	<div class="max-w-260 mx-auto px-8 max-sm:px-5 py-6">
		<h2 class="font-game text-xl text-star-white mb-4">PER-LEVEL STATS</h2>
		{#if p.levels.length === 0}
			<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted">No level stats yet.</div>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
				{#each p.levels as lv}
					<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 flex items-center justify-between">
						<div>
							<div class="font-game text-star-white">{levelName(lv.level_number)}</div>
							<div class="text-[11px] text-text-muted">
								{lv.success_rate != null ? `${lv.success_rate}% ${lv.level_number >= 3 ? 'evasion' : 'detection'}` : 'no data'} · {lv.xp} XP
							</div>
						</div>
						<div class="text-right">
							<div class="font-game text-lg text-sky-200">#{lv.rank ?? '—'}</div>
							<div class="text-[11px] text-text-muted">of {lv.players}</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div class="max-w-260 mx-auto px-8 max-sm:px-5 py-6">
		<h2 class="font-game text-xl text-star-white mb-4">ENTRIES</h2>
		{#if p.recent_entries.length === 0}
			<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted">No entries.</div>
		{:else}
			<div class="flex flex-col gap-2">
				{#each p.recent_entries as e}
					<a
						href={`/levels/${e.id}`}
						class="flex items-center gap-4 p-4 border-2 border-sky-500/20 bg-sky-500/5 hover:border-sky-400/50 hover:bg-sky-500/10 transition-colors"
					>
						<span class="font-game text-sm text-sky-200 w-24">{levelName(e.level_number ?? 1).split(' · ')[0]}</span>
						<span class="flex-1 text-xs text-text-muted">
							{e.success_rate != null ? `${e.success_rate}% ${e.is_attack ? 'evasion' : 'detection'}` : 'pending'}
							· {e.xp_awarded != null ? `${e.xp_awarded} XP` : 'unclaimed'}
						</span>
						<span class="text-[11px] font-mono text-text-muted/70">{e.game_version ?? 'v0.0.4'}</span>
						<span class="text-[10px] font-game tracking-wider px-2 py-1 border {statusColor(e.status)}">{e.status.toUpperCase()}</span>
						<span class="text-[11px] text-text-muted/70 w-20 text-right">{when(e.created_at)}</span>
					</a>
				{/each}
			</div>
		{/if}
	</div>
</div>
