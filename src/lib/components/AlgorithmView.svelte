<script lang="ts">
	import {
		CATEGORY_COLORS,
		blockDef,
		blockParts,
		isContainerBlock,
		type Block,
		type BlockNames,
		type BlockScript
	} from '$lib/ts/blockDefs';

	let {
		scripts = [],
		names = {},
		empty = 'No algorithm data available.'
	}: { scripts?: BlockScript[]; names?: BlockNames; empty?: string } = $props();

	let nonEmpty = $derived(scripts.filter((script) => (script.blocks ?? []).length > 0));
</script>

{#snippet node(block: Block)}
	{@const colors = CATEGORY_COLORS[blockDef(block.type).category]}
	<div class="rounded-md" style={`background:${colors.background};border-left:4px solid ${colors.edge}`}>
		<div class="flex flex-wrap items-center gap-2 px-3 py-1.5">
			{#each blockParts(block, names) as part}
				{#if part.kind === 'text'}
					<span class="text-white text-[13px] leading-tight">{part.text}</span>
				{:else}
					<span class="px-2 py-0.5 rounded text-white text-[12px] tabular-nums" style="background:rgba(0,0,0,0.24)">{part.text}</span>
				{/if}
			{/each}
		</div>
		{#if isContainerBlock(block.type) && (block.children?.length ?? 0) > 0}
			<div class="ml-3 mr-2 mb-2 rounded p-2 flex flex-col gap-1.5" style="background:rgba(0,0,0,0.18)">
				{#each block.children ?? [] as child}
					{@render node(child)}
				{/each}
			</div>
		{/if}
	</div>
{/snippet}

{#if nonEmpty.length === 0}
	<p class="text-dim text-sm">{empty}</p>
{:else}
	<div class="flex flex-wrap gap-5 items-start">
		{#each nonEmpty as script}
			<div class="flex flex-col gap-2 min-w-[230px]">
				{#each script.blocks ?? [] as block}
					{@render node(block)}
				{/each}
			</div>
		{/each}
	</div>
{/if}
