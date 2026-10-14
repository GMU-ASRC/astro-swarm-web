<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		id,
		title,
		description = '',
		children
	}: { id: string; title: string; description?: string; children: Snippet } = $props();
</script>

<section {id} class="settings-section">
	<header>
		<h2>{title}</h2>
		{#if description}<p class="meta">{description}</p>{/if}
	</header>
	{@render children()}
</section>

<style>
	.settings-section {
		margin: 0 0 2.5rem 0;
		scroll-margin-top: 1.5rem;
	}

	header {
		margin-bottom: 1rem;
	}

	header h2 {
		margin: 0 0 0.25rem 0;
	}
</style>
