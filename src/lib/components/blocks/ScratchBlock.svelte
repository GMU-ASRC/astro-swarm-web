<script lang="ts">
	import ScratchBlock from './ScratchBlock.svelte';
	import { CATEGORY_COLORS, blockDef, blockParts, isContainerBlock, isHatBlock, type Block, type BlockNames } from '$lib/ts/blockDefs';
	import { ARM_WIDTH, FOOTER_HEIGHT, HAT_HEIGHT, NOTCH_DEPTH, SHADOW_OFFSET, highlightSegments, outline, pointList } from '$lib/ts/blockShape';

	let { block, names = {} }: { block: Block; names?: BlockNames } = $props();

	const PAD_LEFT = 10;
	const PAD_RIGHT = 6;
	const HEADER_GAP = 4;

	let width = $state(0);
	let height = $state(0);
	let headerHeight = $state(0);
	let mouthHeight = $state(0);

	let colors = $derived(CATEGORY_COLORS[blockDef(block.type).category]);
	let isHat = $derived(isHatBlock(block.type));
	let isContainer = $derived(isContainerBlock(block.type));
	let children = $derived(block.children ?? []);
	let padTop = $derived(isHat ? HAT_HEIGHT + 8 : NOTCH_DEPTH + 5);
	let padBottom = $derived(isContainer ? FOOTER_HEIGHT : 7);
	let mouth = $derived(isContainer ? { x: PAD_LEFT + ARM_WIDTH, y: padTop + headerHeight + HEADER_GAP, height: mouthHeight } : null);
	let shape = $derived(width > 0 ? outline(width, height, isHat, mouth) : []);
</script>

<div
	class="scratch-block"
	style:padding={`${padTop}px ${PAD_RIGHT}px ${padBottom}px ${PAD_LEFT}px`}
	bind:clientWidth={width}
	bind:clientHeight={height}
>
	{#if shape.length > 0}
		<svg class="outline" width={width} height={height + NOTCH_DEPTH + SHADOW_OFFSET} aria-hidden="true">
			<polygon points={pointList(shape, SHADOW_OFFSET)} fill="rgba(0,0,0,0.35)" />
			<polygon points={pointList(shape)} fill={colors.background} />
			{#each highlightSegments(width, isHat) as [start, end]}
				<line x1={start.x} y1={start.y} x2={end.x} y2={end.y} stroke="rgba(255,255,255,0.22)" stroke-width="2" />
			{/each}
			<polygon points={pointList(shape)} fill="none" stroke={colors.edge} stroke-width="2" stroke-linejoin="round" />
		</svg>
	{/if}
	<div class="header" bind:clientHeight={headerHeight}>
		{#each blockParts(block, names) as part}
			{#if part.kind === 'text'}
				<span class="label">{part.text}</span>
			{:else}
				<span class="slot" class:choice={part.kind === 'choice'}>{part.text}</span>
			{/if}
		{/each}
	</div>
	{#if isContainer}
		<div class="mouth" style:margin-top={`${HEADER_GAP}px`} style:margin-left={`${ARM_WIDTH}px`} bind:clientHeight={mouthHeight}>
			{#each children as child}
				<ScratchBlock block={child} {names} />
			{/each}
		</div>
	{/if}
</div>

<style>
	.scratch-block {
		position: relative;
		width: max-content;
		min-width: 64px;
	}

	.outline {
		position: absolute;
		top: 0;
		left: 0;
		overflow: visible;
		pointer-events: none;
	}

	.header {
		position: relative;
		display: flex;
		flex-wrap: nowrap;
		align-items: center;
		gap: 6px;
		min-height: 24px;
	}

	.label {
		color: #ffffff;
		font-size: 13px;
		line-height: 1.2;
		white-space: nowrap;
		text-shadow: 0 1px 0 rgba(0, 0, 0, 0.35);
	}

	.slot {
		padding: 3px 9px;
		border-radius: 8px;
		background: rgba(0, 0, 0, 0.25);
		color: #ffffff;
		font-size: 12px;
		line-height: 1.3;
		white-space: nowrap;
		font-variant-numeric: tabular-nums;
	}

	.slot.choice::after {
		content: '▾';
		margin-left: 5px;
		opacity: 0.7;
		font-size: 10px;
	}

	.mouth {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		min-height: 14px;
	}
</style>
