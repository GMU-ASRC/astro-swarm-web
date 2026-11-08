<script lang="ts">
	import Skeleton from './Skeleton.svelte';

	let { lines = 4, media = false }: { lines?: number; media?: boolean } = $props();
</script>

<div class="skeleton-panel" role="status" aria-label="Loading">
	{#if media}
		<Skeleton height="auto" class="skeleton-media" radius="6px" />
	{/if}
	{#each Array(lines) as _, index}
		<Skeleton width={index === lines - 1 ? '55%' : `${80 + ((index * 11) % 20)}%`} />
	{/each}
</div>

<style>
	.skeleton-panel {
		display: flex;
		flex-direction: column;
		gap: 0.7rem;
	}

	.skeleton-panel :global(.skeleton-media) {
		aspect-ratio: 16 / 9;
		margin-bottom: 0.5rem;
	}
</style>
