<script lang="ts">
	import Skeleton from './Skeleton.svelte';

	let { count = 6, minWidth = '17rem' }: { count?: number; minWidth?: string } = $props();
</script>

<div class="skeleton-cards" style:grid-template-columns={`repeat(auto-fill, minmax(${minWidth}, 1fr))`} role="status" aria-label="Loading">
	{#each Array(count) as _, index}
		<div class="card skeleton-card">
			<div class="skeleton-card-head">
				<Skeleton width={`${55 + ((index * 17) % 30)}%`} height="1.05rem" />
				<Skeleton width="3.5rem" height="1.2rem" radius="3px" />
			</div>
			<Skeleton width="40%" height="0.75rem" />
			<div class="skeleton-card-stats">
				<Skeleton height="2.4rem" />
				<Skeleton height="2.4rem" />
				<Skeleton height="2.4rem" />
			</div>
		</div>
	{/each}
</div>

<style>
	.skeleton-cards {
		display: grid;
		gap: 1rem;
	}

	.skeleton-card {
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
		padding: 1.15rem 1.25rem;
	}

	.skeleton-card-head {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
	}

	.skeleton-card-stats {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.6rem;
		margin-top: 0.35rem;
	}
</style>
