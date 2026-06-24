<script lang="ts">
	import { onDestroy } from 'svelte';
	import type { Replay } from '$lib/ts/evaluation';

	let { replay }: { replay: Replay } = $props();

	let canvas: HTMLCanvasElement | undefined = $state();
	let playing = $state(true);
	let frameIndex = $state(0);
	let speed = $state(1);

	const WIDTH = 800;
	const HEIGHT = 450;

	let raf = 0;
	let acc = 0;
	let last = 0;
	let subFrame = 0;
	let stars: { x: number; y: number; r: number; a: number }[] = [];

	function sampleFrame(i: number, t: number): number[] {
		const a = replay.frames[i] ?? [];
		const b = replay.frames[Math.min(replay.frames.length - 1, i + 1)] ?? a;
		const out: number[] = [];
		const slots = a.length / 3;
		for (let s = 0; s < slots; s++) {
			const ax = a[s * 3];
			const ay = a[s * 3 + 1];
			const arot = a[s * 3 + 2];
			const bx = b[s * 3];
			const by = b[s * 3 + 1];
			const brot = b[s * 3 + 2];
			const dead = ax < 0 || bx < 0;
			out.push(dead ? ax : ax + (bx - ax) * t);
			out.push(dead ? ay : ay + (by - ay) * t);
			out.push(arot + (brot - arot) * t);
		}
		return out;
	}

	const outcomeMeta: Record<string, { label: string; color: string }> = {
		win: { label: 'INTERCEPTED', color: '#4ade80' },
		lose: { label: 'PLANET HIT', color: '#f87171' },
		timeout: { label: 'TIMEOUT', color: '#fbbf24' }
	};

	let meta = $derived(outcomeMeta[replay.outcome] ?? { label: replay.outcome.toUpperCase(), color: '#cbd5f5' });
	let frameCount = $derived(Math.max(1, replay.frames.length));
	let totalSeconds = $derived(frameCount / Math.max(1, replay.fps));
	let currentSeconds = $derived(frameIndex / Math.max(1, replay.fps));

	function makeStars(aw: number, ah: number) {
		let seed = 1337;
		const rand = () => {
			seed = (seed * 1664525 + 1013904223) % 4294967296;
			return seed / 4294967296;
		};
		const out = [];
		for (let i = 0; i < 140; i++) {
			out.push({ x: rand() * aw, y: rand() * ah, r: 0.6 + rand() * 1.4, a: 0.1 + rand() * 0.45 });
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

		ctx.fillStyle = '#05060c';
		ctx.fillRect(0, 0, WIDTH, HEIGHT);

		for (const s of stars) {
			ctx.fillStyle = `rgba(255,255,255,${s.a})`;
			ctx.fillRect(s.x * sx, s.y * sy, s.r, s.r);
		}

		ctx.strokeStyle = 'rgba(124,158,255,0.35)';
		ctx.lineWidth = 2;
		ctx.strokeRect(1, 1, WIDTH - 2, HEIGHT - 2);

		const [px, py, pr] = replay.planet;
		const cx = px * sx;
		const cy = py * sy;
		const cr = pr * sx;

		const glow = ctx.createRadialGradient(cx, cy, cr, cx, cy, cr * 1.8);
		glow.addColorStop(0, 'rgba(74,222,128,0.25)');
		glow.addColorStop(1, 'rgba(74,222,128,0)');
		ctx.fillStyle = glow;
		ctx.beginPath();
		ctx.arc(cx, cy, cr * 1.8, 0, Math.PI * 2);
		ctx.fill();

		const body = ctx.createRadialGradient(cx - cr * 0.3, cy - cr * 0.3, cr * 0.2, cx, cy, cr);
		body.addColorStop(0, '#6fbf8f');
		body.addColorStop(1, '#2b5a42');
		ctx.fillStyle = body;
		ctx.beginPath();
		ctx.arc(cx, cy, cr, 0, Math.PI * 2);
		ctx.fill();
		ctx.strokeStyle = 'rgba(255,255,255,0.18)';
		ctx.lineWidth = 1;
		ctx.stroke();

		const total = frame.length / 3;

		const coneRadius = (replay.view ?? 300) * sx;
		const half = ((replay.fov ?? 70) * Math.PI) / 360;
		for (let s = 0; s < replay.defenders; s++) {
			const x = frame[s * 3];
			const y = frame[s * 3 + 1];
			const rot = (frame[s * 3 + 2] * Math.PI) / 180;
			drawCone(ctx, x * sx, y * sy, rot, coneRadius, half);
		}

		for (let s = 0; s < total; s++) {
			const x = frame[s * 3];
			const y = frame[s * 3 + 1];
			const rot = (frame[s * 3 + 2] * Math.PI) / 180;
			const isEnemy = s === replay.defenders;
			if (isEnemy && x < 0) continue;
			drawShip(ctx, x * sx, y * sy, rot, isEnemy);
		}
	}

	function drawCone(ctx: CanvasRenderingContext2D, x: number, y: number, rot: number, radius: number, half: number) {
		const grad = ctx.createRadialGradient(x, y, 0, x, y, radius);
		grad.addColorStop(0, 'rgba(124,158,255,0.28)');
		grad.addColorStop(1, 'rgba(124,158,255,0.04)');
		ctx.beginPath();
		ctx.moveTo(x, y);
		ctx.arc(x, y, radius, rot - half, rot + half);
		ctx.closePath();
		ctx.fillStyle = grad;
		ctx.fill();
		ctx.strokeStyle = 'rgba(124,158,255,0.25)';
		ctx.lineWidth = 1;
		ctx.stroke();
	}

	function drawShip(ctx: CanvasRenderingContext2D, x: number, y: number, rot: number, isEnemy: boolean) {
		const scale = isEnemy ? 1.1 : 1.0;
		ctx.save();
		ctx.translate(x, y);
		ctx.rotate(rot);
		ctx.beginPath();
		ctx.moveTo(8 * scale, 0);
		ctx.lineTo(-6 * scale, -5 * scale);
		ctx.lineTo(-6 * scale, 5 * scale);
		ctx.closePath();
		ctx.fillStyle = isEnemy ? '#ff6a52' : '#7c9eff';
		ctx.fill();
		ctx.strokeStyle = 'rgba(5,5,12,0.9)';
		ctx.lineWidth = 1.4;
		ctx.stroke();
		ctx.restore();
	}

	function tick(now: number) {
		if (!playing) {
			last = now;
			raf = requestAnimationFrame(tick);
			return;
		}
		if (last === 0) last = now;
		acc += ((now - last) / 1000) * speed;
		last = now;
		const step = 1 / Math.max(1, replay.fps);
		while (acc >= step) {
			acc -= step;
			frameIndex = (frameIndex + 1) % frameCount;
		}
		subFrame = Math.min(0.999, acc / step);
		draw(sampleFrame(frameIndex, subFrame));
		raf = requestAnimationFrame(tick);
	}

	function restart() {
		frameIndex = 0;
		acc = 0;
		playing = true;
	}

	function stepBy(delta: number) {
		playing = false;
		frameIndex = Math.min(frameCount - 1, Math.max(0, frameIndex + delta));
	}

	$effect(() => {
		replay;
		stars = makeStars(replay.arena[0], replay.arena[1]);
		frameIndex = 0;
		acc = 0;
		last = 0;
		playing = true;
		cancelAnimationFrame(raf);
		raf = requestAnimationFrame(tick);
	});

	$effect(() => {
		if (!playing) draw(sampleFrame(frameIndex, 0));
	});

	onDestroy(() => cancelAnimationFrame(raf));
</script>

<div class="flex flex-col gap-3">
	<div class="flex items-center justify-between flex-wrap gap-2">
		<div class="flex items-center gap-4 text-xs text-text-muted">
			<span class="flex items-center gap-1.5"><span class="inline-block w-3 h-3" style="background:#7c9eff"></span> Defender</span>
			<span class="flex items-center gap-1.5"><span class="inline-block w-3 h-3" style="background:#ff6a52"></span> Raider</span>
		</div>
		<div class="text-sm font-sim tracking-wider" style={`color:${meta.color}`}>{meta.label}</div>
	</div>

	<canvas bind:this={canvas} width={WIDTH} height={HEIGHT} class="w-full bg-[#05060c] border border-sky-500/20"></canvas>

	<div class="flex items-center gap-2 flex-wrap">
		<button type="button" class="px-2.5 py-1 border border-sky-500/30 text-sky-200 text-xs hover:bg-sky-500/10" onclick={restart} aria-label="Restart">⟲</button>
		<button type="button" class="px-2.5 py-1 border border-sky-500/30 text-sky-200 text-xs hover:bg-sky-500/10" onclick={() => stepBy(-1)} aria-label="Step back">‹</button>
		<button type="button" class="px-3 py-1 border border-sky-500/30 text-sky-200 text-xs hover:bg-sky-500/10 min-w-16" onclick={() => (playing = !playing)}>
			{playing ? 'PAUSE' : 'PLAY'}
		</button>
		<button type="button" class="px-2.5 py-1 border border-sky-500/30 text-sky-200 text-xs hover:bg-sky-500/10" onclick={() => stepBy(1)} aria-label="Step forward">›</button>

		<div class="flex items-center gap-1 ml-1">
			{#each [0.5, 1, 2] as s}
				<button
					type="button"
					onclick={() => (speed = s)}
					class="px-2 py-1 border text-xs transition-colors {speed === s ? 'border-sky-400 bg-sky-500/15 text-sky-200' : 'border-sky-500/20 text-text-muted hover:border-sky-400/50'}"
				>{s}x</button>
			{/each}
		</div>

		<span class="ml-auto text-xs text-text-muted tabular-nums">{currentSeconds.toFixed(1)}s / {totalSeconds.toFixed(1)}s</span>
	</div>

	<input
		type="range"
		min="0"
		max={frameCount - 1}
		bind:value={frameIndex}
		oninput={() => (playing = false)}
		class="w-full"
	/>
</div>
