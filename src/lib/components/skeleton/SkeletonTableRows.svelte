<script lang="ts">
	import Skeleton from './Skeleton.svelte';

	let { rows = 8, columns }: { rows?: number; columns: number } = $props();
</script>

{#each Array(rows) as _, rowIndex}
	<tr aria-hidden="true">
		{#each Array(columns) as _cell, columnIndex}
			<td><Skeleton width={`${45 + (((rowIndex + 1) * (columnIndex + 3) * 7) % 45)}%`} height="0.8rem" /></td>
		{/each}
	</tr>
{/each}
