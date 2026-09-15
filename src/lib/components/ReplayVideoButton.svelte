<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '@iconify/svelte';
	import { canExportVideo, encodeVideo, saveBlob, type VideoJob } from '$lib/ts/replayVideo';

	let { createJob, fileName }: { createJob: () => VideoJob; fileName: string } = $props();

	let supported = $state(true);
	let exporting = $state(false);
	let progress = $state(0);
	let failed = $state(false);

	const label = $derived.by(() => {
		if (exporting) return `Rendering ${Math.round(progress * 100)}%`;
		if (failed) return 'Export failed, retry';
		return 'Download MP4';
	});

	const tooltip = $derived(
		supported
			? 'Download this replay as an MP4 video'
			: 'This browser cannot encode MP4 video. Try a recent Chrome, Edge, Safari, or Firefox.'
	);

	onMount(async () => {
		const job = createJob();
		supported = await canExportVideo(job.width, job.height).catch(() => false);
	});

	async function downloadVideo() {
		if (exporting || !supported) return;
		const job = createJob();
		const exportedFileName = fileName;
		exporting = true;
		failed = false;
		progress = 0;
		try {
			const video = await encodeVideo(job, (fraction) => (progress = fraction));
			saveBlob(video, exportedFileName);
		} catch (error) {
			console.error('Replay video export failed', error);
			failed = true;
		} finally {
			exporting = false;
		}
	}
</script>

<button
	type="button"
	class="video-button"
	class:failed
	title={tooltip}
	disabled={exporting || !supported}
	onclick={downloadVideo}
>
	<Icon icon={exporting ? 'ph:spinner-gap-bold' : 'ph:download-simple-bold'} width="13" class={exporting ? 'spin' : ''} />
	{label}
</button>

<style>
	.video-button {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		height: 1.9rem;
		padding: 0 0.6rem;
		border: 1px solid var(--color-line);
		border-radius: 3px;
		background: var(--color-surface-raised);
		color: var(--color-dim);
		font-size: 0.7rem;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
		cursor: pointer;
		transition:
			color 0.15s,
			border-color 0.15s;
	}

	.video-button:hover:not(:disabled) {
		border-color: var(--color-line-strong);
		color: var(--color-heading);
	}

	.video-button:disabled {
		cursor: default;
		opacity: 0.6;
	}

	.video-button.failed {
		border-color: var(--color-loss);
		color: var(--color-loss);
	}

	.video-button :global(.spin) {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
