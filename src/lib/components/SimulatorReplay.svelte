<script lang="ts">
	import { onDestroy } from 'svelte';
	import Icon from '@iconify/svelte';
	import ReplayVideoButton from '$lib/components/ReplayVideoButton.svelte';
	import type { SimulatorEntry, SimulatorReplay } from '$lib/ts/simulator';
	import { durationLabel } from '$lib/ts/simulator';
	import {
		createScene,
		drawScene,
		replayDuration,
		sampleRobots,
		speciesCounts
	} from '$lib/ts/simulatorRenderer';
	import { VIDEO_FPS, evenSize, videoFrameCount, type VideoJob } from '$lib/ts/replayVideo';

	let { entry, replay }: { entry: SimulatorEntry; replay: SimulatorReplay } = $props();

	const MAX_STAGE_WIDTH = 960;
	const MAX_STAGE_HEIGHT = 720;
	const VIDEO_SCALE = 2;
	const SPEEDS = [0.5, 1, 2, 4];

	let canvas: HTMLCanvasElement | undefined = $state();
	let seconds = $state(0);
	let playing = $state(false);
	let speed = $state(1);
	let loop = $state(false);
	let showVisionCones = $state(true);

	let animationFrame = 0;
	let lastTimestamp = 0;

	const scene = $derived(createScene(entry, replay));
	const duration = $derived(replayDuration(scene));
	const stageScale = $derived(Math.min(MAX_STAGE_WIDTH / entry.arena_width, MAX_STAGE_HEIGHT / entry.arena_height));
	const stageWidth = $derived(evenSize(entry.arena_width * stageScale));
	const stageHeight = $derived(evenSize(entry.arena_height * stageScale));
	const robots = $derived(sampleRobots(scene, seconds));
	const counts = $derived(speciesCounts(scene, robots));

	$effect(() => {
		if (!canvas) return;
		const context = canvas.getContext('2d');
		if (!context) return;
		drawScene(context, scene, robots, stageWidth, stageHeight, seconds, { showVisionCones });
	});

	function tick(timestamp: number) {
		if (lastTimestamp === 0) lastTimestamp = timestamp;
		const elapsed = ((timestamp - lastTimestamp) / 1000) * speed;
		lastTimestamp = timestamp;
		const nextSeconds = seconds + elapsed;
		if (nextSeconds >= duration) {
			if (loop && duration > 0) {
				seconds = nextSeconds % duration;
			} else {
				seconds = duration;
				stop();
				return;
			}
		} else {
			seconds = nextSeconds;
		}
		animationFrame = requestAnimationFrame(tick);
	}

	function play() {
		if (playing) return;
		if (seconds >= duration) seconds = 0;
		playing = true;
		lastTimestamp = 0;
		animationFrame = requestAnimationFrame(tick);
	}

	function stop() {
		playing = false;
		cancelAnimationFrame(animationFrame);
	}

	function togglePlay() {
		if (playing) stop();
		else play();
	}

	function restart() {
		stop();
		seconds = 0;
		play();
	}

	function stepBy(frames: number) {
		stop();
		seconds = Math.min(duration, Math.max(0, seconds + frames * replay.interval));
	}

	function onKeyDown(event: KeyboardEvent) {
		const frames = event.shiftKey ? 10 : 1;
		if (event.key === ' ') togglePlay();
		else if (event.key === 'ArrowLeft') stepBy(-frames);
		else if (event.key === 'ArrowRight') stepBy(frames);
		else return;
		event.preventDefault();
	}

	function createVideoJob(): VideoJob {
		const width = stageWidth * VIDEO_SCALE;
		const height = stageHeight * VIDEO_SCALE;
		const videoScene = scene;
		const videoOptions = { showVisionCones };
		return {
			width,
			height,
			frameCount: videoFrameCount(duration),
			drawFrame: (context, videoFrame) => {
				const time = Math.min(duration, videoFrame / VIDEO_FPS);
				drawScene(context, videoScene, sampleRobots(videoScene, time), width, height, time, videoOptions);
			}
		};
	}

	function videoFileName(): string {
		const slug = entry.title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
		return `simulator-${slug || entry.id.slice(0, 8)}.mp4`;
	}

	onDestroy(stop);
</script>

<div class="player">
	<canvas
		bind:this={canvas}
		width={stageWidth}
		height={stageHeight}
		class="stage"
		tabindex="0"
		role="application"
		aria-label="Simulator replay"
		onkeydown={onKeyDown}
	></canvas>

	<div class="bar">
		<div class="transport">
			<button type="button" class="control" title="Restart" onclick={restart}>
				<Icon icon="ph:skip-back-fill" width="13" />
			</button>
			<button type="button" class="control" title="Step back (left arrow)" onclick={() => stepBy(-1)}>
				<Icon icon="ph:caret-left-bold" width="13" />
			</button>
			<button type="button" class="control control-play" title={playing ? 'Pause (space)' : 'Play (space)'} onclick={togglePlay}>
				<Icon icon={playing ? 'ph:pause-fill' : 'ph:play-fill'} width="15" />
			</button>
			<button type="button" class="control" title="Step forward (right arrow)" onclick={() => stepBy(1)}>
				<Icon icon="ph:caret-right-bold" width="13" />
			</button>
		</div>
		<input
			type="range"
			class="scrub"
			min="0"
			max={duration}
			step={replay.interval}
			bind:value={seconds}
			oninput={stop}
			aria-label="Scrub"
		/>
		<span class="clock">{durationLabel(seconds)} / {durationLabel(duration)}</span>
	</div>

	<div class="bar bar-wrap">
		<div class="speeds">
			{#each SPEEDS as option}
				<button type="button" class="control" class:active={speed === option} onclick={() => (speed = option)}>
					{option}x
				</button>
			{/each}
		</div>
		<label class="toggle">
			<input type="checkbox" bind:checked={loop} />
			Loop
		</label>
		<label class="toggle">
			<input type="checkbox" bind:checked={showVisionCones} />
			Vision cones
		</label>
		<ReplayVideoButton createJob={createVideoJob} fileName={videoFileName()} />
		<span class="hint">Click the stage, then space to play, arrows to step</span>
	</div>

	<div class="legend">
		{#each entry.species as species}
			<span class="legend-item">
				<span class="swatch" style={`background:${species.color}`}></span>
				{species.name}
				<span class="legend-count">{counts.get(species.id) ?? 0}</span>
			</span>
		{/each}
	</div>
</div>

<style>
	.player {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}

	.stage {
		display: block;
		width: 100%;
		height: auto;
		border: 1px solid var(--color-line);
		border-radius: 4px;
		background: #d5d5db;
	}

	.stage:focus-visible {
		outline: 2px solid var(--color-brand);
		outline-offset: 2px;
	}

	.bar {
		display: flex;
		align-items: center;
		gap: 0.6rem;
	}

	.bar-wrap {
		flex-wrap: wrap;
		gap: 0.5rem 0.9rem;
	}

	.transport,
	.speeds {
		display: flex;
		gap: 0.2rem;
	}

	.control {
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
	}

	.control:hover {
		border-color: var(--color-line-strong);
		color: var(--color-heading);
	}

	.control-play {
		min-width: 2.3rem;
		color: var(--color-heading);
		border-color: var(--color-line-strong);
	}

	.control.active {
		border-color: var(--color-brand);
		color: var(--color-heading);
	}

	.scrub {
		flex: 1;
		min-width: 6rem;
		accent-color: var(--color-brand);
	}

	.clock {
		font-size: 0.72rem;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
		color: var(--color-faint);
	}

	.toggle {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.72rem;
		color: var(--color-faint);
	}

	.hint {
		margin-left: auto;
		font-size: 0.7rem;
		color: var(--color-faint);
	}

	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem 1.1rem;
		font-size: 0.75rem;
		color: var(--color-dim);
	}

	.legend-item {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}

	.swatch {
		width: 0.7rem;
		height: 0.7rem;
		border-radius: 50%;
	}

	.legend-count {
		font-variant-numeric: tabular-nums;
		color: var(--color-faint);
	}
</style>
