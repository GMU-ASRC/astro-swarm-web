import type { ChartConfiguration } from 'chart.js';

export const GRID = '#e5e7eb';
export const TEXT = '#374151';
export const FAINT = '#6b7280';

export const DETECTION = '#2563eb';
export const CAPTURE = '#dc2626';
export const SUCCESS = '#4ade80';
export const RISK = '#dc2626';

export const COMPARISON_COLORS = [
	'#2563eb',
	'#16a34a',
	'#d97706',
	'#9333ea',
	'#0891b2',
	'#db2777',
	'#65a30d',
	'#e11d48'
];

const PERCENT_CEILING = 100;
const PERCENT_TICK_STEP = 25;
const VALUE_HEADROOM = '8%';

export type SweepRow = {
	n: number;
	outcome?: string;
	detection_time?: number;
	capture_time?: number;
	detection_rate?: number;
	capture_rate?: number;
	risk?: number;
};

export type AttritionRow = {
	defenders: number;
	launched: number;
	risk: number;
	capture_rate: number;
};

export type AttritionSeries = {
	n: number;
	points: AttritionRow[];
};

export type ComparisonEntry = {
	id: string;
	username: string;
	success_rate: number;
	sweep: { n: number; capture_rate: number; risk: number }[];
};

export function baseOptions(
	title: string,
	yTitle: string,
	xTitle: string,
	showLegend = false,
	subtitle = ''
) {
	return {
		responsive: true,
		maintainAspectRatio: false,
		plugins: {
			title: { display: true, text: title, color: TEXT, font: { size: 15 } },
			subtitle: {
				display: subtitle !== '',
				text: subtitle,
				color: FAINT,
				font: { size: 11 },
				padding: { bottom: 10 }
			},
			legend: { display: showLegend, labels: { color: TEXT } }
		},
		scales: {
			y: {
				title: { display: true, text: yTitle, color: TEXT },
				ticks: { color: TEXT },
				grid: { color: GRID },
				grace: VALUE_HEADROOM
			},
			x: {
				title: { display: true, text: xTitle, color: TEXT },
				ticks: { color: TEXT },
				grid: { color: GRID },
				offset: true
			}
		}
	};
}

export function placementSource(trials: number): string {
	return `Placement runs · ${trials} ${trials === 1 ? 'trial' : 'trials'}`;
}

export function sweepSource(rows: SweepRow[]): string {
	if (rows.length === 0) return 'Ring sweep runs';
	const sizes = rows.map((row) => row.n);
	return `Ring sweep runs · n = ${Math.min(...sizes)}–${Math.max(...sizes)}`;
}

export function percentScale(scale: object) {
	return {
		...scale,
		min: 0,
		max: PERCENT_CEILING,
		ticks: { color: TEXT, stepSize: PERCENT_TICK_STEP }
	};
}

export function sweepPoints(rows: SweepRow[]): SweepRow[] {
	return [...rows].sort((a, b) => a.n - b.n);
}

export function rateOf(
	row: SweepRow,
	rateKey: 'detection_rate' | 'capture_rate',
	timeKey: 'detection_time' | 'capture_time'
): number {
	const averaged = row[rateKey];
	if (averaged != null) return averaged;
	const time = row[timeKey];
	return time != null && time >= 0 ? 100 : 0;
}

// Ring sizes are consecutive, so drawing every one of them is unreadable. Take
// an even spread across the sweep and always keep the largest ring, which is the
// one the sweep stopped at.
export function spreadSeries<T>(series: T[], limit: number): T[] {
	if (series.length <= limit) return series;
	const step = (series.length - 1) / (limit - 1);
	return Array.from({ length: limit }, (_, index) => series[Math.round(index * step)]);
}

export function riskOf(row: SweepRow): number {
	if (row.risk != null) return row.risk;
	return 100 - rateOf(row, 'capture_rate', 'capture_time');
}

export type { ChartConfiguration };
