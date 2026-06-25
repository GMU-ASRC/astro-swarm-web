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

<div class="flex items-center justify-between gap-4 mt-4 font-game text-xs tracking-wider text-text-muted">
	<span>
		{#if total === 0}
			No entries
		{:else}
			Showing {firstItem}–{lastItem} of {total}
		{/if}
	</span>

	<div class="flex items-center gap-2">
		<button
			type="button"
			onclick={prev}
			disabled={page <= 1}
			class="px-3 py-1 border border-sky-400/40 text-sky-200 hover:bg-sky-500/15 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
		>
			PREV
		</button>
		<span class="text-sky-300">PAGE {page} / {pageCount}</span>
		<button
			type="button"
			onclick={next}
			disabled={page >= pageCount}
			class="px-3 py-1 border border-sky-400/40 text-sky-200 hover:bg-sky-500/15 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
		>
			NEXT
		</button>
	</div>
</div>
