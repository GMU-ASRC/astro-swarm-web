<script lang="ts">
	import type { EvalPoint } from '$lib/ts/evaluation';

	interface Series {
		label: string;
		color: string;
		points: EvalPoint[];
	}

	let {
		series = [],
		title = '',
		subtitle = '',
		yLabel = 'Success Rate (%)',
		xLabel = 'Number of Agents',
		height = 380
	}: {
		series?: Series[];
		title?: string;
		subtitle?: string;
		yLabel?: string;
		xLabel?: string;
		height?: number;
	} = $props();

	const width = 580;
	const pad = { top: 54, right: 24, bottom: 48, left: 56 };

	let maxN = $derived(Math.max(1, ...series.flatMap((s) => s.points.map((p) => p.n))));
	let plotW = $derived(width - pad.left - pad.right);
	let plotH = $derived(height - pad.top - pad.bottom);

	const yTicks = [0, 20, 40, 60, 80, 100];

	function x(n: number): number {
		return pad.left + (maxN <= 1 ? 0 : (n - 1) / (maxN - 1)) * plotW;
	}
	function y(v: number): number {
		return pad.top + (1 - v / 100) * plotH;
	}
	function linePath(points: EvalPoint[]): string {
		return points
			.map((p, i) => `${i === 0 ? 'M' : 'L'} ${x(p.n).toFixed(1)} ${y(p.success_rate).toFixed(1)}`)
			.join(' ');
	}

	function buildXTicks(m: number): number[] {
		const step = m <= 10 ? 1 : m <= 50 ? 10 : 20;
		const ticks: number[] = [];
		for (let i = 1; i <= m; i += step) ticks.push(i);
		if (ticks[ticks.length - 1] !== m) ticks.push(m);
		return ticks;
	}
	let xTicks = $derived(buildXTicks(maxN));
</script>

<figure class="w-full overflow-x-auto">
	<svg viewBox="0 0 {width} {height}" class="w-full max-w-[640px]" role="img" aria-label={title}>
		<rect x="0" y="0" {width} {height} fill="#ffffff" rx="6" />

		{#if title}
			<text x={width / 2} y="24" text-anchor="middle" fill="#111827" font-size="16" font-weight="600">{title}</text>
		{/if}
		{#if subtitle}
			<text x={width / 2} y="42" text-anchor="middle" fill="#6b7280" font-size="11">{subtitle}</text>
		{/if}

		{#each yTicks as t}
			<line x1={pad.left} y1={y(t)} x2={width - pad.right} y2={y(t)} stroke="#e5e7eb" stroke-width="1" />
			<text x={pad.left - 8} y={y(t) + 4} text-anchor="end" fill="#6b7280" font-size="11">{t}</text>
		{/each}

		{#each xTicks as t}
			<text x={x(t)} y={height - pad.bottom + 18} text-anchor="middle" fill="#6b7280" font-size="11">{t}</text>
		{/each}

		<line x1={pad.left} y1={pad.top} x2={pad.left} y2={height - pad.bottom} stroke="#9ca3af" stroke-width="1" />
		<line x1={pad.left} y1={height - pad.bottom} x2={width - pad.right} y2={height - pad.bottom} stroke="#9ca3af" stroke-width="1" />

		<text x={16} y={height / 2} text-anchor="middle" fill="#374151" font-size="12" transform="rotate(-90 16 {height / 2})">{yLabel}</text>
		<text x={width / 2} y={height - 6} text-anchor="middle" fill="#374151" font-size="12">{xLabel}</text>

		{#each series as s}
			<path d={linePath(s.points)} fill="none" stroke={s.color} stroke-width="2" />
			{#each s.points as p}
				<circle cx={x(p.n)} cy={y(p.success_rate)} r="2.6" fill={s.color} />
			{/each}
		{/each}

		{#each series as s, i}
			<line x1={pad.left + 12} y1={pad.top + 14 + i * 18} x2={pad.left + 32} y2={pad.top + 14 + i * 18} stroke={s.color} stroke-width="2" />
			<circle cx={pad.left + 22} cy={pad.top + 14 + i * 18} r="2.6" fill={s.color} />
			<text x={pad.left + 38} y={pad.top + 18 + i * 18} fill="#374151" font-size="11">{s.label}</text>
		{/each}
	</svg>
</figure>
