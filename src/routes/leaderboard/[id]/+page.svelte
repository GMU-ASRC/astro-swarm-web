<script lang="ts">
	import type { LeaderboardEntry } from '$lib/ts/leaderboard';

	interface PageData {
		entry: LeaderboardEntry;
	}

	let { data }: { data: PageData } = $props();

	function formatTime(seconds: number): string {
		return seconds.toFixed(2) + 's';
	}
	
	function formatLabel(label: string): string {
		return label.replace('when_', '').replace('do_', '').replace(/_/g, ' ').toUpperCase();
	}
</script>

<svelte:head>
	<title>{data.entry.username.toUpperCase()} — AstroSwarm</title>
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-10">
		<a href="/leaderboard" class="font-game text-sm text-text-muted hover:text-accent-cyan transition-colors no-underline mb-6 inline-block">
			← BACK TO LEADERBOARD
		</a>
		<h1 class="font-game text-[clamp(1.8rem,4vw,3rem)] font-bold text-star-white leading-tight mb-3" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			PILOT: {data.entry.username.toUpperCase()}
		</h1>
		<p class="font-game text-lg text-accent-cyan tracking-wider">
			RECORD TIME: {formatTime(data.entry.time_seconds)}
		</p>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 py-12 max-md:py-8">
		<h2 class="font-game text-xl text-star-white mb-6">FLIGHT ALGORITHM</h2>
		
		<div class="bg-page-bg/80 border-2 border-btn-border p-6 max-w-xl">
			{#if !data.entry.algorithm || data.entry.algorithm.length === 0}
				<p class="font-game text-text-muted">NO ALGORITHM DATA AVAILABLE.</p>
			{:else}
				<div class="flex flex-col gap-2 font-game text-sm">
					{#each data.entry.algorithm as rule}
						<div class="px-4 py-2 border-l-4 border-accent-cyan bg-accent-blue/10 text-accent-cyan">
							{formatLabel('when_' + rule.condition)}
						</div>
						{#if rule.actions}
							{#each rule.actions as act}
								<div class="px-4 py-2 border-l-4 border-star-white bg-white/5 text-star-white ml-6">
									{formatLabel('do_' + act.id)}
								</div>
							{/each}
						{/if}
					{/each}
				</div>
			{/if}
		</div>
	</div>
</div>
