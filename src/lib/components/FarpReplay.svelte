<script lang="ts">
	import { onDestroy } from 'svelte';
	import type { Replay } from '$lib/ts/evaluation';

	let { replay }: { replay: Replay } = $props();

	let canvas: HTMLCanvasElement | undefined = $state();
	let playing = $state(true);
	let frameIndex = $state(0);

	const WIDTH = 640;
	const HEIGHT = 360;

	let raf = 0;
	let acc = 0;
	let last = 0;

	function sampleFrame(i: number, t: number): number[] {
		const a = replay.frames[i] ?? [];
		const b = replay.frames[Math.min(replay.frames.length - 1, i + 1)] ?? a;
		const out: number[] = [];
		const slots = a.length / 3;
		for (let s = 0; s < slots; s++) {
			const ax = a[s * 3];
			const bx = b[s * 3];
			const dead = ax < 0 || bx < 0;
			out.push(dead ? ax : ax + (bx - ax) * t);
			out.push(dead ? a[s * 3 + 1] : a[s * 3 + 1] + (b[s * 3 + 1] - a[s * 3 + 1]) * t);
			out.push(a[s * 3 + 2] + (b[s * 3 + 2] - a[s * 3 + 2]) * t);
		}
		return out;
	}

	function draw(frame: number[]) {
		if (!canvas) return;
		const ctx = canvas.getContext('2d');
		if (!ctx) return;

		const [aw, ah] = replay.arena;
		const sx = WIDTH / aw;
		const sy = HEIGHT / ah;

		ctx.fillStyle = '#0a0a12';
		ctx.fillRect(0, 0, WIDTH, HEIGHT);

		// planet
		const [px, py, pr] = replay.planet;
		ctx.beginPath();
		ctx.arc(px * sx, py * sy, pr * sx, 0, Math.PI * 2);
		ctx.fillStyle = '#3a7d5a';
		ctx.fill();
		ctx.strokeStyle = 'rgba(255,255,255,0.15)';
		ctx.stroke();

		const total = frame.length / 3;

		const coneRadius = (replay.view ?? 300) * sx;
		const half = ((replay.fov ?? 70) * Math.PI) / 360;
		for (let s = 0; s < replay.defenders; s++) {
			const x = frame[s * 3];
			const y = frame[s * 3 + 1];
			const rot = (frame[s * 3 + 2] * Math.PI) / 180;
			drawCone(ctx, x * sx, y * sy, rot, coneRadius, half, 'rgba(124,158,255,0.16)');
		}

		for (let s = 0; s < total; s++) {
			const x = frame[s * 3];
			const y = frame[s * 3 + 1];
			const rot = (frame[s * 3 + 2] * Math.PI) / 180;
			const isEnemy = s === replay.defenders;
			if (isEnemy && x < 0) continue; // enemy destroyed
			drawShip(ctx, x * sx, y * sy, rot, isEnemy ? '#ff6a52' : '#7c9eff');
		}

		ctx.fillStyle = 'rgba(255,255,255,0.5)';
		ctx.font = '11px monospace';
		ctx.fillText(`N=${replay.defenders}  ${replay.outcome}  frame ${frameIndex + 1}/${replay.frames.length}`, 8, 16);
	}

	function drawCone(
		ctx: CanvasRenderingContext2D,
		x: number,
		y: number,
		rot: number,
		radius: number,
		half: number,
		color: string
	) {
		ctx.beginPath();
		ctx.moveTo(x, y);
		ctx.arc(x, y, radius, rot - half, rot + half);
		ctx.closePath();
		ctx.fillStyle = color;
		ctx.fill();
	}

	function drawShip(ctx: CanvasRenderingContext2D, x: number, y: number, rot: number, color: string) {
		ctx.save();
		ctx.translate(x, y);
		ctx.rotate(rot);
		ctx.beginPath();
		ctx.moveTo(6, 0);
		ctx.lineTo(-4, -3.5);
		ctx.lineTo(-4, 3.5);
		ctx.closePath();
		ctx.fillStyle = color;
		ctx.fill();
		ctx.restore();
	}

	function tick(now: number) {
		if (!playing) {
			last = now;
			raf = requestAnimationFrame(tick);
			return;
		}
		if (last === 0) last = now;
		acc += (now - last) / 1000;
		last = now;
		const step = 1 / replay.fps;
		while (acc >= step) {
			acc -= step;
			frameIndex = (frameIndex + 1) % Math.max(1, replay.frames.length);
		}
		draw(sampleFrame(frameIndex, Math.min(0.999, acc / step)));
		raf = requestAnimationFrame(tick);
	}

	$effect(() => {
		// restart playback when the replay changes
		replay;
		frameIndex = 0;
		acc = 0;
		last = 0;
		playing = true;
		cancelAnimationFrame(raf);
		raf = requestAnimationFrame(tick);
	});

	$effect(() => {
		// redraw immediately when scrubbing while paused
		if (!playing) draw(sampleFrame(frameIndex, 0));
	});

	onDestroy(() => cancelAnimationFrame(raf));
</script>

<div class="flex flex-col gap-2">
	<canvas bind:this={canvas} width={WIDTH} height={HEIGHT} class="w-full bg-[#0a0a12] border border-sky-500/20"></canvas>
	<div class="flex items-center gap-3">
		<button
			type="button"
			class="px-3 py-1 border border-sky-500/30 text-sky-200 text-xs font-sim hover:bg-sky-500/10"
			onclick={() => (playing = !playing)}
		>
			{playing ? 'PAUSE' : 'PLAY'}
		</button>
		<input
			type="range"
			min="0"
			max={Math.max(0, replay.frames.length - 1)}
			bind:value={frameIndex}
			oninput={() => (playing = false)}
			class="flex-1"
		/>
	</div>
</div>
