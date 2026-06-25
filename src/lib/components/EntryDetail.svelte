<script lang="ts">
	let { record }: { record: Record<string, any> } = $props();

	function isComplex(value: any): boolean {
		return value !== null && typeof value === 'object';
	}
</script>

<div class="flex flex-col gap-3">
	{#each Object.entries(record) as [key, value]}
		<div class="border border-sky-500/20 bg-sky-500/5">
			<div class="px-4 py-2 border-b border-sky-500/20 font-game text-xs tracking-wider text-sky-400 uppercase">
				{key}
			</div>
			{#if isComplex(value)}
				<pre class="p-4 text-xs text-text-muted overflow-x-auto whitespace-pre-wrap break-words">{JSON.stringify(value, null, 2)}</pre>
			{:else}
				<div class="p-4 text-sm text-star-white break-words">
					{value === null || value === '' ? '—' : value}
				</div>
			{/if}
		</div>
	{/each}
</div>
