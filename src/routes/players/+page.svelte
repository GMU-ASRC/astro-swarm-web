<script lang="ts">
	import type { PlayerListItem } from '$lib/ts/evaluation';

	interface PageData {
		players: PlayerListItem[];
		apiError: boolean;
	}

	let { data }: { data: PageData } = $props();

	function statusColor(status: string): string {
		if (status === 'done') return 'text-green-300 border-green-400/40 bg-green-400/10';
		if (status === 'running' || status === 'queued') return 'text-sky-300 border-sky-400/40 bg-sky-400/10';
		return 'text-red-300 border-red-400/40 bg-red-400/10';
	}
</script>

<svelte:head>
	<title>Players — AstroSwarm</title>
	<meta name="description" content="Players who have completed the FARP scenario and the performance of their defender algorithms." />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-10">
		<h1 class="font-game text-[clamp(1.8rem,4vw,3rem)] font-bold text-star-white leading-tight mb-3" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			Players
		</h1>
		<p class="font-sim text-base text-text-muted leading-relaxed max-w-150">
			Every commander who has cleared the FARP scenario has their defender algorithm benchmarked headlessly across swarm sizes. Open a player to see their capture-rate curve.
		</p>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 py-12 max-md:py-8">
		{#if data.apiError}
			<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center">
				COMMUNICATION ERROR. UNABLE TO LOAD PLAYER DATA.
			</div>
		{:else if data.players.length === 0}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No evaluations yet. Complete the FARP scenario in-game to appear here.
			</div>
		{:else}
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				{#each data.players as player}
					<a
						href={`/players/${player.player_id}`}
						class="block p-5 border-2 border-sky-500/20 bg-sky-500/5 hover:border-sky-400/50 hover:bg-sky-500/10 transition-colors"
					>
						<div class="flex items-center justify-between gap-3">
							<span class="font-game text-lg text-star-white tracking-wide">{player.username}</span>
							<span class="text-[10px] font-game tracking-wider px-2 py-1 border {statusColor(player.status)}">
								{player.status.toUpperCase()}
							</span>
						</div>
						<div class="mt-3 text-xs text-text-muted">
							N = 1…{player.n_max} · {player.trials} trials each
						</div>
					</a>
				{/each}
			</div>
		{/if}
	</div>
</div>
