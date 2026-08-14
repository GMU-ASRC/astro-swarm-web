import type { ChartConfiguration } from 'chart.js';

const GRID = '#e5e7eb';
const TEXT = '#374151';
const FAINT = '#6b7280';

const DETECTION = '#2563eb';
const CAPTURE = '#dc2626';
const SUCCESS = '#4ade80';

const PERCENT_CEILING = 100;
const PERCENT_TICK_STEP = 25;
const VALUE_HEADROOM = '8%';

function baseOptions(
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

function placementSource(trials: number): string {
	return `Placement runs · ${trials} ${trials === 1 ? 'trial' : 'trials'}`;
}

function sweepSource(rows: SweepRow[]): string {
	if (rows.length === 0) return 'Ring sweep runs';
	const sizes = rows.map((row) => row.n);
	return `Ring sweep runs · n = ${Math.min(...sizes)}–${Math.max(...sizes)}`;
}

function percentScale(scale: object) {
	return {
		...scale,
		min: 0,
		max: PERCENT_CEILING,
		ticks: { color: TEXT, stepSize: PERCENT_TICK_STEP }
	};
}

export function headlineRatesConfig(
	detectionRate: number,
	captureRate: number,
	successRate: number,
	trials: number
): ChartConfiguration {
	const options = baseOptions('Outcome Rates', '% of trials', '', false, placementSource(trials));
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'bar',
		data: {
			labels: ['Evader detected', 'Evader captured', 'Defenders win'],
			datasets: [
				{
					label: 'Percent of trials',
					data: [detectionRate, captureRate, successRate],
					backgroundColor: [DETECTION, CAPTURE, SUCCESS]
				}
			]
		},
		options
	};
}

export function lineConfig(outcomes: string[]): ChartConfiguration {
	let wins = 0;
	const labels: number[] = [];
	const data: number[] = [];
	outcomes.forEach((outcome, index) => {
		if (outcome === 'win') wins += 1;
		labels.push(index + 1);
		data.push((100 * wins) / (index + 1));
	});

	const options = baseOptions(
		'Cumulative Defender Win Rate',
		'Defenders win (%)',
		'Trial',
		false,
		placementSource(outcomes.length)
	);
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels,
			datasets: [
				{ label: 'Defenders win', data, borderColor: DETECTION, backgroundColor: DETECTION, pointRadius: 0, borderWidth: 2, tension: 0.1 }
			]
		},
		options
	};
}

export function barConfig(outcomes: string[]): ChartConfiguration {
	const counts = { win: 0, lose: 0, timeout: 0 };
	for (const outcome of outcomes) {
		if (outcome === 'win') counts.win += 1;
		else if (outcome === 'lose') counts.lose += 1;
		else counts.timeout += 1;
	}
	const total = Math.max(1, outcomes.length);
	const values = [
		(100 * counts.win) / total,
		(100 * counts.lose) / total,
		(100 * counts.timeout) / total
	];

	const options = baseOptions(
		'Outcome Breakdown',
		'% of trials',
		'',
		false,
		placementSource(outcomes.length)
	);
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'bar',
		data: {
			labels: ['Intercept', 'Planet hit', 'Timeout'],
			datasets: [{ label: '% of trials', data: values, backgroundColor: ['#4ade80', '#f87171', '#fbbf24'] }]
		},
		options
	};
}

type SweepRow = {
	n: number;
	outcome?: string;
	detection_time?: number;
	capture_time?: number;
	detection_rate?: number;
	capture_rate?: number;
};

function sweepPoints(rows: SweepRow[]): SweepRow[] {
	return [...rows].sort((a, b) => a.n - b.n);
}

function rateOf(
	row: SweepRow,
	rateKey: 'detection_rate' | 'capture_rate',
	timeKey: 'detection_time' | 'capture_time'
): number {
	const averaged = row[rateKey];
	if (averaged != null) return averaged;
	const time = row[timeKey];
	return time != null && time >= 0 ? 100 : 0;
}

function sweepRateConfig(
	rows: SweepRow[],
	rateKey: 'detection_rate' | 'capture_rate',
	timeKey: 'detection_time' | 'capture_time',
	title: string,
	yTitle: string,
	color: string
): ChartConfiguration {
	const points = sweepPoints(rows);
	const rate = (row: SweepRow) => rateOf(row, rateKey, timeKey);

	const options = baseOptions(title, yTitle, 'Defenders in ring (n)', false, sweepSource(rows));
	options.scales.y = percentScale(options.scales.y) as never;

	return {
		type: 'line',
		data: {
			labels: points.map((point) => point.n),
			datasets: [
				{ label: yTitle, data: points.map(rate), borderColor: color, backgroundColor: color, pointRadius: 0, borderWidth: 2 }
			]
		},
		options
	};
}

export function detectionRateConfig(rows: SweepRow[]): ChartConfiguration {
	return sweepRateConfig(
		rows,
		'detection_rate',
		'detection_time',
		'Detection Success Rate vs Number of Defenders',
		'Detection success rate (%)',
		DETECTION
	);
}

export function captureRateConfig(rows: SweepRow[]): ChartConfiguration {
	return sweepRateConfig(
		rows,
		'capture_rate',
		'capture_time',
		'Capture Success Rate vs Number of Defenders',
		'Capture success rate (%)',
		CAPTURE
	);
}

export function combinedRatesConfig(rows: SweepRow[]): ChartConfiguration {
	const points = sweepPoints(rows);
	const options = baseOptions(
		'Detection and Capture Rates vs Number of Defenders',
		'% of ring sweep runs',
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
					label: 'Seen at least once',
					data: points.map((point) => rateOf(point, 'detection_rate', 'detection_time')),
					borderColor: DETECTION,
					backgroundColor: DETECTION,
					pointRadius: 0,
					borderWidth: 2
				},
				{
					label: 'Captured',
					data: points.map((point) => rateOf(point, 'capture_rate', 'capture_time')),
					borderColor: CAPTURE,
					backgroundColor: CAPTURE,
					pointRadius: 0,
					borderWidth: 2
				}
			]
		},
		options
	};
}

export function timesConfig(detection: number[], capture: number[]): ChartConfiguration {
	const count = Math.max(detection.length, capture.length);
	const labels = Array.from({ length: count }, (_, index) => index + 1);
	const clamp = (values: number[]) =>
		labels.map((_, index) => {
			const value = values[index];
			return value != null && value >= 0 ? value : 0;
		});

	return {
		type: 'bar',
		data: {
			labels,
			datasets: [
				{ label: 'Detection time', data: clamp(detection), backgroundColor: '#4ade80' },
				{ label: 'Capture time', data: clamp(capture), backgroundColor: '#f87171' }
			]
		},
		options: baseOptions(
			'Detection and Capture Times per Trial',
			'Time (s)',
			'Trial',
			true,
			placementSource(count)
		)
	};
}
