<script lang="ts">
	import { onMount } from 'svelte';
	import { BattleState, drawBattle } from '$lib/ts/live_battle';

	let canvas: HTMLCanvasElement;
	let animationId: number;
	let state = new BattleState();

	function rebuild() {
		const w = canvas.clientWidth;
		const h = canvas.clientHeight;
		canvas.width = w;
		canvas.height = h;
		state.init(w, h);
	}

	onMount(() => {
		const ctx = canvas.getContext('2d')!;
		let lastTime = performance.now();

		rebuild();

		const resizeObserver = new ResizeObserver(() => rebuild());
		resizeObserver.observe(canvas);

		function loop(now: number) {
			const delta = Math.min((now - lastTime) / 1000, 0.1); 
			lastTime = now;

			state.update(delta);
			drawBattle(ctx, state);
			
			animationId = requestAnimationFrame(loop);
		}

		animationId = requestAnimationFrame(loop);

		return () => {
			cancelAnimationFrame(animationId);
			resizeObserver.disconnect();
		};
	});
</script>

<canvas bind:this={canvas} class="live-battle-canvas"></canvas>

<style>
	.live-battle-canvas {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		pointer-events: none;
		z-index: 0;
		image-rendering: pixelated;
		opacity: 0.6;
	}
</style>
