export type HeatmapLegend = {
	label: string;
	ticks: string[];
	stops: number[];
	colors: string[];
	missingColor: string | null;
};

export type HeatmapExport = {
	plot: SVGSVGElement;
	title: string;
	subtitle: string;
	legend: HeatmapLegend;
};

const SVG_NAMESPACE = 'http://www.w3.org/2000/svg';
const FONT_FAMILY = 'Helvetica, Arial, sans-serif';
const TEXT_COLOR = '#374151';
const MUTED_COLOR = '#6b7280';
const PADDING = 24;
const HEADER_HEIGHT = 58;
const LEGEND_HEIGHT = 70;
const LEGEND_BAR_WIDTH = 300;
const LEGEND_BAR_HEIGHT = 10;
const SWATCH_SIZE = 12;
const PIXEL_RATIO = 2;

function escapeText(value: string): string {
	return value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
}

function plotMarkup(plot: SVGSVGElement, x: number, y: number): string {
	const copy = plot.cloneNode(true) as SVGSVGElement;
	copy.querySelectorAll('.crosshair').forEach((node) => node.remove());
	copy.querySelectorAll('text').forEach((node) => {
		const isTitle = node.classList.contains('axis-title');
		node.setAttribute('fill', TEXT_COLOR);
		node.setAttribute('font-size', isTitle ? '12' : '11');
		if (isTitle) node.setAttribute('font-weight', '600');
	});
	copy.setAttribute('x', `${x}`);
	copy.setAttribute('y', `${y}`);
	copy.setAttribute('width', `${plot.viewBox.baseVal.width}`);
	copy.setAttribute('height', `${plot.viewBox.baseVal.height}`);
	return new XMLSerializer().serializeToString(copy);
}

function legendMarkup(legend: HeatmapLegend, centerX: number, top: number): string {
	const barX = centerX - LEGEND_BAR_WIDTH / 2;
	const barY = top + 22;
	const gradientStops = legend.colors
		.map((color, index) => `<stop offset="${legend.stops[index] * 100}%" stop-color="${color}" />`)
		.join('');
	const ticks = legend.ticks
		.map((tick, index) => {
			const anchor = index === 0 ? 'start' : index === legend.ticks.length - 1 ? 'end' : 'middle';
			const tickX = barX + legend.stops[index] * LEGEND_BAR_WIDTH;
			return `<text x="${tickX}" y="${barY + LEGEND_BAR_HEIGHT + 14}" text-anchor="${anchor}" font-size="11" fill="${MUTED_COLOR}">${escapeText(tick)}</text>`;
		})
		.join('');
	const swatchX = barX + LEGEND_BAR_WIDTH + 28;
	const missing = legend.missingColor
		? `<rect x="${swatchX}" y="${barY - 1}" width="${SWATCH_SIZE}" height="${SWATCH_SIZE}" fill="${legend.missingColor}" stroke="#d1d5db" />
			<text x="${swatchX + SWATCH_SIZE + 6}" y="${barY + 9}" font-size="11" fill="${MUTED_COLOR}">No data</text>`
		: '';
	return `<defs><linearGradient id="legend-ramp">${gradientStops}</linearGradient></defs>
		<text x="${centerX}" y="${top + 12}" text-anchor="middle" font-size="11" font-weight="600" fill="${TEXT_COLOR}">${escapeText(legend.label)}</text>
		<rect x="${barX}" y="${barY}" width="${LEGEND_BAR_WIDTH}" height="${LEGEND_BAR_HEIGHT}" fill="url(#legend-ramp)" stroke="#e5e7eb" />
		${ticks}
		${missing}`;
}

function buildDocument(chart: HeatmapExport): { markup: string; width: number; height: number } {
	const plotWidth = chart.plot.viewBox.baseVal.width;
	const plotHeight = chart.plot.viewBox.baseVal.height;
	const width = plotWidth + PADDING * 2;
	const height = HEADER_HEIGHT + plotHeight + LEGEND_HEIGHT + PADDING;
	const centerX = width / 2;
	const markup = `<svg xmlns="${SVG_NAMESPACE}" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" font-family="${FONT_FAMILY}">
		<rect width="${width}" height="${height}" fill="#ffffff" />
		<text x="${centerX}" y="${PADDING + 8}" text-anchor="middle" font-size="16" font-weight="700" fill="${TEXT_COLOR}">${escapeText(chart.title)}</text>
		<text x="${centerX}" y="${PADDING + 28}" text-anchor="middle" font-size="12" fill="${MUTED_COLOR}">${escapeText(chart.subtitle)}</text>
		${plotMarkup(chart.plot, PADDING, HEADER_HEIGHT)}
		${legendMarkup(chart.legend, centerX, HEADER_HEIGHT + plotHeight + 12)}
	</svg>`;
	return { markup, width, height };
}

function loadImage(url: string): Promise<HTMLImageElement> {
	return new Promise((resolve, reject) => {
		const image = new Image();
		image.onload = () => resolve(image);
		image.onerror = () => reject(new Error('The heatmap could not be drawn'));
		image.src = url;
	});
}

function saveBlob(blob: Blob, fileName: string) {
	const url = URL.createObjectURL(blob);
	const link = document.createElement('a');
	link.href = url;
	link.download = fileName;
	link.click();
	URL.revokeObjectURL(url);
}

export function heatmapFileName(title: string): string {
	const slug = title
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-|-$/g, '');
	return `${slug}-heatmap.png`;
}

export async function downloadHeatmapPng(chart: HeatmapExport, fileName: string): Promise<void> {
	const { markup, width, height } = buildDocument(chart);
	const svgUrl = URL.createObjectURL(new Blob([markup], { type: 'image/svg+xml;charset=utf-8' }));
	try {
		const image = await loadImage(svgUrl);
		const canvas = document.createElement('canvas');
		canvas.width = width * PIXEL_RATIO;
		canvas.height = height * PIXEL_RATIO;
		const context = canvas.getContext('2d');
		if (!context) throw new Error('The heatmap could not be drawn');
		context.drawImage(image, 0, 0, canvas.width, canvas.height);
		const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'));
		if (!blob) throw new Error('The heatmap could not be saved');
		saveBlob(blob, fileName);
	} finally {
		URL.revokeObjectURL(svgUrl);
	}
}
