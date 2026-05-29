<script lang="ts">
	import '$lib/css/leaderboard.css';
	import LeaderboardTable from '$lib/components/LeaderboardTable.svelte';
	import type { LeaderboardEntry } from '$lib/ts/leaderboard';

	interface PageData {
		entries: LeaderboardEntry[];
		apiError: boolean;
	}

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Leaderboard — AstroSwarm</title>
	<meta name="description" content="AstroSwarm Timed Local game mode leaderboards." />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-10">
		<h1 class="font-game text-[clamp(1.8rem,4vw,3rem)] font-bold text-star-white leading-tight mb-3" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			Global Leaderboard
		</h1>
		<p class="font-sim text-base text-text-muted leading-relaxed max-w-150">
			The top times for completing the Timed Local game mode. Can you build a swarm fast enough to reach the top?
		</p>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 py-12 max-md:py-8">
		{#if data.apiError}
			<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center">
				COMMUNICATION ERROR. UNABLE TO LOAD LEADERBOARD DATA.
			</div>
		{:else}
			<LeaderboardTable entries={data.entries} />
		{/if}
	</div>
</div>
