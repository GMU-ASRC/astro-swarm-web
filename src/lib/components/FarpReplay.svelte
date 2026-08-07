<script lang="ts">
	import { onDestroy } from 'svelte';
	import Icon from '@iconify/svelte';
	import type { Replay } from '$lib/ts/evaluation';

	let { replay }: { replay: Replay } = $props();

	let canvas: HTMLCanvasElement | undefined = $state();
	let playing = $state(true);
	let frameIndex = $state(0);
	let speed = $state(1);
	let loop = $state(true);
	let selectedSlot = $state<number | null>(null);

	const SPEEDS = [0.5, 1, 2, 3];
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

		if (selectedSlot !== null && selectedSlot < total) {
			const hx = frame[selectedSlot * 3];
			const hy = frame[selectedSlot * 3 + 1];
			if (hx >= 0) {
				ctx.beginPath();
				ctx.arc(hx * sx, hy * sy, 10, 0, Math.PI * 2);
				ctx.strokeStyle = '#ffd54a';
				ctx.lineWidth = 2;
				ctx.stroke();
			}
		}

		ctx.fillStyle = 'rgba(255,255,255,0.5)';
		ctx.font = '11px monospace';
		const seconds = (frameIndex / (replay.fps || 1)).toFixed(1);
		ctx.fillText(
			`N=${replay.defenders}  ${replay.outcome}  frame ${frameIndex + 1}/${replay.frames.length}  t=${seconds}s`,
			8,
			16
		);
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
		acc += ((now - last) / 1000) * speed;
		last = now;
		const step = 1 / replay.fps;
		const lastFrame = Math.max(0, replay.frames.length - 1);
		while (acc >= step) {
			acc -= step;
			if (frameIndex >= lastFrame) {
				if (loop) {
					frameIndex = 0;
				} else {
					frameIndex = lastFrame;
					playing = false;
					acc = 0;
					break;
				}
			} else {
				frameIndex += 1;
			}
		}
		draw(sampleFrame(frameIndex, Math.min(0.999, acc / step)));
		raf = requestAnimationFrame(tick);
	}

	function restart() {
		frameIndex = 0;
		acc = 0;
		last = 0;
		playing = true;
	}

	function stepBy(delta: number) {
		playing = false;
		const lastFrame = Math.max(0, replay.frames.length - 1);
		frameIndex = Math.min(lastFrame, Math.max(0, frameIndex + delta));
	}

	function shipAtCanvas(mx: number, my: number): number | null {
		const frame = sampleFrame(frameIndex, 0);
		const [aw, ah] = replay.arena;
		const sx = WIDTH / aw;
		const sy = HEIGHT / ah;
		const total = frame.length / 3;
		let best: number | null = null;
		let bestDist = 14;
		for (let s = 0; s < total; s++) {
			const x = frame[s * 3];
			const y = frame[s * 3 + 1];
			if (x < 0) continue;
			const d = Math.hypot(x * sx - mx, y * sy - my);
			if (d < bestDist) {
				bestDist = d;
				best = s;
			}
		}
		return best;
	}

	function onCanvasClick(event: MouseEvent) {
		if (!canvas) return;
		const rect = canvas.getBoundingClientRect();
		const mx = ((event.clientX - rect.left) / rect.width) * WIDTH;
		const my = ((event.clientY - rect.top) / rect.height) * HEIGHT;
		const slot = shipAtCanvas(mx, my);
		selectedSlot = slot;
		if (slot !== null) playing = false;
	}

	const selectedInfo = $derived.by(() => {
		if (selectedSlot === null) return null;
		const frame = replay.frames[frameIndex] ?? [];
		const s = selectedSlot;
		const x = frame[s * 3];
		if (x === undefined || x < 0) return null;
		const y = frame[s * 3 + 1];
		const rot = frame[s * 3 + 2] ?? 0;
		const isEnemy = s === replay.defenders;
		return {
			isEnemy,
			label: isEnemy ? 'Enemy raider' : `Defender #${s + 1}`,
			x: Math.round(x),
			y: Math.round(y),
			heading: (((Math.round(rot) % 360) + 360) % 360),
			view: replay.view,
			fov: replay.fov
		};
	});

	const tooltipPos = $derived.by(() => {
		if (selectedSlot === null) return null;
		const frame = replay.frames[frameIndex] ?? [];
		const x = frame[selectedSlot * 3];
		const y = frame[selectedSlot * 3 + 1];
		if (x === undefined || x < 0) return null;
		const [aw, ah] = replay.arena;
		return { left: (x / aw) * 100, top: (y / ah) * 100 };
	});

	$effect(() => {
		// restart playback when the replay changes
		replay;
		frameIndex = 0;
		acc = 0;
		last = 0;
		playing = true;
		selectedSlot = null;
		cancelAnimationFrame(raf);
		raf = requestAnimationFrame(tick);
	});

	$effect(() => {
		// redraw immediately when scrubbing or selecting while paused
		selectedSlot;
		if (!playing) draw(sampleFrame(frameIndex, 0));
	});

	onDestroy(() => cancelAnimationFrame(raf));
</script>

<div class="flex flex-col gap-2">
	<div class="replay-stage">
		<canvas
			bind:this={canvas}
			width={WIDTH}
			height={HEIGHT}
			onclick={onCanvasClick}
			class="replay-canvas"
		></canvas>
		{#if selectedInfo && tooltipPos}
			<div class="ship-card" style="left:{tooltipPos.left}%; top:{tooltipPos.top}%">
				<div class="ship-card-head">
					<span class="ship-card-title {selectedInfo.isEnemy ? 'enemy' : 'defender'}">{selectedInfo.label}</span>
					<button type="button" class="ship-card-close" aria-label="Close" onclick={() => (selectedSlot = null)}>×</button>
				</div>
				<dl class="ship-card-body">
					<div><dt>Position</dt><dd>{selectedInfo.x}, {selectedInfo.y}</dd></div>
					<div><dt>Heading</dt><dd>{selectedInfo.heading}°</dd></div>
					{#if selectedInfo.isEnemy}
						<div><dt>Role</dt><dd>Incoming raider</dd></div>
					{:else}
						<div><dt>View</dt><dd>{selectedInfo.view}px</dd></div>
						<div><dt>FOV</dt><dd>{selectedInfo.fov}°</dd></div>
					{/if}
					<div><dt>Time</dt><dd>{(frameIndex / (replay.fps || 1)).toFixed(1)}s</dd></div>
				</dl>
			</div>
		{/if}
	</div>
	<p class="replay-tip">Tip: pause, then click a ship to inspect it.</p>

	<div class="replay-controls">
		<button type="button" title="Restart" class="btn btn-sm btn-ghost" onclick={restart}>
			<Icon icon="ph:skip-back-fill" width="13" />
		</button>
		<button type="button" title="Step back" class="btn btn-sm btn-ghost" onclick={() => stepBy(-1)}>
			<Icon icon="ph:caret-left-bold" width="13" />
		</button>
		<button type="button" class="btn btn-sm" onclick={() => (playing = !playing)}>
			<Icon icon={playing ? 'ph:pause-fill' : 'ph:play-fill'} width="13" />
			{playing ? 'Pause' : 'Play'}
		</button>
		<button type="button" title="Step forward" class="btn btn-sm btn-ghost" onclick={() => stepBy(1)}>
			<Icon icon="ph:caret-right-bold" width="13" />
		</button>
		<input
			type="range"
			min="0"
			max={Math.max(0, replay.frames.length - 1)}
			bind:value={frameIndex}
			oninput={() => (playing = false)}
			class="replay-scrub"
		/>
		<span class="replay-counter">
			frame {frameIndex + 1}/{replay.frames.length} · {(frameIndex / (replay.fps || 1)).toFixed(1)}s
		</span>
	</div>

	<div class="replay-controls">
		<span class="replay-tip">Speed</span>
		{#each SPEEDS as s}
			<button
				type="button"
				class="btn btn-sm"
				class:btn-ghost={speed !== s}
				onclick={() => (speed = s)}
			>
				{s}x
			</button>
		{/each}
		<label class="replay-loop">
			<input type="checkbox" bind:checked={loop} />
			Loop
		</label>
	</div>
</div>

<style>
	.replay-stage {
		position: relative;
		line-height: 0;
	}

	.replay-canvas {
		display: block;
		width: 100%;
		background: var(--color-ink);
		border: 1px solid var(--color-line);
		border-radius: 4px;
		cursor: pointer;
	}

	.replay-tip {
		font-size: 0.72rem;
		color: var(--color-faint);
	}

	.replay-controls {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.replay-scrub {
		flex: 1;
		min-width: 4rem;
		accent-color: var(--color-brand);
		cursor: pointer;
	}

	.replay-counter {
		font-size: 0.72rem;
		color: var(--color-faint);
		white-space: nowrap;
	}

	.replay-loop {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		margin-left: 0.5rem;
		font-size: 0.72rem;
		color: var(--color-faint);
	}

	.ship-card {
		position: absolute;
		transform: translate(-50%, calc(-100% - 14px));
		min-width: 9.5rem;
		max-width: 14rem;
		background: var(--color-surface-raised);
		border: 1px solid var(--color-line-strong);
		border-radius: 4px;
		padding: 0.45rem 0.55rem;
		color: var(--color-heading);
		z-index: 5;
		line-height: 1.3;
	}

	.ship-card-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		margin-bottom: 0.35rem;
	}

	.ship-card-title {
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.03em;
	}

	.ship-card-title.defender {
		color: var(--color-brand-hover);
	}

	.ship-card-title.enemy {
		color: var(--color-loss);
	}

	.ship-card-close {
		border: none;
		background: transparent;
		color: var(--color-faint);
		font-size: 0.9rem;
		line-height: 1;
		cursor: pointer;
		padding: 0 0.15rem;
	}

	.ship-card-close:hover {
		color: var(--color-heading);
	}

	.ship-card-body {
		display: grid;
		grid-template-columns: 1fr;
		gap: 0.12rem;
		margin: 0;
	}

	.ship-card-body > div {
		display: flex;
		justify-content: space-between;
		gap: 0.75rem;
		font-size: 0.68rem;
	}

	.ship-card-body dt {
		color: var(--color-faint);
	}

	.ship-card-body dd {
		margin: 0;
		color: var(--color-heading);
		font-variant-numeric: tabular-nums;
	}
</style>
