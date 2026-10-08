import type { ProgressSeries } from './chartBase';

export type HeatmapKey = 'capture_rate' | 'risk' | 'defenders';

const BAD = [0.62, 0.2, 25];
const NEUTRAL = [0.93, 0.005, 280];
const GOOD = [0.55, 0.17, 255];

export function divergingShade(goodness: number): string {
	const clamped = Math.max(0, Math.min(1, goodness));
	const lowHalf = clamped < 0.5;
	const from = lowHalf ? BAD : NEUTRAL;
	const to = lowHalf ? NEUTRAL : GOOD;
	const local = lowHalf ? clamped / 0.5 : (clamped - 0.5) / 0.5;
	const mix = (index: number) => from[index] + (to[index] - from[index]) * local;
	const hue = lowHalf ? BAD[2] : GOOD[2];
	return `oklch(${mix(0).toFixed(3)} ${mix(1).toFixed(3)} ${hue.toFixed(1)})`;
}

export function goodnessOf(key: HeatmapKey, value: number, ringSize: number): number {
	if (key === 'capture_rate') return value / 100;
	if (key === 'risk') return 1 - value / 100;
	return ringSize > 0 ? value / ringSize : 0;
}

export function formatValue(key: HeatmapKey, value: number): string {
	return key === 'defenders' ? `${Math.round(value)}` : `${value.toFixed(1)}%`;
}

export type HeatmapGrid = {
	rings: number[];
	faced: number[];
	values: (number | null)[][];
};

export function buildGrid(series: ProgressSeries[], key: HeatmapKey): HeatmapGrid {
	const chosen = [...series].sort((a, b) => a.n - b.n);
	const facedSet = new Set<number>();
	for (const entry of chosen) {
		for (const point of entry.points) facedSet.add(point.faced);
	}
	const faced = [...facedSet].sort((a, b) => a - b);
	const values = chosen.map((entry) => {
		const byFaced = new Map(entry.points.map((point) => [point.faced, point[key]]));
		return faced.map((value) => byFaced.get(value) ?? null);
	});
	return { rings: chosen.map((entry) => entry.n), faced, values };
}

export function ringTicks(rings: number[]): number[] {
	if (rings.length <= 12) return rings.map((_, index) => index);
	return rings
		.map((n, index) => (index === 0 || n % 10 === 0 ? index : -1))
		.filter((index) => index >= 0);
}
