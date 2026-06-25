<script lang="ts">
	let { page = $bindable(1), pageSize = 25, total = 0 } = $props();

	const pageCount = $derived(Math.max(1, Math.ceil(total / pageSize)));
	const firstItem = $derived(total === 0 ? 0 : (page - 1) * pageSize + 1);
	const lastItem = $derived(Math.min(page * pageSize, total));

	$effect(() => {
		if (page > pageCount) page = pageCount;
	});

	function prev() {
		if (page > 1) page -= 1;
	}

	function next() {
		if (page < pageCount) page += 1;
	}
</script>

<div class="pagination">
	<span>
		{#if total === 0}
			No entries
		{:else}
			Showing {firstItem}–{lastItem} of {total}
		{/if}
	</span>

	<span class="controls">
		<button type="button" onclick={prev} disabled={page <= 1}>Prev</button>
		<span>Page {page} / {pageCount}</span>
		<button type="button" onclick={next} disabled={page >= pageCount}>Next</button>
	</span>
</div>

<style>
	.pagination {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin-top: 0.75rem;
		flex-wrap: wrap;
	}

	.controls {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	button {
		font: inherit;
		padding: 0.3rem 0.6rem;
		border: 1px solid #6b7280;
		background: #f9fafb;
		cursor: pointer;
	}

	button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
