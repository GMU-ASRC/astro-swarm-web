<script lang="ts">
	import { onMount } from 'svelte';
	import { buildLayers, drawFrame, lerpMouse, type StarLayerData } from '$lib/ts/starfield';

	let canvas: HTMLCanvasElement;
	let animationId: number;

	let mouseNormX = 0;
	let mouseNormY = 0;
	let targetNormX = 0;
	let targetNormY = 0;

	let layers: StarLayerData[] = [];
	let seed = Math.floor(Math.random() * 9999999);

	function handleMouseMove(event: MouseEvent) {
		const rect = canvas.getBoundingClientRect();
		const cx = rect.width / 2;
		const cy = rect.height / 2;
		targetNormX = Math.max(-1, Math.min(1, (event.clientX - rect.left - cx) / cx));
		targetNormY = Math.max(-1, Math.min(1, (event.clientY - rect.top - cy) / cy));
	}

	function rebuild() {
		const w = canvas.clientWidth;
		const h = canvas.clientHeight;
		canvas.width = w;
		canvas.height = h;
		layers = buildLayers(w, h, seed);
	}

	onMount(() => {
		const ctx = canvas.getContext('2d')!;
		let lastTime = performance.now();
		let elapsed = 0;

		rebuild();

		const resizeObserver = new ResizeObserver(() => rebuild());
		resizeObserver.observe(canvas);

		window.addEventListener('mousemove', handleMouseMove);

		function loop(now: number) {
			const delta = (now - lastTime) / 1000;
			lastTime = now;
			elapsed += delta;

			[mouseNormX, mouseNormY] = lerpMouse(mouseNormX, mouseNormY, targetNormX, targetNormY, delta);

			drawFrame(ctx, layers, canvas.width, canvas.height, mouseNormX, mouseNormY, elapsed);
			animationId = requestAnimationFrame(loop);
		}

		animationId = requestAnimationFrame(loop);

		return () => {
			cancelAnimationFrame(animationId);
			resizeObserver.disconnect();
			window.removeEventListener('mousemove', handleMouseMove);
		};
	});
</script>

<canvas bind:this={canvas} class="starfield-canvas"></canvas>

<style>
	.starfield-canvas {
		position: fixed;
		inset: 0;
		width: 100%;
		height: 100%;
		pointer-events: none;
		z-index: 0;
		image-rendering: pixelated;
	}
</style>
