<script lang="ts">
	import type { LeaderboardEntry } from '$lib/ts/leaderboard';

	interface Props {
		entries: LeaderboardEntry[];
	}

	let { entries }: Props = $props();

	function formatTime(seconds: number): string {
		return seconds.toFixed(2) + 's';
	}
</script>

<div class="leaderboard-container overflow-x-auto">
	<table class="leaderboard-table font-game">
		<thead>
			<tr>
				<th class="leaderboard-th w-24">RANK</th>
				<th class="leaderboard-th">PILOT</th>
				<th class="leaderboard-th text-right">TIME</th>
			</tr>
		</thead>
		<tbody>
			{#if entries.length === 0}
				<tr>
					<td colspan="3" class="leaderboard-td text-center text-text-muted py-8">
						NO RECORDS FOUND. BE THE FIRST!
					</td>
				</tr>
			{:else}
				{#each entries as entry, index}
					<tr class="leaderboard-tr">
						<td class="leaderboard-td">
							<span class="
								{index === 0 ? 'rank-1' : ''}
								{index === 1 ? 'rank-2' : ''}
								{index === 2 ? 'rank-3' : ''}
							">
								#{index + 1}
							</span>
						</td>
						<td class="leaderboard-td tracking-wide">
							<a href="/leaderboard/{entry.id}" class="text-star-white no-underline hover:text-accent-cyan transition-colors">
								{entry.username.toUpperCase()}
							</a>
						</td>
						<td class="leaderboard-td text-right text-accent-cyan">
							{formatTime(entry.time_seconds)}
						</td>
					</tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>
