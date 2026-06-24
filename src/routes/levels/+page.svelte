<script lang="ts">
	import type { PlayerListItem } from '$lib/ts/evaluation';

	interface PageData {
		players: PlayerListItem[];
		apiError: boolean;
	}

	let { data }: { data: PageData } = $props();

	const levels = [{ id: 'farp', label: 'FARP', enabled: true }];
	let selectedLevel = $state('farp');

	function statusColor(status: string): string {
		if (status === 'done') return 'text-green-300 border-green-400/40 bg-green-400/10';
		if (status === 'running' || status === 'queued') return 'text-sky-300 border-sky-400/40 bg-sky-400/10';
		return 'text-red-300 border-red-400/40 bg-red-400/10';
	}

	function when(iso: string): string {
		return new Date(iso).toLocaleString();
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

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 py-12 max-md:py-8">
		{#if data.apiError}
			<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center">
				COMMUNICATION ERROR. UNABLE TO LOAD LEVEL DATA.
			</div>
		{:else if data.players.length === 0}
			<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-center">
				No evaluations yet. Complete the FARP scenario in-game to appear here.
			</div>
		{:else}
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				{#each data.players as player}
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
							{#if player.status === 'running' || player.status === 'queued'}
								Benchmarking · {Math.round((player.progress ?? 0) * 100)}%
							{:else if player.success_rate !== null && player.success_rate !== undefined}
								FARP · {player.success_rate}% capture rate · {player.trials} trials
							{:else}
								FARP · {player.trials} trials
							{/if}
						</div>
						<div class="mt-1 text-[11px] text-text-muted/70">{when(player.created_at)}</div>
					</a>
				{/each}
			</div>
		{/if}
	</div>
</div>
