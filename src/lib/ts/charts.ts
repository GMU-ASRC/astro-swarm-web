import type { ChartConfiguration } from 'chart.js';

import {
	CAPTURE,
	DETECTION,
	SUCCESS,
	baseOptions,
	percentScale,
	placementSource,
	rateOf,
	sweepPoints,
	sweepSource,
	type SweepRow
} from '$lib/ts/chartBase';

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
