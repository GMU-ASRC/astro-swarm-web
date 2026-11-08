<script lang="ts">
	import Skeleton from './Skeleton.svelte';
	import SkeletonPanel from './SkeletonPanel.svelte';
</script>

<div class="page" role="status" aria-label="Loading page">
	<div class="shell shell-wide">
		<div class="page-head">
			<Skeleton width="7rem" height="0.8rem" />
			<div class="skeleton-title"><Skeleton width="min(28rem, 80%)" height="2.4rem" /></div>
			<Skeleton width="min(40rem, 95%)" height="0.9rem" />
		</div>
		<div class="skeleton-stats">
			{#each Array(4) as _}
				<div class="stat">
					<Skeleton width="55%" height="1.6rem" />
					<div class="skeleton-gap"><Skeleton width="75%" height="0.7rem" /></div>
				</div>
			{/each}
		</div>
		<div class="skeleton-body">
			<div class="card skeleton-box"><SkeletonPanel media lines={2} /></div>
			<div class="card skeleton-box"><SkeletonPanel lines={6} /></div>
		</div>
	</div>
</div>

<style>
	.skeleton-title {
		margin: 1rem 0 1.1rem;
	}

	.skeleton-stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: 0.75rem;
		margin-bottom: 1.5rem;
	}

	.skeleton-gap {
		margin-top: 0.6rem;
	}

	.skeleton-body {
		display: grid;
		grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
		gap: 1rem;
	}

	.skeleton-box {
		padding: 1.25rem;
	}

	@media (max-width: 860px) {
		.skeleton-body {
			grid-template-columns: minmax(0, 1fr);
		}
	}
</style>
