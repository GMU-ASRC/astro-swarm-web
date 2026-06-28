import type { ChartConfiguration } from 'chart.js';

const GRID = '#e5e7eb';
const TEXT = '#374151';

function baseOptions(title: string, yTitle: string, xTitle: string, showLegend = false) {
	return {
		responsive: true,
		maintainAspectRatio: false,
		plugins: {
			title: { display: true, text: title, color: TEXT, font: { size: 15 } },
			legend: { display: showLegend, labels: { color: TEXT } }
		},
		scales: {
			y: { title: { display: true, text: yTitle, color: TEXT }, ticks: { color: TEXT }, grid: { color: GRID } },
			x: { title: { display: true, text: xTitle, color: TEXT }, ticks: { color: TEXT }, grid: { color: GRID } }
		}
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

	const options = baseOptions('Cumulative Detection Rate', 'Detection Rate (%)', 'Trial');
	options.scales.y = { ...options.scales.y, min: 0, max: 100 } as never;

	return {
		type: 'line',
		data: {
			labels,
			datasets: [
				{ label: 'Detection rate', data, borderColor: '#2563eb', backgroundColor: '#2563eb', pointRadius: 0, borderWidth: 2, tension: 0.1 }
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

	const options = baseOptions('Outcome Breakdown', '% of trials', '');
	options.scales.y = { ...options.scales.y, min: 0, max: 100 } as never;

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

function sweepRateConfig(
	rows: SweepRow[],
	rateKey: 'detection_rate' | 'capture_rate',
	timeKey: 'detection_time' | 'capture_time',
	title: string,
	yTitle: string,
	color: string
): ChartConfiguration {
	const points = [...rows].sort((a, b) => a.n - b.n);
	const rate = (row: SweepRow) => {
		const averaged = row[rateKey];
		if (averaged != null) return averaged;
		const time = row[timeKey];
		return time != null && time >= 0 ? 100 : 0;
	};

	const options = baseOptions(title, yTitle, 'Defenders in ring (n)');
	options.scales.y = { ...options.scales.y, min: 0, max: 100 } as never;

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
		'#2563eb'
	);
}

export function captureRateConfig(rows: SweepRow[]): ChartConfiguration {
	return sweepRateConfig(
		rows,
		'capture_rate',
		'capture_time',
		'Capture Success Rate vs Number of Defenders',
		'Capture success rate (%)',
		'#dc2626'
	);
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
		options: baseOptions('Detection and Capture Times per Trial', 'Time (s)', 'Trial', true)
	};
}
