<script lang="ts">
	import type { LevelStat } from '$lib/ts/adminStats';
	import { levelById } from '$lib/ts/levels';
	import { formatAgo, formatCount, formatPercent } from '$lib/ts/adminFormat';

	let { levels }: { levels: LevelStat[] } = $props();

	let peak = $derived(Math.max(1, ...levels.map((level) => level.entries)));
</script>

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Level</th>
				<th>Entries</th>
				<th>Players</th>
				<th>Finished</th>
				<th>Best</th>
				<th>Average</th>
				<th>Last entry</th>
			</tr>
		</thead>
		<tbody>
			{#each levels as level}
				<tr>
					<td>
						<div class="level-name">{levelById(level.level_id)?.name ?? level.level_id}</div>
						<div class="level-id">{level.level_id}</div>
					</td>
					<td>
						<div class="count-cell">
							<span>{formatCount(level.entries)}</span>
							<span class="count-bar" style:width={`${(100 * level.entries) / peak}%`}></span>
						</div>
					</td>
					<td>{formatCount(level.players)}</td>
					<td>{formatCount(level.done)}</td>
					<td>{formatPercent(level.best_rate)}</td>
					<td>{formatPercent(level.average_rate)}</td>
					<td title={level.last_entry ?? ''}>{formatAgo(level.last_entry)}</td>
				</tr>
			{:else}
				<tr><td colspan="7">No entries yet.</td></tr>
			{/each}
		</tbody>
	</table>
</div>

<style>
	.level-name {
		color: var(--color-heading);
	}

	.level-id {
		color: var(--color-faint);
		font-size: 0.72rem;
	}

	.count-cell {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		min-width: 5rem;
		font-variant-numeric: tabular-nums;
	}

	.count-bar {
		height: 3px;
		border-radius: 2px;
		background: var(--color-brand);
	}
</style>
