import type { Replay } from '$lib/ts/evaluation';

export type ReplayMode = 'defense' | 'swarm';

export interface StageOverlay {
	frameIndex: number;
	selectedSlot: number | null;
}

export const STAGE_WIDTH = 640;
export const STAGE_HEIGHT = 360;

// The ship icons are drawn far larger than the ships really are, so the hull
// each one actually collides with is drawn underneath at true scale.
const DEFAULT_HULL = 9;

const DEFENDER = '#7c9eff';
const EVADER = '#ff6a52';
const SWARM_A = '#ff6a52';
const SWARM_B = '#7c9eff';
const LEADER = '#ffd54a';

// In swarm mode the recorded ships are two milling groups followed by the
// player-flown leader, so they are colored by group instead of by team.
export function shipColor(replay: Replay, mode: ReplayMode, slot: number): string {
	if (mode !== 'swarm') return slot >= replay.defenders ? EVADER : DEFENDER;
	if (slot >= replay.defenders) return LEADER;
	return slot < replay.defenders / 2 ? SWARM_A : SWARM_B;
}

export function shipLabel(replay: Replay, mode: ReplayMode, slot: number): string {
	if (mode !== 'swarm')
		return slot >= replay.defenders
			? `Evader #${slot - replay.defenders + 1}`
			: `Defender #${slot + 1}`;
	if (slot >= replay.defenders) return 'Leader';
	const group = slot < replay.defenders / 2 ? 'A' : 'B';
	return `Group ${group} agent #${slot + 1}`;
}

export function sampleFrame(replay: Replay, frameIndex: number, blend: number): number[] {
	const current = replay.frames[frameIndex] ?? [];
	const next = replay.frames[Math.min(replay.frames.length - 1, frameIndex + 1)] ?? current;
	const sampled: number[] = [];
	const slots = current.length / 3;
	for (let slot = 0; slot < slots; slot++) {
		const currentX = current[slot * 3];
		const nextX = next[slot * 3];
		const currentY = current[slot * 3 + 1];
		const currentRotation = current[slot * 3 + 2];
		const dead = currentX < 0 || nextX < 0;
		sampled.push(dead ? currentX : currentX + (nextX - currentX) * blend);
		sampled.push(dead ? currentY : currentY + (next[slot * 3 + 1] - currentY) * blend);
		sampled.push(currentRotation + (next[slot * 3 + 2] - currentRotation) * blend);
	}
	return sampled;
}

export function drawStage(
	ctx: CanvasRenderingContext2D,
	replay: Replay,
	mode: ReplayMode,
	frame: number[],
	overlay: StageOverlay
) {
	const [arenaWidth, arenaHeight] = replay.arena;
	const sx = STAGE_WIDTH / arenaWidth;
	const sy = STAGE_HEIGHT / arenaHeight;

	ctx.fillStyle = '#0a0a12';
	ctx.fillRect(0, 0, STAGE_WIDTH, STAGE_HEIGHT);

	drawPlanet(ctx, replay, sx, sy);

	const total = frame.length / 3;

	const coneRadius = (replay.view ?? 300) * sx;
	const halfAngle = ((replay.fov ?? 70) * Math.PI) / 360;
	for (let slot = 0; slot < replay.defenders; slot++) {
		const x = frame[slot * 3];
		const y = frame[slot * 3 + 1];
		const rotation = (frame[slot * 3 + 2] * Math.PI) / 180;
		if (x < 0) continue;
		const cone =
			mode === 'swarm' ? `${shipColor(replay, mode, slot)}22` : 'rgba(124,158,255,0.16)';
		drawCone(ctx, x * sx, y * sy, rotation, coneRadius, halfAngle, cone);
	}

	if (mode === 'swarm') drawSwarmMarkers(ctx, replay, frame, sx, sy);

	const hullRadius = (replay.hull ?? DEFAULT_HULL) * sx;
	for (let slot = 0; slot < total; slot++) {
		const x = frame[slot * 3];
		const y = frame[slot * 3 + 1];
		const rotation = (frame[slot * 3 + 2] * Math.PI) / 180;
		if (x < 0) continue;
		const color = shipColor(replay, mode, slot);
		drawHull(ctx, x * sx, y * sy, hullRadius, color);
		drawShip(ctx, x * sx, y * sy, rotation, color);
	}

	drawSelection(ctx, frame, overlay.selectedSlot, sx, sy);
	drawHud(ctx, replay, mode, overlay.frameIndex);
}

function drawPlanet(ctx: CanvasRenderingContext2D, replay: Replay, sx: number, sy: number) {
	const [planetX, planetY, planetRadius] = replay.planet;
	ctx.beginPath();
	ctx.arc(planetX * sx, planetY * sy, planetRadius * sx, 0, Math.PI * 2);
	ctx.fillStyle = '#3a7d5a';
	ctx.fill();
	ctx.strokeStyle = 'rgba(255,255,255,0.15)';
	ctx.lineWidth = 1;
	ctx.stroke();
}

function drawSelection(
	ctx: CanvasRenderingContext2D,
	frame: number[],
	selectedSlot: number | null,
	sx: number,
	sy: number
) {
	if (selectedSlot === null || selectedSlot >= frame.length / 3) return;
	const x = frame[selectedSlot * 3];
	const y = frame[selectedSlot * 3 + 1];
	if (x < 0) return;
	ctx.beginPath();
	ctx.arc(x * sx, y * sy, 10, 0, Math.PI * 2);
	ctx.strokeStyle = '#ffd54a';
	ctx.lineWidth = 2;
	ctx.stroke();
}

function drawHud(ctx: CanvasRenderingContext2D, replay: Replay, mode: ReplayMode, frameIndex: number) {
	ctx.fillStyle = 'rgba(255,255,255,0.5)';
	ctx.font = '11px monospace';
	const seconds = (frameIndex / (replay.fps || 1)).toFixed(1);
	const count = mode === 'swarm' ? `agents=${replay.defenders}` : `N=${replay.defenders}`;
	ctx.fillText(
		`${count}  ${replay.outcome}  frame ${frameIndex + 1}/${replay.frames.length}  t=${seconds}s`,
		8,
		16
	);
}

function groupCenter(frame: number[], from: number, to: number): [number, number] | null {
	let sumX = 0;
	let sumY = 0;
	let count = 0;
	for (let slot = from; slot < to; slot++) {
		if (frame[slot * 3] < 0) continue;
		sumX += frame[slot * 3];
		sumY += frame[slot * 3 + 1];
		count++;
	}
	return count === 0 ? null : [sumX / count, sumY / count];
}

function isMerged(replay: Replay, frame: number[]): boolean {
	const limit = replay.stats?.merge_distance;
	if (!limit || replay.defenders < 2) return false;
	const reached = new Set<number>([0]);
	const queue = [0];
	while (queue.length > 0) {
		const current = queue.pop() as number;
		for (let slot = 0; slot < replay.defenders; slot++) {
			if (reached.has(slot)) continue;
			const dx = frame[current * 3] - frame[slot * 3];
			const dy = frame[current * 3 + 1] - frame[slot * 3 + 1];
			if (Math.hypot(dx, dy) <= limit) {
				reached.add(slot);
				queue.push(slot);
			}
		}
	}
	return reached.size === replay.defenders;
}

function drawSwarmMarkers(
	ctx: CanvasRenderingContext2D,
	replay: Replay,
	frame: number[],
	sx: number,
	sy: number
) {
	const half = Math.floor(replay.defenders / 2);
	const goalRadius = replay.stats?.goal_radius;
	if (goalRadius) {
		const [planetX, planetY] = replay.planet;
		ctx.beginPath();
		ctx.arc(planetX * sx, planetY * sy, goalRadius * sx, 0, Math.PI * 2);
		ctx.strokeStyle = 'rgba(255,255,255,0.25)';
		ctx.lineWidth = 1;
		ctx.setLineDash([4, 4]);
		ctx.stroke();
		ctx.setLineDash([]);
	}
	if (isMerged(replay, frame)) {
		drawMarker(ctx, groupCenter(frame, 0, replay.defenders), sx, sy, LEADER);
		return;
	}
	drawMarker(ctx, groupCenter(frame, 0, half), sx, sy, SWARM_A);
	drawMarker(ctx, groupCenter(frame, half, replay.defenders), sx, sy, SWARM_B);
}

function drawMarker(
	ctx: CanvasRenderingContext2D,
	center: [number, number] | null,
	sx: number,
	sy: number,
	color: string
) {
	if (!center) return;
	const x = center[0] * sx;
	const y = center[1] * sy;
	const radius = 6;
	const arm = radius + 4;
	ctx.strokeStyle = color;
	ctx.lineWidth = 1.5;
	ctx.beginPath();
	ctx.arc(x, y, radius, 0, Math.PI * 2);
	ctx.stroke();
	ctx.beginPath();
	ctx.moveTo(x - arm, y);
	ctx.lineTo(x + arm, y);
	ctx.moveTo(x, y - arm);
	ctx.lineTo(x, y + arm);
	ctx.stroke();
}

function drawCone(
	ctx: CanvasRenderingContext2D,
	x: number,
	y: number,
	rotation: number,
	radius: number,
	halfAngle: number,
	color: string
) {
	ctx.beginPath();
	ctx.moveTo(x, y);
	ctx.arc(x, y, radius, rotation - halfAngle, rotation + halfAngle);
	ctx.closePath();
	ctx.fillStyle = color;
	ctx.fill();
}

// Two of these circles touching is exactly what the simulator counts as a
// capture, which the oversized icons alone never showed.
function drawHull(ctx: CanvasRenderingContext2D, x: number, y: number, radius: number, color: string) {
	ctx.beginPath();
	ctx.arc(x, y, Math.max(radius, 1), 0, Math.PI * 2);
	ctx.strokeStyle = `${color}88`;
	ctx.lineWidth = 1;
	ctx.stroke();
}

function drawShip(ctx: CanvasRenderingContext2D, x: number, y: number, rotation: number, color: string) {
	ctx.save();
	ctx.translate(x, y);
	ctx.rotate(rotation);
	ctx.beginPath();
	ctx.moveTo(6, 0);
	ctx.lineTo(-4, -3.5);
	ctx.lineTo(-4, 3.5);
	ctx.closePath();
	ctx.fillStyle = color;
	ctx.fill();
	ctx.restore();
}
