<script lang="ts">
	import { onDestroy } from 'svelte';
	import Icon from '@iconify/svelte';
	import ReplayVideoButton from '$lib/components/ReplayVideoButton.svelte';
	import type { Replay } from '$lib/ts/evaluation';
	import {
		STAGE_HEIGHT,
		STAGE_WIDTH,
		drawStage,
		sampleFrame,
		shipLabel,
		type ReplayMode
	} from '$lib/ts/replayRenderer';

	let { replay, mode = 'defense' }: { replay: Replay; mode?: ReplayMode } = $props();

	let canvas: HTMLCanvasElement | undefined = $state();
	let playing = $state(false);
	let frameIndex = $state(0);
	let speed = $state(1);
	let loop = $state(true);
	let selectedSlot = $state<number | null>(null);

	const SPEEDS = [0.5, 1, 2, 3];

	let raf = 0;
	let acc = 0;
	let last = 0;

	function draw(frame: number[]) {
		const ctx = canvas?.getContext('2d');
		if (!ctx) return;
		drawStage(ctx, replay, mode, frame, { frameIndex, selectedSlot });
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
		draw(sampleFrame(replay, frameIndex, Math.min(0.999, acc / step)));
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
		const frame = sampleFrame(replay, frameIndex, 0);
		const [aw, ah] = replay.arena;
		const sx = STAGE_WIDTH / aw;
		const sy = STAGE_HEIGHT / ah;
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
		const mx = ((event.clientX - rect.left) / rect.width) * STAGE_WIDTH;
		const my = ((event.clientY - rect.top) / rect.height) * STAGE_HEIGHT;
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
			label: shipLabel(replay, mode, s),
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
		// rewind to a paused first frame when the replay changes
		replay;
		frameIndex = 0;
		acc = 0;
		last = 0;
		playing = false;
		selectedSlot = null;
		cancelAnimationFrame(raf);
		raf = requestAnimationFrame(tick);
	});

	$effect(() => {
		// redraw immediately when scrubbing or selecting while paused
		selectedSlot;
		if (!playing) draw(sampleFrame(replay, frameIndex, 0));
	});

	onDestroy(() => cancelAnimationFrame(raf));
</script>

<div class="flex flex-col gap-2">
	<div class="replay-stage">
		<canvas
			bind:this={canvas}
			width={STAGE_WIDTH}
			height={STAGE_HEIGHT}
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

		<ReplayVideoButton {replay} {mode} />

		<span class="hint">Click the stage, then space to play, arrows to step</span>
	</div>

	{#if markers.length > 0}
		<div class="player-bar player-bar-legend">
			{#each markers as marker}
				<span class="legend-item">
					<span class="dot dot-{marker.kind}"></span>
					{marker.label}
					<span class="legend-time">{marker.time.toFixed(2)}s</span>
				</span>
			{/each}
			<span class="hint">Click a ship to inspect it</span>
		</div>
	{/if}
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

	.player-bar-legend {
		flex-wrap: wrap;
		gap: 0.35rem 1rem;
		font-size: 0.7rem;
		color: var(--color-faint);
	}

	.legend-item {
		display: inline-flex;
		flex: 0 0 auto;
		align-items: center;
		gap: 0.35rem;
		white-space: nowrap;
	}

	.legend-time {
		font-variant-numeric: tabular-nums;
		color: var(--color-dim);
	}

	.dot {
		width: 0.45rem;
		height: 0.45rem;
		border-radius: 50%;
	}

	.hint {
		margin-left: auto;
		flex: 0 0 auto;
		font-size: 0.7rem;
		white-space: nowrap;
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
