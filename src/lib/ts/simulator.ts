import type { BlockNames, BlockScript } from '$lib/ts/blockDefs';

export interface SimulatorSpeciesSummary {
	id: string;
	name: string;
	color: string;
}

export interface SimulatorSpeciesConfig {
	speed?: number;
	turn_rate?: number;
	vision?: number;
	fov?: number;
	size?: number;
}

export interface SimulatorSpecies extends SimulatorSpeciesSummary {
	config: SimulatorSpeciesConfig;
}

export interface SimulatorVariable {
	name: string;
	type: 'int' | 'string';
}

export type SimulatorObstacle =
	| { type: 'wall'; position: [number, number]; size: [number, number] }
	| { type: 'circle'; position: [number, number]; radius: number };

export interface SimulatorSpawnZone {
	id: number;
	name: string;
	position: [number, number];
	size: [number, number];
	type_id: string;
	enabled: boolean;
}

export interface SimulatorEntryListItem {
	id: string;
	player_id: string;
	username: string;
	title: string;
	description: string;
	game_version: string;
	species: SimulatorSpeciesSummary[];
	placement_count: number;
	peak_robot_count: number;
	spawn_zone_count: number;
	duration_seconds: number;
	created_at: string;
}

export interface SimulatorEntry extends Omit<SimulatorEntryListItem, 'species'> {
	arena_width: number;
	arena_height: number;
	species: SimulatorSpecies[];
	behaviors: Record<string, BlockScript[]>;
	arena_program: BlockScript[];
	variables: SimulatorVariable[];
	obstacles: SimulatorObstacle[];
	spawn_zones: SimulatorSpawnZone[];
	frame_count: number;
	record_interval: number;
}

export interface SimulatorReplay {
	interval: number;
	robots: [number, string][];
	frames: number[][];
}

export const PIXELS_PER_METER = 40;
export const VALUES_PER_ROBOT = 4;

export function blockNamesFor(entry: SimulatorEntry): BlockNames {
	const species: Record<string, string> = {};
	for (const item of entry.species) species[item.id] = item.name;
	const zones: Record<string, string> = {};
	for (const zone of entry.spawn_zones) zones[String(zone.id)] = zone.name;
	return { species, zones };
}

export function speciesConfigRows(config: SimulatorSpeciesConfig): { label: string; value: string }[] {
	const rows: { label: string; value: string }[] = [];
	if (config.speed != null) rows.push({ label: 'Speed', value: `${config.speed.toFixed(2)} m/s` });
	if (config.turn_rate != null) rows.push({ label: 'Turn rate', value: `${Math.round(config.turn_rate)}°/s` });
	if (config.vision != null) rows.push({ label: 'Vision range', value: `${config.vision.toFixed(2)} m` });
	if (config.fov != null) rows.push({ label: 'Field of view', value: `${Math.round(config.fov)}°` });
	if (config.size != null) rows.push({ label: 'Size', value: `${config.size} px` });
	return rows;
}

export function durationLabel(seconds: number): string {
	const minutes = Math.floor(seconds / 60);
	const remainder = Math.floor(seconds % 60);
	return `${minutes}:${String(remainder).padStart(2, '0')}`;
}

export function shortDate(iso: string): string {
	const date = new Date(iso);
	const month = String(date.getMonth() + 1).padStart(2, '0');
	const day = String(date.getDate()).padStart(2, '0');
	return `${month}/${day}/${date.getFullYear()}`;
}
