export interface StarLayer {
	count: number;
	scale: number;
	minArm: number;
	maxArm: number;
	alpha: number;
	twinkleWeight: number;
}

export interface Star {
	x: number;
	y: number;
	arm: number;
	phase: number;
	twinkleSpeed: number;
	color: [number, number, number];
}

export interface StarLayerData {
	config: StarLayer;
	stars: Star[];
}

const PIXEL = 2;
const MAX_SHIFT = 28;
const EASE = 9;

const STAR_COLORS: [number, number, number][] = [
	[255, 255, 255],
	[140, 217, 255],
	[115, 153, 255],
	[255, 217, 140],
	[255, 153, 217],
	[199, 153, 255],
	[153, 255, 178],
	[255, 140, 115],
];

const LAYERS: StarLayer[] = [
	{ count: 70, scale: 0.20, minArm: 0, maxArm: 1, alpha: 0.45, twinkleWeight: 0.25 },
	{ count: 45, scale: 0.50, minArm: 1, maxArm: 1, alpha: 0.70, twinkleWeight: 0.45 },
	{ count: 24, scale: 1.00, minArm: 1, maxArm: 2, alpha: 1.00, twinkleWeight: 0.75 },
];

function seededRandom(seed: number) {
	let s = seed;
	return () => {
		s = (s * 1664525 + 1013904223) & 0xffffffff;
		return (s >>> 0) / 0xffffffff;
	};
}

function pickColor(rand: () => number): [number, number, number] {
	if (rand() < 0.35) return STAR_COLORS[0];
	return STAR_COLORS[Math.floor(rand() * (STAR_COLORS.length - 1)) + 1];
}

export function buildLayers(width: number, height: number, seed: number): StarLayerData[] {
	const rand = seededRandom(seed);
	return LAYERS.map((cfg) => {
		const stars: Star[] = [];
		for (let i = 0; i < cfg.count; i++) {
			stars.push({
				x: rand() * width,
				y: rand() * height,
				arm: cfg.minArm + Math.round(rand() * (cfg.maxArm - cfg.minArm)),
				phase: rand() * Math.PI * 2,
				twinkleSpeed: 0.6 + rand() * 1.0,
				color: pickColor(rand),
			});
		}
		return { config: cfg, stars };
	});
}

export function drawFrame(
	ctx: CanvasRenderingContext2D,
	layers: StarLayerData[],
	width: number,
	height: number,
	mouseNormX: number,
	mouseNormY: number,
	time: number
) {
	ctx.clearRect(0, 0, width, height);

	for (const layer of layers) {
		const { config: cfg, stars } = layer;
		const offsetX = -mouseNormX * MAX_SHIFT * cfg.scale;
		const offsetY = -mouseNormY * MAX_SHIFT * cfg.scale;

		for (const star of stars) {
			const px = Math.round(((star.x + offsetX) % width + width) % width / PIXEL) * PIXEL;
			const py = Math.round(((star.y + offsetY) % height + height) % height / PIXEL) * PIXEL;

			const pulse = 0.5 + 0.5 * Math.sin(time * star.twinkleSpeed * 2 + star.phase);
			const twinkle = 1.0 - cfg.twinkleWeight + cfg.twinkleWeight * pulse;
			const alpha = cfg.alpha * twinkle;
			const arm = Math.round(star.arm * (0.4 + 0.6 * pulse));

			const [r, g, b] = star.color;
			ctx.fillStyle = `rgba(${r},${g},${b},${alpha})`;

			if (arm <= 0) {
				ctx.fillRect(px, py, PIXEL, PIXEL);
			} else {
				drawPlus(ctx, px, py, arm, r, g, b, alpha);
			}
		}
	}
}

function drawPlus(
	ctx: CanvasRenderingContext2D,
	x: number,
	y: number,
	arm: number,
	r: number,
	g: number,
	b: number,
	alpha: number
) {
	ctx.fillStyle = `rgba(${r},${g},${b},${alpha})`;
	ctx.fillRect(x, y, PIXEL, PIXEL);
	for (let i = 1; i <= arm; i++) {
		const d = i * PIXEL;
		ctx.fillRect(x, y - d, PIXEL, PIXEL);
		ctx.fillRect(x, y + d, PIXEL, PIXEL);
		ctx.fillRect(x - d, y, PIXEL, PIXEL);
		ctx.fillRect(x + d, y, PIXEL, PIXEL);
	}
}

export function lerpMouse(
	currentX: number,
	currentY: number,
	targetX: number,
	targetY: number,
	delta: number
): [number, number] {
	const t = Math.min(delta * EASE, 1.0);
	return [currentX + (targetX - currentX) * t, currentY + (targetY - currentY) * t];
}
