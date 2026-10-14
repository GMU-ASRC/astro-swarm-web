<script lang="ts">
	import type { AdminSettings } from '$lib/ts/adminSettings';

	let { settings }: { settings: AdminSettings } = $props();

	const rows = $derived(
		settings.levels
			.filter((level) => settings.pilot_level_ids.includes(level.id))
			.map((level) => ({ level, limits: settings.pilot_run_limits[level.id] }))
	);
</script>

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Level</th>
				<th>Time limit</th>
				<th>Max recorded ships</th>
				<th>Frame slots</th>
			</tr>
		</thead>
		<tbody>
			{#each rows as { level, limits } (level.id)}
				<tr>
					<td><strong>{level.name}</strong></td>
					<td>{limits?.max_seconds ?? '—'} s</td>
					<td>{limits?.max_ships ?? '—'}</td>
					<td>{limits?.frame_slots ?? 'Placements + 1'}</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
<p class="meta">
	Every recorded run is capped at {settings.pilot_max_fps} fps and pays up to {settings.pilot_max_xp} XP.
	A run past these limits is rejected on upload. These limits are set in <code>server/schemas.py</code>.
</p>
