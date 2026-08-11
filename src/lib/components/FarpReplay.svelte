<script lang="ts">
	import { onDestroy } from 'svelte';
	import Icon from '@iconify/svelte';
	import type { Replay } from '$lib/ts/evaluation';

	let { replay, mode = 'defence' }: { replay: Replay; mode?: 'defence' | 'swarm' } = $props();

	// The ship icons are drawn far larger than the ships really are, so the hull
	// each one actually collides with is drawn underneath at true scale.
	const DEFAULT_HULL = 9;

	const SWARM_A = '#ff6a52';
	const SWARM_B = '#7c9eff';
	const LEADER = '#ffd54a';

	// In swarm mode the recorded ships are two milling groups followed by the
	// player-flown leader, so they are coloured by group instead of by team.
	function shipColor(slot: number): string {
		if (mode !== 'swarm') return slot === replay.defenders ? '#ff6a52' : '#7c9eff';
		if (slot >= replay.defenders) return LEADER;
		return slot < replay.defenders / 2 ? SWARM_A : SWARM_B;
	}

	function shipLabel(slot: number): string {
		if (mode !== 'swarm') return slot === replay.defenders ? 'Enemy raider' : `Defender #${slot + 1}`;
		if (slot >= replay.defenders) return 'Leader';
		const group = slot < replay.defenders / 2 ? 'A' : 'B';
		return `Group ${group} agent #${slot + 1}`;
	}

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
			const cone = mode === 'swarm' ? coneColor(shipColor(s)) : 'rgba(124,158,255,0.16)';
			drawCone(ctx, x * sx, y * sy, rot, coneRadius, half, cone);
		}

		if (mode === 'swarm') drawSwarmMarkers(ctx, frame, sx, sy);

		const hullRadius = (replay.hull ?? DEFAULT_HULL) * sx;
		for (let s = 0; s < total; s++) {
			const x = frame[s * 3];
			const y = frame[s * 3 + 1];
			const rot = (frame[s * 3 + 2] * Math.PI) / 180;
			if (x < 0) continue;
			drawHull(ctx, x * sx, y * sy, hullRadius, shipColor(s));
			drawShip(ctx, x * sx, y * sy, rot, shipColor(s));
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
		const count = mode === 'swarm' ? `agents=${replay.defenders}` : `N=${replay.defenders}`;
		ctx.fillText(
			`${count}  ${replay.outcome}  frame ${frameIndex + 1}/${replay.frames.length}  t=${seconds}s`,
			8,
			16
		);
	}

	function coneColor(color: string): string {
		return `${color}22`;
	}

	function groupCenter(frame: number[], from: number, to: number): [number, number] | null {
		let sumX = 0;
		let sumY = 0;
		let count = 0;
		for (let s = from; s < to; s++) {
			if (frame[s * 3] < 0) continue;
			sumX += frame[s * 3];
			sumY += frame[s * 3 + 1];
			count++;
		}
		return count === 0 ? null : [sumX / count, sumY / count];
	}

	function isMerged(frame: number[]): boolean {
		const limit = replay.stats?.merge_distance;
		if (!limit || replay.defenders < 2) return false;
		const reached = new Set<number>([0]);
		const queue = [0];
		while (queue.length > 0) {
			const current = queue.pop() as number;
			for (let s = 0; s < replay.defenders; s++) {
				if (reached.has(s)) continue;
				const dx = frame[current * 3] - frame[s * 3];
				const dy = frame[current * 3 + 1] - frame[s * 3 + 1];
				if (Math.hypot(dx, dy) <= limit) {
					reached.add(s);
					queue.push(s);
				}
			}
		}
		return reached.size === replay.defenders;
	}

	function drawSwarmMarkers(ctx: CanvasRenderingContext2D, frame: number[], sx: number, sy: number) {
		const half = Math.floor(replay.defenders / 2);
		const goalRadius = replay.stats?.goal_radius;
		if (goalRadius) {
			const [px, py] = replay.planet;
			ctx.beginPath();
			ctx.arc(px * sx, py * sy, goalRadius * sx, 0, Math.PI * 2);
			ctx.strokeStyle = 'rgba(255,255,255,0.25)';
			ctx.setLineDash([4, 4]);
			ctx.stroke();
			ctx.setLineDash([]);
		}
		if (isMerged(frame)) {
			drawMarker(ctx, groupCenter(frame, 0, replay.defenders), sx, sy, LEADER);
			return;
		}
		drawMarker(ctx, groupCenter(frame, 0, half), sx, sy, SWARM_A);
		drawMarker(ctx, groupCenter(frame, half, replay.defenders), sx, sy, SWARM_B);
	}

	function drawMarker(
		ctx: CanvasRenderingContext2D,
		center: [number, number] | null,
		sx: number,
		sy: number,
		color: string
	) {
		if (!center) return;
		const x = center[0] * sx;
		const y = center[1] * sy;
		const radius = 6;
		const arm = radius + 4;
		ctx.strokeStyle = color;
		ctx.lineWidth = 1.5;
		ctx.beginPath();
		ctx.arc(x, y, radius, 0, Math.PI * 2);
		ctx.stroke();
		ctx.beginPath();
		ctx.moveTo(x - arm, y);
		ctx.lineTo(x + arm, y);
		ctx.moveTo(x, y - arm);
		ctx.lineTo(x, y + arm);
		ctx.stroke();
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

	// Two of these circles touching is exactly what the simulator counts as a
	// capture, which the oversized icons alone never showed.
	function drawHull(ctx: CanvasRenderingContext2D, x: number, y: number, radius: number, color: string) {
		ctx.beginPath();
		ctx.arc(x, y, Math.max(radius, 1), 0, Math.PI * 2);
		ctx.strokeStyle = `${color}88`;
		ctx.lineWidth = 1;
		ctx.stroke();
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

	const lastFrame = $derived(Math.max(0, replay.frames.length - 1));
	const elapsed = $derived(`${(frameIndex / (replay.fps || 1)).toFixed(2)}s`);
	const duration = $derived(`${(lastFrame / (replay.fps || 1)).toFixed(2)}s`);

	// The run's own events, pinned on the scrub bar so a near miss is easy to find.
	const markers = $derived.by(() => {
		const events = [
			{ kind: 'detect', label: 'Detected', time: replay.detection_time },
			{ kind: 'capture', label: 'Captured', time: replay.capture_time },
			{ kind: 'goal', label: 'Reached planet', time: replay.goal_time }
		];
		const span = lastFrame / (replay.fps || 1);
		if (span <= 0) return [];
		return events
			.filter((event) => event.time != null && event.time >= 0)
			.map((event) => ({
				...event,
				time: event.time as number,
				percent: Math.min(100, Math.max(0, ((event.time as number) / span) * 100))
			}));
	});

	function togglePlay() {
		playing = !playing;
	}

	function onKeyDown(event: KeyboardEvent) {
		const jump = event.shiftKey ? 10 : 1;
		if (event.key === ' ') {
			togglePlay();
		} else if (event.key === 'ArrowLeft') {
			stepBy(-jump);
		} else if (event.key === 'ArrowRight') {
			stepBy(jump);
		} else {
			return;
		}
		event.preventDefault();
	}

	function restart() {
		frameIndex = 0;
		acc = 0;
		last = 0;
		playing = true;
	}

	function stepBy(delta: number) {
		playing = false;
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
			label: shipLabel(s),
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
			onkeydown={onKeyDown}
			tabindex="0"
			role="application"
			aria-label="Replay stage"
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
						<div><dt>Role</dt><dd>{mode === 'swarm' ? 'Player-flown leader' : 'Incoming raider'}</dd></div>
					{:else}
						<div><dt>View</dt><dd>{selectedInfo.view}px</dd></div>
						<div><dt>FOV</dt><dd>{selectedInfo.fov}°</dd></div>
					{/if}
					<div><dt>Time</dt><dd>{(frameIndex / (replay.fps || 1)).toFixed(1)}s</dd></div>
				</dl>
			</div>
		{/if}
	</div>
	<div class="player-bar">
		<div class="transport">
			<button type="button" title="Restart" class="ctrl" onclick={restart}>
				<Icon icon="ph:skip-back-fill" width="13" />
			</button>
			<button type="button" title="Step back (left arrow)" class="ctrl" onclick={() => stepBy(-1)}>
				<Icon icon="ph:caret-left-bold" width="13" />
			</button>
			<button
				type="button"
				title={playing ? 'Pause (space)' : 'Play (space)'}
				class="ctrl ctrl-play"
				onclick={togglePlay}
			>
				<Icon icon={playing ? 'ph:pause-fill' : 'ph:play-fill'} width="15" />
			</button>
			<button type="button" title="Step forward (right arrow)" class="ctrl" onclick={() => stepBy(1)}>
				<Icon icon="ph:caret-right-bold" width="13" />
			</button>
		</div>

		<div class="track">
			<input
				type="range"
				min="0"
				max={lastFrame}
				bind:value={frameIndex}
				oninput={() => (playing = false)}
				aria-label="Scrub"
				class="scrub"
			/>
			{#each markers as marker}
				<span
					class="marker marker-{marker.kind}"
					style="left:{marker.percent}%"
					title={`${marker.label} at ${marker.time.toFixed(2)}s`}
				></span>
			{/each}
		</div>

		<span class="clock">{elapsed} / {duration}</span>
	</div>

	<div class="player-bar player-bar-sub">
		<div class="speeds">
			{#each SPEEDS as option}
				<button
					type="button"
					class="ctrl ctrl-speed"
					class:active={speed === option}
					onclick={() => (speed = option)}
				>
					{option}x
				</button>
			{/each}
		</div>

		<label class="toggle">
			<input type="checkbox" bind:checked={loop} />
			Loop
		</label>

		{#if markers.length > 0}
			<div class="legend">
				{#each markers as marker}
					<span class="legend-item"><span class="dot dot-{marker.kind}"></span>{marker.label}</span>
				{/each}
			</div>
		{/if}

		<span class="hint">Click the stage, then space to play, arrows to step. Click a ship to inspect it.</span>
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

	.replay-canvas:focus-visible {
		outline: 2px solid var(--color-brand);
		outline-offset: 2px;
	}

	.player-bar {
		display: flex;
		align-items: center;
		gap: 0.6rem;
	}

	.player-bar-sub {
		flex-wrap: wrap;
		gap: 0.5rem 0.9rem;
	}

	.transport {
		display: flex;
		align-items: center;
		gap: 0.2rem;
	}

	.ctrl {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 1.9rem;
		height: 1.9rem;
		padding: 0 0.4rem;
		border: 1px solid var(--color-line);
		border-radius: 3px;
		background: var(--color-surface-raised);
		color: var(--color-dim);
		font-size: 0.7rem;
		cursor: pointer;
		transition:
			color 0.15s,
			border-color 0.15s;
	}

	.ctrl:hover {
		border-color: var(--color-line-strong);
		color: var(--color-heading);
	}

	.ctrl-play {
		min-width: 2.3rem;
		color: var(--color-heading);
		border-color: var(--color-line-strong);
	}

	.ctrl-speed.active {
		border-color: var(--color-brand);
		color: var(--color-heading);
	}

	.track {
		position: relative;
		flex: 1;
		min-width: 6rem;
		display: flex;
		align-items: center;
	}

	.scrub {
		width: 100%;
		accent-color: var(--color-brand);
		cursor: pointer;
	}

	/* Event pins sit under the thumb, so a detection or capture is easy to scrub to. */
	.marker {
		position: absolute;
		bottom: 0;
		width: 2px;
		height: 0.55rem;
		transform: translateX(-1px);
		pointer-events: none;
	}

	.marker-detect,
	.dot-detect {
		background: var(--color-warn);
	}

	.marker-capture,
	.dot-capture {
		background: var(--color-win);
	}

	.marker-goal,
	.dot-goal {
		background: var(--color-loss);
	}

	.clock {
		font-size: 0.72rem;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
		color: var(--color-faint);
	}

	.speeds {
		display: flex;
		gap: 0.2rem;
	}

	.toggle {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.72rem;
		color: var(--color-faint);
	}

	.legend {
		display: flex;
		gap: 0.7rem;
		font-size: 0.7rem;
		color: var(--color-faint);
	}

	.legend-item {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
	}

	.dot {
		width: 0.45rem;
		height: 0.45rem;
		border-radius: 50%;
	}

	.hint {
		margin-left: auto;
		font-size: 0.7rem;
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
