<script lang="ts">
	import Skeleton from './Skeleton.svelte';

	let { count = 8 }: { count?: number } = $props();
</script>

<div class="skeleton-list" role="status" aria-label="Loading">
	{#each Array(count) as _, index}
		<div class="card skeleton-row">
			<Skeleton width="2rem" height="2rem" radius="50%" />
			<div class="skeleton-row-text">
				<Skeleton width={`${30 + ((index * 13) % 35)}%`} height="0.95rem" />
				<Skeleton width="22%" height="0.7rem" />
			</div>
			<Skeleton width="4.5rem" height="1.4rem" />
		</div>
	{/each}
</div>

<style>
	.skeleton-list {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}

	.skeleton-row {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 0.85rem 1.1rem;
	}

	.skeleton-row-text {
		display: flex;
		flex: 1;
		flex-direction: column;
		gap: 0.45rem;
	}
</style>
