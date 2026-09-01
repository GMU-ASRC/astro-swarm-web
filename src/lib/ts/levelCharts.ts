import type { ChartConfiguration } from 'chart.js';

import {
	COMPARISON_COLORS,
	FAINT,
	RISK,
	SUCCESS,
	baseOptions,
	countScale,
	percentScale,
	rateOf,
	riskOf,
	placementSource,
	seriesRamp,
	sweepPoints,
	sweepSource,
	LEGEND_MAX,
	type AttritionRow,
	type AttritionSeries,
	type ProgressSeries,
	type ComparisonEntry,
	type SweepRow
} from '$lib/ts/chartBase';


export function riskConfig(rows: SweepRow[]): ChartConfiguration {
	const points = sweepPoints(rows);
	const options = baseOptions(
		'Risk by Ring Size',
		'Risk (%)',
		'Ring size (n)',
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
					label: 'Capture rate',
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

// Risk trial by trial, so a run that averages well but collapses on a handful of
// scatters can be told apart from one that holds the same line every time.
export function trialRiskConfig(destroyed: number[], resolved: number[]): ChartConfiguration {
	const labels = destroyed.map((_, index) => index + 1);
	const risks = destroyed.map((count, index) => {
		const faced = resolved[index] ?? 0;
		return faced > 0 ? Math.round((100 - (100 * count) / faced) * 10) / 10 : 100;
	});
	const totalDestroyed = destroyed.reduce((sum, count) => sum + count, 0);
	const totalResolved = resolved.reduce((sum, count) => sum + count, 0);
	const overall =
		totalResolved > 0 ? Math.round((100 - (100 * totalDestroyed) / totalResolved) * 10) / 10 : 100;

	const options = baseOptions(
		'Risk per Trial',
		'Risk (%)',
		'Trial',
		true,
		placementSource(labels.length)
	);
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels,
			datasets: [
				{
					label: 'Risk in this trial',
					data: risks,
					borderColor: RISK,
					backgroundColor: RISK,
					pointRadius: 0,
					borderWidth: 2
				},
				{
					label: `Risk across every trial · ${overall}%`,
					data: labels.map(() => overall),
					borderColor: FAINT,
					backgroundColor: FAINT,
					pointRadius: 0,
					borderWidth: 2,
					borderDash: [6, 4]
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
		'Risk as the Line Thins',
		'Risk (%)',
		'Defenders standing',
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
					label: 'Risk',
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
	const chosen = [...series].sort((a, b) => a.n - b.n);
	const colors = seriesRamp(chosen.length);
	const sizes = new Set<number>();
	for (const entry of chosen) {
		for (const point of entry.points) sizes.add(point.defenders);
	}
	const labels = [...sizes].sort((a, b) => a - b);

	const options = baseOptions(
		'Risk as the Line Thins, per Ring',
		'Risk (%)',
		'Defenders standing',
		true,
		rampSource(
			`${series.length} ${series.length === 1 ? 'ring size' : 'ring sizes'} measured`,
			chosen.length
		)
	);
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels,
			datasets: chosen.map((entry, index) => {
				const byDefenders = new Map(entry.points.map((point) => [point.defenders, point.risk]));
				const color = colors[index];
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
	const chosen = [...series].sort((a, b) => a.n - b.n);
	const colors = seriesRamp(chosen.length);
	const faced = new Set<number>();
	for (const entry of chosen) {
		for (const point of entry.points) faced.add(point.faced);
	}
	const labels = [...faced].sort((a, b) => a - b);

	const options = baseOptions(
		title,
		yTitle,
		'Evaders faced',
		true,
		ringSource(series, chosen.length)
	);
	// The line left is a count of ships; the other two curves are rates.
	const counting = key === 'defenders';
	options.scales.y = (
		counting ? countScale(options.scales.y) : percentScale(options.scales.y)
	) as never;

	return {
		type: 'line',
		data: {
			labels,
			datasets: chosen.map((entry, index) => {
				const byFaced = new Map(entry.points.map((point) => [point.faced, point[key]]));
				const color = colors[index];
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

function ringSource(series: ProgressSeries[], drawn: number): string {
	if (series.length === 0) return 'Ring sweep runs';
	const sizes = series.map((entry) => entry.n);
	return rampSource(`n = ${Math.min(...sizes)}–${Math.max(...sizes)}`, drawn);
}

// With one line per ring size the legend runs long, so the subtitle says how to
// read the color and that a legend entry can be clicked to isolate one ring.
function rampSource(detail: string, drawn: number): string {
	const source = `Ring sweep runs · ${detail}`;
	if (drawn <= LEGEND_MAX) return source;
	return `${source} · blue is the smallest ring, red the largest · click a legend entry to isolate it`;
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

	const options = baseOptions(title, yTitle, 'Ring size (n)', true, comparisonSource(entries));
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
