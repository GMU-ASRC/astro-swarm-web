<script lang="ts">
	import Icon from '@iconify/svelte';
	import type { ProgressSeries } from '$lib/ts/chartBase';
	import {
		buildGrid,
		divergingShade,
		formatValue,
		goodnessOf,
		ringTicks,
		type HeatmapKey
	} from '$lib/ts/heatmapScale';
	import { downloadHeatmapPng, heatmapFileName } from '$lib/ts/heatmapExport';

	let {
		series,
		key,
		title,
		downloadUrl = ''
	}: { series: ProgressSeries[]; key: HeatmapKey; title: string; downloadUrl?: string } = $props();

	const WIDTH = 720;
	const LEFT = 56;
	const RIGHT = 12;
	const TOP = 8;
	const BOTTOM = 46;
	const PLOT_TARGET_HEIGHT = 360;
	const GAP = 1;
	const MAX_FACED_LABELS = 20;

	const MISSING_FILL = '#f3f4f6';
	const LEGEND_STOPS = [0, 0.25, 0.5, 0.75, 1];

	const READINGS: Record<HeatmapKey, { label: string; ticks: string[]; reading: string }> = {
		capture_rate: {
			label: 'Capture rate',
			ticks: ['0%', '25%', '50%', '75%', '100%'],
			reading: 'red is a low capture rate, blue a high one'
		},
		risk: {
			label: 'Risk',
			ticks: ['100%', '75%', '50%', '25%', '0%'],
			reading: 'red is a high risk, blue a low one'
		},
		defenders: {
			label: 'Defenders remaining, as a share of ring size n',
			ticks: ['0%', '25%', '50%', '75%', '100%'],
			reading: 'red is a ring worn down, blue a ring still whole'
		}
	};

	let hovered = $state<{ row: number; column: number } | null>(null);
	let svgElement: SVGSVGElement | undefined = $state();

	let grid = $derived(buildGrid(series, key));
	let rows = $derived(grid.rings.length);
	let columns = $derived(grid.faced.length);
	let plotWidth = WIDTH - LEFT - RIGHT;
	let cellWidth = $derived(columns > 0 ? plotWidth / columns : plotWidth);
	let cellHeight = $derived(rows > 0 ? Math.max(4, Math.min(28, PLOT_TARGET_HEIGHT / rows)) : 0);
	let plotHeight = $derived(rows * cellHeight);
	let height = $derived(TOP + plotHeight + BOTTOM);
	let rowTicks = $derived(ringTicks(grid.rings));
	let facedStep = $derived(Math.max(1, Math.ceil(columns / MAX_FACED_LABELS)));
	let reading = $derived(READINGS[key]);
	let hoveredValue = $derived(hovered ? grid.values[hovered.row][hovered.column] : null);
	let hasMissingCells = $derived(grid.values.some((rowValues) => rowValues.includes(null)));
	let legendColors = $derived(LEGEND_STOPS.map((stop) => divergingShade(stop)));
	let legendGradient = $derived(legendColors.join(', '));
	let subtitle = $derived(`Rows are ring size n, columns are evaders faced. ${reading.reading}.`);
	let downloadFailed = $state(false);

	async function downloadHeatmap() {
		if (!svgElement) return;
		downloadFailed = false;
		const legend = {
			label: reading.label,
			ticks: reading.ticks,
			stops: LEGEND_STOPS,
			colors: legendColors,
			missingColor: hasMissingCells ? MISSING_FILL : null
		};
		try {
			await downloadHeatmapPng({ plot: svgElement, title, subtitle, legend }, heatmapFileName(title));
		} catch {
			downloadFailed = true;
		}
	}

	function cellX(column: number): number {
		return LEFT + column * cellWidth;
	}

	function cellY(row: number): number {
		return TOP + (rows - 1 - row) * cellHeight;
	}

	function fillFor(row: number, value: number | null): string {
		if (value === null) return MISSING_FILL;
		return divergingShade(goodnessOf(key, value, grid.rings[row]));
	}

	function onPointerMove(event: PointerEvent) {
		if (!svgElement) return;
		const box = svgElement.getBoundingClientRect();
		const scale = WIDTH / box.width;
		const x = (event.clientX - box.left) * scale;
		const y = (event.clientY - box.top) * scale;
		const column = Math.floor((x - LEFT) / cellWidth);
		const row = rows - 1 - Math.floor((y - TOP) / cellHeight);
		hovered = column >= 0 && column < columns && row >= 0 && row < rows ? { row, column } : null;
	}

	let tooltipStyle = $derived.by(() => {
		if (!hovered || !svgElement) return '';
		const box = svgElement.getBoundingClientRect();
		const scale = box.width / WIDTH;
		const left = (cellX(hovered.column) + cellWidth / 2) * scale;
		const top = cellY(hovered.row) * scale;
		return `left: ${Math.max(80, Math.min(box.width - 80, left))}px; top: ${top}px;`;
	});
</script>

<div class="heatmap-card">
	<div class="heatmap-plot">
		<div class="heatmap-head">
			<h3>{title}</h3>
			<p>{subtitle}</p>
		</div>

		<div class="heatmap-frame">
			<svg
				bind:this={svgElement}
				viewBox="0 0 {WIDTH} {height}"
				role="img"
				aria-label="{title} by ring size and evaders faced"
				onpointermove={onPointerMove}
				onpointerleave={() => (hovered = null)}
			>
				{#each grid.values as rowValues, row (row)}
					{#each rowValues as value, column (column)}
						<rect
							x={cellX(column) + GAP / 2}
							y={cellY(row)}
							width={Math.max(0.5, cellWidth - GAP)}
							height={cellHeight}
							fill={fillFor(row, value)}
						/>
					{/each}
				{/each}

				{#each rowTicks as row (row)}
					<text class="axis-text" x={LEFT - 8} y={cellY(row) + cellHeight / 2 + 3.5} text-anchor="end">{grid.rings[row]}</text>
				{/each}
				{#each grid.faced as faced, column (column)}
					{#if column % facedStep === 0}
						<text class="axis-text" x={cellX(column) + cellWidth / 2} y={TOP + plotHeight + 16} text-anchor="middle">{faced}</text>
					{/if}
				{/each}
				<text class="axis-title" x={LEFT + plotWidth / 2} y={height - 8} text-anchor="middle">Evaders faced</text>
				<text class="axis-title" x="14" y={TOP + plotHeight / 2} text-anchor="middle" transform="rotate(-90 14 {TOP + plotHeight / 2})">Ring size (n)</text>

				{#if hovered}
					<rect class="crosshair" x={cellX(hovered.column)} y={cellY(hovered.row) - 1} width={cellWidth} height={cellHeight + 2} />
				{/if}
			</svg>

			{#if hovered}
				<div class="heatmap-tooltip" style={tooltipStyle}>
					n = <strong>{grid.rings[hovered.row]}</strong> · evader <strong>{grid.faced[hovered.column]}</strong><br />
					{#if hoveredValue === null}
						no data
					{:else}
						<strong>{formatValue(key, hoveredValue)}</strong>
					{/if}
				</div>
			{/if}
		</div>

		<div class="heatmap-legend">
			<div class="legend-scale">
				<span class="legend-title">{reading.label}</span>
				<div class="legend-bar" style="background: linear-gradient(to right, {legendGradient});"></div>
				<div class="legend-ticks">
					{#each reading.ticks as tick, index (index)}
						<span style="left: {LEGEND_STOPS[index] * 100}%;">{tick}</span>
					{/each}
				</div>
			</div>
			{#if hasMissingCells}
				<div class="legend-missing">
					<span class="legend-swatch" style="background: {MISSING_FILL};"></span>
					No data
				</div>
			{/if}
		</div>
	</div>

	<div class="heatmap-actions">
		<button type="button" class="btn btn-sm btn-ghost" onclick={downloadHeatmap}>
			<Icon icon="ph:download-simple-bold" width="14" />
			Download heatmap
		</button>
		{#if downloadUrl}
			<a class="btn btn-sm btn-ghost" href={downloadUrl}>
				<Icon icon="ph:download-simple-bold" width="14" />
				Download line graph
			</a>
		{/if}
		{#if downloadFailed}
			<span class="download-error">The heatmap could not be downloaded.</span>
		{/if}
		<details class="heatmap-table">
			<summary>Data table</summary>
			<div class="table-scroll">
				<table>
					<thead>
						<tr>
							<th>n</th>
							{#each grid.faced as faced (faced)}
								<th>{faced}</th>
							{/each}
						</tr>
					</thead>
					<tbody>
						{#each grid.values as rowValues, row (row)}
							<tr>
								<th>{grid.rings[row]}</th>
								{#each rowValues as value, column (column)}
									<td>{value === null ? '' : formatValue(key, value)}</td>
								{/each}
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</details>
	</div>
</div>

<style>
	.heatmap-card {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		min-width: 0;
	}

	.heatmap-plot {
		padding: 1.2rem 1.4rem 1rem;
		background: #ffffff;
		color: #374151;
		border: 1px solid var(--color-line);
		border-radius: var(--radius-panel);
		display: grid;
		gap: 0.75rem;
	}

	.heatmap-head h3 {
		margin: 0;
		font-size: 0.95rem;
		font-weight: 700;
		color: #374151;
		text-align: center;
	}

	.heatmap-head p {
		margin: 0.2rem 0 0;
		font-size: 0.78rem;
		color: #6b7280;
		text-align: center;
	}

	.heatmap-frame {
		position: relative;
		min-width: 0;
	}

	svg {
		display: block;
		width: 100%;
		height: auto;
	}

	.axis-text {
		fill: #374151;
		font-size: 11px;
	}

	.axis-title {
		fill: #374151;
		font-size: 12px;
		font-weight: 600;
	}

	.crosshair {
		fill: none;
		stroke: #111827;
		stroke-width: 1.5;
		pointer-events: none;
	}

	.heatmap-tooltip {
		position: absolute;
		pointer-events: none;
		background: #111827;
		color: #f9fafb;
		font-size: 0.78rem;
		line-height: 1.4;
		padding: 6px 10px;
		border-radius: 6px;
		white-space: nowrap;
		transform: translate(-50%, calc(-100% - 8px));
	}

	.heatmap-legend {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		justify-content: center;
		gap: 0.5rem 1.5rem;
	}

	.legend-scale {
		display: grid;
		gap: 3px;
		width: min(300px, 100%);
	}

	.legend-title {
		font-size: 0.72rem;
		font-weight: 600;
		color: #374151;
		text-align: center;
	}

	.legend-bar {
		height: 10px;
		border-radius: 3px;
		border: 1px solid #e5e7eb;
	}

	.legend-ticks {
		position: relative;
		height: 1.1em;
		font-size: 0.72rem;
		color: #6b7280;
	}

	.legend-ticks span {
		position: absolute;
		top: 0;
		transform: translateX(-50%);
		white-space: nowrap;
	}

	.legend-ticks span:first-child {
		transform: none;
	}

	.legend-ticks span:last-child {
		transform: translateX(-100%);
	}

	.legend-missing {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.72rem;
		color: #6b7280;
	}

	.legend-swatch {
		width: 12px;
		height: 12px;
		border-radius: 2px;
		border: 1px solid #d1d5db;
	}

	.heatmap-actions {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-start;
		gap: 0.6rem;
	}

	.download-error {
		align-self: center;
		font-size: 0.78rem;
		color: #b91c1c;
	}

	.heatmap-table {
		flex: 1 1 100%;
		font-size: 0.8rem;
	}

	.heatmap-table summary {
		cursor: pointer;
	}

	.table-scroll {
		overflow: auto;
		max-height: 360px;
		margin-top: 0.5rem;
		border: 1px solid var(--color-line);
		border-radius: var(--radius-panel);
	}

	table {
		border-collapse: collapse;
		font-variant-numeric: tabular-nums;
		font-size: 0.75rem;
	}

	th,
	td {
		padding: 3px 8px;
		text-align: right;
		border-bottom: 1px solid var(--color-line);
		white-space: nowrap;
	}

	thead th {
		position: sticky;
		top: 0;
		background: var(--color-ink);
	}
</style>
