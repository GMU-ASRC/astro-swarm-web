<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		count,
		noun = 'entry',
		nounPlural = 'entries',
		onclear,
		children
	}: {
		count: number;
		noun?: string;
		nounPlural?: string;
		onclear: () => void;
		children: Snippet;
	} = $props();
</script>

{#if count > 0}
	<div class="bulk-bar">
		<span class="count">{count} {count === 1 ? noun : nounPlural} selected</span>
		{@render children()}
		<button type="button" class="clear" onclick={onclear}>Clear selection</button>
	</div>
{/if}

<style>
	.bulk-bar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem;
		margin: 0 0 0.75rem 0;
		padding: 0.7rem 0.9rem;
		background: var(--color-surface);
		border: 1px solid var(--color-line);
		border-left: 2px solid var(--color-brand);
	}

	.bulk-bar :global(button) {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.4rem 0.75rem;
	}

	.count {
		margin-right: 0.35rem;
		color: var(--color-heading);
		font-size: 0.85rem;
		font-weight: 600;
	}

	.clear {
		margin-left: auto;
	}
</style>
