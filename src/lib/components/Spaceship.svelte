<script lang="ts">
	import { shipSvgMarkup, type ShipVariant } from '$lib/ts/ships';

	interface Props {
		variant: ShipVariant;
		size?: string;
		title?: string;
	}

	let { variant, size = '100%', title = '' }: Props = $props();

	const uid = $props.id();
	const clipId = `ship-clip-${uid}`;
	const markup = $derived(shipSvgMarkup(variant, clipId, size));
</script>

<span class="spaceship" role={title ? 'img' : 'presentation'} aria-label={title || undefined}>
	{@html markup}
</span>

<style>
	.spaceship {
		display: inline-flex;
		line-height: 0;
	}

	.spaceship :global(svg) {
		image-rendering: pixelated;
	}
</style>
