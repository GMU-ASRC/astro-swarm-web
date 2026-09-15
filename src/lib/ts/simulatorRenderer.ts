import {
	PIXELS_PER_METER,
	VALUES_PER_ROBOT,
	type SimulatorEntry,
	type SimulatorReplay
} from '$lib/ts/simulator';

const BACKGROUND = '#d5d5db';
const GRID = '#bdbdc4';
const BORDER = '#e63333';
const WALL_FILL = '#737385';
const WALL_EDGE = '#383847';
const CIRCLE_FILL = '#8c664d';
const CIRCLE_EDGE = '#4d3326';
const HUD_TEXT = 'rgba(35, 35, 50, 0.75)';
const UNKNOWN_SPECIES = '#888888';
const ZONE_FILL_ALPHA = '29';
const DEFAULT_ROBOT_RADIUS = 6;
const MIN_ROBOT_RADIUS = 2;

export interface SimulatorScene {
	entry: SimulatorEntry;
	replay: SimulatorReplay;
	robotSpecies: Map<number, string>;
	speciesColors: Map<string, string>;
	speciesRadii: Map<string, number>;
}

export interface RobotPose {
	id: number;
	x: number;
	y: number;
	rotation: number;
}

export function createScene(entry: SimulatorEntry, replay: SimulatorReplay): SimulatorScene {
	const speciesColors = new Map<string, string>();
	const speciesRadii = new Map<string, number>();
	for (const species of entry.species) {
		speciesColors.set(species.id, species.color);
		speciesRadii.set(species.id, species.config.size ?? DEFAULT_ROBOT_RADIUS);
	}
	return {
		entry,
		replay,
		robotSpecies: new Map(replay.robots),
		speciesColors,
		speciesRadii
	};
}

export function replayDuration(scene: SimulatorScene): number {
	return Math.max(0, scene.replay.frames.length - 1) * scene.replay.interval;
}

export function sampleRobots(scene: SimulatorScene, seconds: number): RobotPose[] {
	const frames = scene.replay.frames;
	if (frames.length === 0) return [];
	const position = Math.min(frames.length - 1, Math.max(0, seconds / scene.replay.interval));
	const frameIndex = Math.floor(position);
	const blend = position - frameIndex;
	const current = frames[frameIndex];
	const next = frames[Math.min(frames.length - 1, frameIndex + 1)];
	const nextById = new Map<number, number>();
	for (let offset = 0; offset < next.length; offset += VALUES_PER_ROBOT) {
		nextById.set(next[offset], offset);
	}
	const robots: RobotPose[] = [];
	for (let offset = 0; offset < current.length; offset += VALUES_PER_ROBOT) {
		const id = current[offset];
		const nextOffset = nextById.get(id);
		const x = current[offset + 1];
		const y = current[offset + 2];
		const rotation = current[offset + 3];
		if (nextOffset === undefined || blend === 0) {
			robots.push({ id, x, y, rotation });
			continue;
		}
		robots.push({
			id,
			x: x + (next[nextOffset + 1] - x) * blend,
			y: y + (next[nextOffset + 2] - y) * blend,
			rotation: rotation + shortestDegrees(rotation, next[nextOffset + 3]) * blend
		});
	}
	return robots;
}

export function speciesCounts(scene: SimulatorScene, robots: RobotPose[]): Map<string, number> {
	const counts = new Map<string, number>();
	for (const robot of robots) {
		const speciesId = scene.robotSpecies.get(robot.id) ?? '';
		counts.set(speciesId, (counts.get(speciesId) ?? 0) + 1);
	}
	return counts;
}

export function drawScene(
	context: CanvasRenderingContext2D,
	scene: SimulatorScene,
	robots: RobotPose[],
	canvasWidth: number,
	canvasHeight: number,
	seconds: number
) {
	const { entry } = scene;
	const scale = Math.min(canvasWidth / entry.arena_width, canvasHeight / entry.arena_height);
	context.save();
	context.fillStyle = BACKGROUND;
	context.fillRect(0, 0, canvasWidth, canvasHeight);
	context.scale(scale, scale);

	drawGrid(context, entry.arena_width, entry.arena_height, scale);
	drawObstacles(context, scene);
	drawSpawnZones(context, scene, scale);
	drawRobots(context, scene, robots, scale);

	context.strokeStyle = BORDER;
	context.lineWidth = 4 / Math.max(scale, 0.25);
	context.strokeRect(0, 0, entry.arena_width, entry.arena_height);
	context.restore();

	context.fillStyle = HUD_TEXT;
	context.font = `${Math.max(11, Math.round(canvasWidth / 80))}px monospace`;
	context.fillText(`t=${seconds.toFixed(1)}s  robots=${robots.length}`, 10, Math.max(16, canvasWidth / 50));
}

function drawGrid(context: CanvasRenderingContext2D, width: number, height: number, scale: number) {
	context.strokeStyle = GRID;
	context.lineWidth = 1 / scale;
	context.beginPath();
	for (let x = 0; x <= width; x += PIXELS_PER_METER) {
		context.moveTo(x, 0);
		context.lineTo(x, height);
	}
	for (let y = 0; y <= height; y += PIXELS_PER_METER) {
		context.moveTo(0, y);
		context.lineTo(width, y);
	}
	context.stroke();
}

function drawObstacles(context: CanvasRenderingContext2D, scene: SimulatorScene) {
	context.lineWidth = 2;
	for (const obstacle of scene.entry.obstacles) {
		const [x, y] = obstacle.position;
		if (obstacle.type === 'wall') {
			const [width, height] = obstacle.size;
			context.fillStyle = WALL_FILL;
			context.strokeStyle = WALL_EDGE;
			context.fillRect(x - width / 2, y - height / 2, width, height);
			context.strokeRect(x - width / 2, y - height / 2, width, height);
		} else {
			context.beginPath();
			context.arc(x, y, obstacle.radius, 0, Math.PI * 2);
			context.fillStyle = CIRCLE_FILL;
			context.strokeStyle = CIRCLE_EDGE;
			context.fill();
			context.stroke();
		}
	}
}

function drawSpawnZones(context: CanvasRenderingContext2D, scene: SimulatorScene, scale: number) {
	context.lineWidth = 2 / Math.max(scale, 0.5);
	context.font = `${12 / Math.max(scale, 0.5)}px sans-serif`;
	for (const zone of scene.entry.spawn_zones) {
		const color = scene.speciesColors.get(zone.type_id) ?? UNKNOWN_SPECIES;
		const [centerX, centerY] = zone.position;
		const [width, height] = zone.size;
		const left = centerX - width / 2;
		const top = centerY - height / 2;
		context.fillStyle = `${color}${ZONE_FILL_ALPHA}`;
		context.fillRect(left, top, width, height);
		context.setLineDash([8, 6]);
		context.strokeStyle = color;
		context.strokeRect(left, top, width, height);
		context.setLineDash([]);
		context.fillStyle = color;
		context.fillText(zone.name, left + 6, top + 16 / Math.max(scale, 0.5));
	}
}

function drawRobots(context: CanvasRenderingContext2D, scene: SimulatorScene, robots: RobotPose[], scale: number) {
	const minimumRadius = MIN_ROBOT_RADIUS / scale;
	for (const robot of robots) {
		const speciesId = scene.robotSpecies.get(robot.id) ?? '';
		const color = scene.speciesColors.get(speciesId) ?? UNKNOWN_SPECIES;
		const radius = Math.max(minimumRadius, scene.speciesRadii.get(speciesId) ?? DEFAULT_ROBOT_RADIUS);
		const heading = (robot.rotation * Math.PI) / 180;
		context.beginPath();
		context.arc(robot.x, robot.y, radius, 0, Math.PI * 2);
		context.fillStyle = color;
		context.fill();
		context.beginPath();
		context.moveTo(robot.x, robot.y);
		context.lineTo(robot.x + Math.cos(heading) * radius * 1.8, robot.y + Math.sin(heading) * radius * 1.8);
		context.strokeStyle = color;
		context.lineWidth = Math.max(1 / scale, radius * 0.35);
		context.stroke();
	}
}

function shortestDegrees(from: number, to: number): number {
	return ((((to - from) % 360) + 540) % 360) - 180;
}
