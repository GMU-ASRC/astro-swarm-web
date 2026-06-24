<script lang="ts">
	import type { EvalPoint } from '$lib/ts/evaluation';

	let {
		points = [],
		title = '',
		color = '#2a6fb0',
		yLabel = 'Success Rate (%)',
		xLabel = 'n',
		height = 380
	}: {
		points?: EvalPoint[];
		title?: string;
		color?: string;
		yLabel?: string;
		xLabel?: string;
		height?: number;
	} = $props();

	const width = 580;
	const pad = { top: 44, right: 20, bottom: 48, left: 56 };

	let plotW = $derived(width - pad.left - pad.right);
	let plotH = $derived(height - pad.top - pad.bottom);
	let count = $derived(Math.max(1, points.length));
	let slot = $derived(plotW / count);
	let barW = $derived(slot * 0.7);

	const yTicks = [0, 20, 40, 60, 80, 100];

	function y(v: number): number {
		return pad.top + (1 - v / 100) * plotH;
	}
	function barX(i: number): number {
		return pad.left + i * slot + (slot - barW) / 2;
	}
	let labelStep = $derived(count <= 30 ? 1 : Math.ceil(count / 30));
</script>

<figure class="w-full overflow-x-auto">
	<svg viewBox="0 0 {width} {height}" class="w-full max-w-[640px]" role="img" aria-label={title}>
		<rect x="0" y="0" {width} {height} fill="#ffffff" rx="6" />

		{#if title}
			<text x={width / 2} y="26" text-anchor="middle" fill="#111827" font-size="16" font-weight="600">{title}</text>
		{/if}

		{#each yTicks as t}
			<line x1={pad.left} y1={y(t)} x2={width - pad.right} y2={y(t)} stroke="#e5e7eb" stroke-width="1" />
			<text x={pad.left - 8} y={y(t) + 4} text-anchor="end" fill="#6b7280" font-size="11">{t}</text>
		{/each}

		{#each points as p, i}
			<rect x={barX(i)} y={y(p.success_rate)} width={barW} height={Math.max(0, height - pad.bottom - y(p.success_rate))} fill={color} />
			{#if i % labelStep === 0}
				<text x={barX(i) + barW / 2} y={height - pad.bottom + 16} text-anchor="middle" fill="#6b7280" font-size="9">{p.n}</text>
			{/if}
		{/each}

		<line x1={pad.left} y1={pad.top} x2={pad.left} y2={height - pad.bottom} stroke="#9ca3af" stroke-width="1" />
		<line x1={pad.left} y1={height - pad.bottom} x2={width - pad.right} y2={height - pad.bottom} stroke="#9ca3af" stroke-width="1" />

		<text x={16} y={height / 2} text-anchor="middle" fill="#374151" font-size="12" transform="rotate(-90 16 {height / 2})">{yLabel}</text>
		<text x={width / 2} y={height - 6} text-anchor="middle" fill="#374151" font-size="12">{xLabel}</text>
	</svg>
</figure>
