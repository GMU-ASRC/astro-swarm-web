<script lang="ts">
	import { onDestroy } from 'svelte';
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
		<a class="chart-download" href={downloadUrl}>Download PNG</a>
	{/if}
</div>

<style>
	.chart-card {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.chart-plot {
		background: #ffffff;
		border: 1px solid #d8e2f5;
		padding: 0.75rem;
		height: 300px;
	}

	.chart-download {
		align-self: flex-start;
		font-size: 0.82rem;
		padding: 0.35rem 0.8rem;
		border: 1px solid #b9caec;
		background: #eaf0fd;
		color: #2c3e6b;
		text-decoration: none;
		cursor: pointer;
	}

	.chart-download:hover {
		background: #dbe6fc;
		border-color: #8fabe6;
		text-decoration: none;
	}
</style>
