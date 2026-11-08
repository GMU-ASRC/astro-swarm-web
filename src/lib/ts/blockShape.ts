export const CORNER = 4;
export const NOTCH_LEFT = 12;
export const NOTCH_WIDTH = 16;
export const NOTCH_SLANT = 4;
export const NOTCH_DEPTH = 5;
export const HAT_HEIGHT = 7;
export const HAT_RISE = 10;
export const HAT_MAX_WIDTH = 96;
export const ARM_WIDTH = 16;
export const FOOTER_HEIGHT = 14;
export const SHADOW_OFFSET = 3;

export interface Point {
	x: number;
	y: number;
}

export interface Mouth {
	x: number;
	y: number;
	height: number;
}

const NOTCH_SPAN = NOTCH_LEFT + NOTCH_SLANT * 2 + NOTCH_WIDTH;

export function outline(width: number, height: number, isHat: boolean, mouth: Mouth | null): Point[] {
	const points: Point[] = [];
	const top = isHat ? HAT_HEIGHT : 0;
	if (isHat) addHatTop(points, width);
	else addNotchedTop(points, width);
	if (mouth && mouth.height > 0) addMouth(points, width, mouth);
	points.push({ x: width, y: Math.max(top + CORNER, height - CORNER) });
	points.push({ x: width - CORNER, y: height });
	addTab(points, 0, height);
	points.push({ x: CORNER, y: height });
	points.push({ x: 0, y: height - CORNER });
	return points;
}

function addNotchedTop(points: Point[], width: number) {
	points.push({ x: 0, y: CORNER });
	points.push({ x: CORNER, y: 0 });
	points.push({ x: NOTCH_LEFT, y: 0 });
	points.push({ x: NOTCH_LEFT + NOTCH_SLANT, y: NOTCH_DEPTH });
	points.push({ x: NOTCH_SPAN - NOTCH_SLANT, y: NOTCH_DEPTH });
	points.push({ x: NOTCH_SPAN, y: 0 });
	points.push({ x: width - CORNER, y: 0 });
	points.push({ x: width, y: CORNER });
}

function addHatTop(points: Point[], width: number) {
	const hatWidth = Math.min(HAT_MAX_WIDTH, width * 0.6);
	points.push({ x: 0, y: HAT_HEIGHT });
	points.push({ x: HAT_RISE, y: 0 });
	points.push({ x: hatWidth - HAT_RISE, y: 0 });
	points.push({ x: hatWidth, y: HAT_HEIGHT });
	points.push({ x: width - CORNER, y: HAT_HEIGHT });
	points.push({ x: width, y: HAT_HEIGHT + CORNER });
}

function addMouth(points: Point[], width: number, mouth: Mouth) {
	const bottom = mouth.y + mouth.height;
	points.push({ x: width, y: mouth.y - CORNER });
	points.push({ x: width - CORNER, y: mouth.y });
	addTab(points, mouth.x, mouth.y);
	points.push({ x: mouth.x, y: mouth.y });
	points.push({ x: mouth.x, y: bottom });
	points.push({ x: width - CORNER, y: bottom });
	points.push({ x: width, y: bottom + CORNER });
}

function addTab(points: Point[], left: number, y: number) {
	const tabEnd = left + NOTCH_SPAN;
	points.push({ x: tabEnd, y });
	points.push({ x: tabEnd - NOTCH_SLANT, y: y + NOTCH_DEPTH });
	points.push({ x: left + NOTCH_LEFT + NOTCH_SLANT, y: y + NOTCH_DEPTH });
	points.push({ x: left + NOTCH_LEFT, y });
}

export function pointList(points: Point[], offsetY = 0): string {
	return points.map((point) => `${point.x},${point.y + offsetY}`).join(' ');
}

export function highlightSegments(width: number, isHat: boolean): [Point, Point][] {
	const right = width - CORNER;
	if (isHat) {
		const y = HAT_HEIGHT + 2;
		return [[{ x: CORNER, y }, { x: right, y }]];
	}
	return [
		[{ x: CORNER, y: 2 }, { x: NOTCH_LEFT, y: 2 }],
		[{ x: NOTCH_SPAN, y: 2 }, { x: right, y: 2 }]
	];
}
