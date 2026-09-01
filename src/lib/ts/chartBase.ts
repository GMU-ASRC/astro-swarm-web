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
const PERCENT_TICKS = [0, 25, 50, 75, 100];

// A line sitting exactly on 0 or 100 is drawn half outside the plot area, so the
// scale runs a little past both ends while the ticks stay on the round numbers.
const PERCENT_HEADROOM = 3;

// Breathing room between the plot area and the edge of the card, so a line that
// runs to the last point does not touch the frame.
const CONTENT_PADDING = { top: 4, right: 14, bottom: 4, left: 6 };
const RAMP_START_HUE = 220;
const RAMP_END_HUE = 0;

// Past this many lines the legend is worth calling out in the subtitle, since it
// wraps into rows. Every line is still named: on the site a legend entry can be
// clicked to pull one ring out of the group.
export const LEGEND_MAX = 8;
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

export type ProgressPoint = {
	faced: number;
	capture_rate: number;
	risk: number;
	defenders: number;
	trials: number;
};

export type ProgressSeries = {
	n: number;
	points: ProgressPoint[];
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
		layout: { padding: CONTENT_PADDING },
		plugins: {
			title: { display: true, text: title, color: TEXT, font: { size: 15 } },
			subtitle: {
				display: subtitle !== '',
				text: subtitle,
				color: FAINT,
				font: { size: 11 },
				padding: { bottom: 10 }
			},
			legend: {
				display: showLegend,
				labels: { color: TEXT, boxWidth: 10, boxHeight: 10, padding: 8, font: { size: 10 } }
			}
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
		min: -PERCENT_HEADROOM,
		max: PERCENT_CEILING + PERCENT_HEADROOM,
		ticks: { color: TEXT, stepSize: PERCENT_TICK_STEP },
		afterBuildTicks: (axis: { ticks: { value: number }[] }) => {
			axis.ticks = PERCENT_TICKS.map((value) => ({ value }));
		}
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

// Ring size is an ordered quantity, so a chart with one line per n reads as a
// ramp rather than as a set of unrelated colors: the smallest ring is blue and
// the largest is red, and a line's color alone says where it sits in the sweep.
export function seriesRamp(count: number): string[] {
	if (count < 2) return [`hsl(${RAMP_END_HUE}, 70%, 45%)`];
	const span = RAMP_START_HUE - RAMP_END_HUE;
	return Array.from({ length: count }, (_, index) => {
		const hue = RAMP_START_HUE - (span * index) / (count - 1);
		return `hsl(${Math.round(hue)}, 70%, 45%)`;
	});
}

export function riskOf(row: SweepRow): number {
	if (row.risk != null) return row.risk;
	return 100 - rateOf(row, 'capture_rate', 'capture_time');
}

export type { ChartConfiguration };
