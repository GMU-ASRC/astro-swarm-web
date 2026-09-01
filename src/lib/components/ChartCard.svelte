<script lang="ts">
	import { onDestroy } from 'svelte';
	import Icon from '@iconify/svelte';
	import Chart from 'chart.js/auto';
	import type { ChartConfiguration } from 'chart.js';

	let { config, downloadUrl = '' }: { config: ChartConfiguration; downloadUrl?: string } = $props();

	let canvas: HTMLCanvasElement | undefined = $state();
	let chart: Chart | undefined;

	$effect(() => {
		if (!canvas) return;
		chart?.destroy();
		chart = new Chart(canvas, config);
	});

	onDestroy(() => chart?.destroy());
</script>

<div class="chart-card">
	<div class="chart-plot"><canvas bind:this={canvas}></canvas></div>
	{#if downloadUrl}
		<a class="btn btn-sm btn-ghost chart-download" href={downloadUrl}>
			<Icon icon="ph:download-simple-bold" width="14" />
			Download PNG
		</a>
	{/if}
</div>

<style>
	.chart-card {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}

	.chart-plot {
		height: 320px;
		padding: 1.4rem;
		background: #ffffff;
		border: 1px solid var(--color-line);
		border-radius: var(--radius-panel);
	}

	.chart-download {
		align-self: flex-start;
	}
</style>
