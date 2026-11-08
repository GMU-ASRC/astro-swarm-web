<script lang="ts">
	import ScratchBlock from '$lib/components/blocks/ScratchBlock.svelte';
	import type { Block, BlockNames, BlockScript } from '$lib/ts/blockDefs';

	let {
		scripts = [],
		names = {},
		empty = 'No algorithm data available.'
	}: { scripts?: BlockScript[]; names?: BlockNames; empty?: string } = $props();

	let nonEmpty = $derived(
		scripts
			.filter((script) => (script.blocks ?? []).length > 0)
			.slice()
			.sort((left, right) => (left.y ?? 0) - (right.y ?? 0) || (left.x ?? 0) - (right.x ?? 0))
	);

	function countBlocks(blocks: Block[]): number {
		return blocks.reduce((total, block) => total + 1 + countBlocks(block.children ?? []), 0);
	}

	let blockCount = $derived(nonEmpty.reduce((total, script) => total + countBlocks(script.blocks ?? []), 0));
</script>

{#if nonEmpty.length === 0}
	<p class="text-dim text-sm">{empty}</p>
{:else}
	<div class="workspace">
		<div class="workspace-meta">
			{nonEmpty.length} {nonEmpty.length === 1 ? 'script' : 'scripts'} · {blockCount} {blockCount === 1 ? 'block' : 'blocks'}
		</div>
		<div class="workspace-board">
			{#each nonEmpty as script}
				<div class="stack">
					{#each script.blocks ?? [] as block}
						<ScratchBlock {block} {names} />
					{/each}
				</div>
			{/each}
		</div>
	</div>
{/if}

<style>
	.workspace {
		border: 1px solid #514e76;
		border-radius: 6px;
		background-color: #11101b;
		background-image: radial-gradient(rgba(255, 255, 255, 0.07) 1px, transparent 1.2px);
		background-size: 24px 24px;
		overflow-x: auto;
	}

	.workspace-meta {
		padding: 0.55rem 0.9rem;
		border-bottom: 1px solid #2a2840;
		background: #211f32;
		color: #8085a0;
		font-size: 0.72rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.workspace-board {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-start;
		gap: 26px 32px;
		padding: 20px 20px 28px;
	}

	.stack {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
	}
</style>
