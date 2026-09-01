import type { ChartConfiguration } from 'chart.js';

import {
	CAPTURE,
	COMPARISON_COLORS,
	RISK,
	SUCCESS,
	baseOptions,
	percentScale,
	rateOf,
	riskOf,
	sweepPoints,
	sweepSource,
	spreadSeries,
	type AttritionRow,
	type AttritionSeries,
	type ProgressSeries,
	type ComparisonEntry,
	type SweepRow
} from '$lib/ts/chartBase';

const MAX_ATTRITION_SERIES = 6;

export function riskConfig(rows: SweepRow[]): ChartConfiguration {
	const points = sweepPoints(rows);
	const options = baseOptions(
		'Risk vs Number of Defenders',
		'Risk = 1 - capture success rate (%)',
		'Defenders in ring (n)',
		true,
		sweepSource(rows)
	);
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels: points.map((point) => point.n),
			datasets: [
				{
					label: 'Risk',
					data: points.map(riskOf),
					borderColor: RISK,
					backgroundColor: RISK,
					pointRadius: 0,
					borderWidth: 2
				},
				{
					label: 'Capture success rate',
					data: points.map((point) => rateOf(point, 'capture_rate', 'capture_time')),
					borderColor: SUCCESS,
					backgroundColor: SUCCESS,
					pointRadius: 0,
					borderWidth: 2
				}
			]
		},
		options
	};
}

// The attrition curve reads right to left: a full line sits at the high end of
// the axis, and every trade moves the algorithm one rung down it.
export function attritionRiskConfig(rows: AttritionRow[]): ChartConfiguration {
	const points = [...rows].sort((a, b) => a.defenders - b.defenders);
	const launched = points.reduce((total, point) => total + point.launched, 0);
	const options = baseOptions(
		'Risk as the Defender Line Thins',
		'Risk = 1 - capture success rate (%)',
		'Defenders still standing when the evader launched',
		false,
		`${launched} evaders launched across every trial`
	);
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels: points.map((point) => point.defenders),
			datasets: [
				{
					label: 'Risk at this line size',
					data: points.map((point) => point.risk),
					borderColor: RISK,
					backgroundColor: RISK,
					pointRadius: 3,
					borderWidth: 2
				}
			]
		},
		options
	};
}

// One line per submitted entry, so a level page can show how every commander's
// algorithm scales against the size of the line it is given.
// One curve per ring size: a line that started at n and traded itself down, so
// two algorithms can be compared on what their risk does as the line thins
// rather than on a single number at full strength.
export function sweepAttritionConfig(series: AttritionSeries[]): ChartConfiguration {
	const chosen = spreadSeries(
		[...series].sort((a, b) => a.n - b.n),
		MAX_ATTRITION_SERIES
	);
	const sizes = new Set<number>();
	for (const entry of chosen) {
		for (const point of entry.points) sizes.add(point.defenders);
	}
	const labels = [...sizes].sort((a, b) => a - b);

	const options = baseOptions(
		'Risk as the Line Thins, by Ring Size',
		'Risk = 1 - capture success rate (%)',
		'Defenders still standing when the evader launched',
		true,
		`Ring sweep runs · ${series.length} ${series.length === 1 ? 'ring size' : 'ring sizes'} measured`
	);
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels,
			datasets: chosen.map((entry, index) => {
				const byDefenders = new Map(entry.points.map((point) => [point.defenders, point.risk]));
				const color = COMPARISON_COLORS[index % COMPARISON_COLORS.length];
				return {
					label: `n = ${entry.n}`,
					data: labels.map((defenders) => byDefenders.get(defenders) ?? null),
					borderColor: color,
					backgroundColor: color,
					pointRadius: 0,
					borderWidth: 2,
					spanGaps: true
				};
			})
		},
		options
	};
}

// The three ring-sweep curves drawn side by side: how each ring size's capture
// rate settles, the risk that leaves, and how much of the line is still
// standing, all read wave by wave rather than in total. Every ring size gets its
// own line, so the sizes can be compared as runs rather than as end numbers.
export function ringProgressConfig(
	series: ProgressSeries[],
	key: 'capture_rate' | 'risk' | 'defenders',
	title: string,
	yTitle: string
): ChartConfiguration {
	const chosen = spreadSeries(
		[...series].sort((a, b) => a.n - b.n),
		MAX_ATTRITION_SERIES
	);
	const faced = new Set<number>();
	for (const entry of chosen) {
		for (const point of entry.points) faced.add(point.faced);
	}
	const labels = [...faced].sort((a, b) => a - b);

	const options = baseOptions(title, yTitle, 'Evaders faced', true, ringSource(series));
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels,
			datasets: chosen.map((entry, index) => {
				// Defenders are shown as a share of the ring so rings of
				// different sizes sit on one axis.
				const scale = key === 'defenders' && entry.n > 0 ? 100 / entry.n : 1;
				const byFaced = new Map(entry.points.map((point) => [point.faced, point[key] * scale]));
				const color = COMPARISON_COLORS[index % COMPARISON_COLORS.length];
				return {
					label: `n = ${entry.n}`,
					data: labels.map((value) => byFaced.get(value) ?? null),
					borderColor: color,
					backgroundColor: color,
					pointRadius: 0,
					borderWidth: 2,
					spanGaps: true
				};
			})
		},
		options
	};
}

function ringSource(series: ProgressSeries[]): string {
	if (series.length === 0) return 'Ring sweep runs';
	const sizes = series.map((entry) => entry.n);
	return `Ring sweep runs · n = ${Math.min(...sizes)}–${Math.max(...sizes)}`;
}

export const BEST_SERIES_ID = 'best';

export function comparisonConfig(
	entries: ComparisonEntry[],
	key: 'capture_rate' | 'risk',
	title: string,
	yTitle: string
): ChartConfiguration {
	const sizes = new Set<number>();
	for (const item of entries) {
		for (const point of item.sweep) sizes.add(point.n);
	}
	const labels = [...sizes].sort((a, b) => a - b);

	const options = baseOptions(title, yTitle, 'Defenders in ring (n)', true, comparisonSource(entries));
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels,
			datasets: entries.map((item, index) => {
				const byN = new Map(item.sweep.map((point) => [point.n, point[key]]));
				const envelope = item.id === BEST_SERIES_ID;
				const color = envelope ? RISK : COMPARISON_COLORS[index % COMPARISON_COLORS.length];
				return {
					label: envelope ? item.username : `${item.username} · ${item.success_rate}%`,
					data: labels.map((n) => byN.get(n) ?? null),
					borderColor: color,
					backgroundColor: color,
					pointRadius: 0,
					borderWidth: envelope ? 4 : 2,
					borderDash: envelope ? [6, 4] : undefined,
					spanGaps: true
				};
			})
		},
		options
	};
}

function comparisonSource(entries: ComparisonEntry[]): string {
	if (entries.length === 0) return 'No submitted entries yet';
	return `Top ${entries.length} ${entries.length === 1 ? 'entry' : 'entries'} by capture success rate`;
}
