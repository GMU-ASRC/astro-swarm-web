export type BlockCategory = 'config' | 'condition' | 'logic' | 'variable' | 'spawn' | 'action';

export interface Block {
	type: string;
	params?: Record<string, unknown>;
	children?: Block[];
}

export interface BlockScript {
	x?: number;
	y?: number;
	blocks?: Block[];
}

export interface BlockDef {
	label: string;
	category: BlockCategory;
	parts?: string[];
	suffix?: string;
	step?: number;
}

export interface BlockNames {
	species?: Record<string, string>;
	zones?: Record<string, string>;
}

export type BlockPart = { kind: 'text'; text: string } | { kind: 'value'; text: string };

export const CATEGORY_COLORS: Record<BlockCategory, { background: string; edge: string }> = {
	config: { background: '#46a952', edge: '#35843f' },
	condition: { background: '#7b4daf', edge: '#5c378a' },
	logic: { background: '#d98533', edge: '#ae6823' },
	variable: { background: '#339e99', edge: '#22736f' },
	spawn: { background: '#c94f80', edge: '#9e375f' },
	action: { background: '#4176d7', edge: '#2e58ae' }
};

const BLOCK_DEFS: Record<string, BlockDef> = {
	set_speed: { label: 'Set speed to', category: 'config', suffix: ' m/s', step: 0.05 },
	set_turn: { label: 'Set turn rate to', category: 'config', suffix: '°/s', step: 5 },
	set_view: { label: 'Set vision range to', category: 'config', suffix: ' m', step: 0.1 },
	set_fov: { label: 'Set FOV to', category: 'config', suffix: '°', step: 1 },
	set_size: { label: 'Set size to', category: 'config', suffix: ' px', step: 0.5 },

	when_start: { label: 'On start', category: 'condition' },
	when_always: { label: 'Always', category: 'condition' },
	when_every: { label: 'Every', category: 'condition', parts: ['Every', '$value:seconds', 'seconds'] },
	when_sees: { label: 'When I see anyone', category: 'condition' },
	when_alone: { label: 'When I see nobody', category: 'condition' },
	when_near_wall: { label: 'When I touch a wall', category: 'condition' },
	when_sees_wall: { label: 'When I see a wall', category: 'condition' },
	when_sees_species: { label: 'When I see a', category: 'condition', parts: ['When I see a', '$value:species'] },
	when_no_sees_species: { label: "When I don't see a", category: 'condition', parts: ["When I don't see a", '$value:species'] },
	when_sees_enemy: { label: 'When I see an enemy', category: 'condition' },
	when_sees_ally: { label: 'When I see an ally', category: 'condition' },
	when_sees_object: { label: 'When I see an object', category: 'condition' },
	when_sees_rim: { label: 'When I see the outer rim', category: 'condition' },

	if_sees: { label: 'If I see anyone', category: 'logic' },
	if_sees_species: { label: 'If I see a', category: 'logic', parts: ['If I see a', '$value:species'] },
	if_within: { label: 'If target within', category: 'logic', suffix: ' m', step: 0.1 },
	if_beyond: { label: 'If target beyond', category: 'logic', suffix: ' m', step: 0.1 },
	if_see: { label: 'If I see', category: 'logic', parts: ['If I see', '$target:target'] },
	if_compare: { label: 'If', category: 'logic', parts: ['If', '$var', '$op:operator', '$value:number'] },
	if_zone_count: {
		label: 'If',
		category: 'logic',
		parts: ['If', '$zone:zone', 'has', '$op:operator', '$value:number', 'robots']
	},
	else: { label: 'Else', category: 'logic' },

	set_var: { label: 'Set', category: 'variable', parts: ['Set', '$var', 'to', '$value:number'] },
	set_var_random: {
		label: 'Set',
		category: 'variable',
		parts: ['Set', '$var', 'to random', '$min:number', 'to', '$max:number']
	},

	do_spawn: { label: 'Spawn', category: 'spawn', parts: ['Spawn', '$count:number', 'robots in', '$zone:zone'] },
	do_zone_species: {
		label: 'Set',
		category: 'spawn',
		parts: ['Set', '$zone:zone', 'species to', '$species:species']
	},
	do_zone_toggle: { label: 'Turn', category: 'spawn', parts: ['Turn', '$zone:zone', '$state'] },

	do_forward: { label: 'Move forward', category: 'action' },
	do_backward: { label: 'Move backward', category: 'action' },
	do_stop: { label: 'Stop', category: 'action' },
	do_wander: { label: 'Wander randomly', category: 'action' },
	do_random_walk: { label: 'Random walk', category: 'action' },
	do_turn_left: { label: 'Turn left at', category: 'action', suffix: '°/s', step: 5 },
	do_turn_right: { label: 'Turn right at', category: 'action', suffix: '°/s', step: 5 },
	do_turn_left_by: { label: 'Turn left by', category: 'action', suffix: '°', step: 1 },
	do_turn_right_by: { label: 'Turn right by', category: 'action', suffix: '°', step: 1 },
	do_face: { label: 'Face the target', category: 'action' },
	do_flee: { label: 'Flee the target', category: 'action' },
	do_fire: { label: 'Fire', category: 'action' },
	do_throttle: { label: 'Throttle to', category: 'action', suffix: '×', step: 0.05 },
	do_stop_sim: { label: 'Stop simulation', category: 'action' },
	do_pause_sim: { label: 'Pause simulation', category: 'action' }
};

const OPERATOR_SYMBOLS: Record<string, string> = { '!=': '≠', '<=': '≤', '>=': '≥' };

const TARGET_LABELS: Record<string, string> = {
	anyone: 'anyone',
	enemy: 'an enemy',
	ally: 'an ally',
	object: 'an object',
	wall: 'a wall'
};

export function blockDef(type: string): BlockDef {
	return BLOCK_DEFS[type] ?? { label: type.replace(/_/g, ' '), category: 'action' };
}

export function isContainerBlock(type: string): boolean {
	return type.startsWith('when_') || type.startsWith('if_') || type === 'else';
}

export function blockParts(block: Block, names: BlockNames = {}): BlockPart[] {
	const def = blockDef(block.type);
	const params = block.params ?? {};
	const template = def.parts ?? [def.label, '$value:single'];
	const parts: BlockPart[] = [];
	for (const piece of template) {
		if (!piece.startsWith('$')) {
			parts.push({ kind: 'text', text: piece });
			continue;
		}
		const [key, format = 'plain'] = piece.slice(1).split(':');
		const value = params[key];
		if (value === undefined || value === null || value === '') continue;
		parts.push({ kind: 'value', text: formatValue(value, format, def, names) });
	}
	return parts;
}

function formatValue(value: unknown, format: string, def: BlockDef, names: BlockNames): string {
	switch (format) {
		case 'single':
			if (typeof value !== 'number') return names.species?.[String(value)] ?? String(value);
			return `${value.toFixed((def.step ?? 1) >= 1 ? 0 : 1)}${def.suffix ?? ''}`;
		case 'seconds':
			return typeof value === 'number' ? trimNumber(value) : String(value);
		case 'number':
			return typeof value === 'number' ? trimNumber(value) : String(value);
		case 'species':
			return names.species?.[String(value)] ?? String(value);
		case 'zone':
			return names.zones?.[String(value)] ?? `Zone ${Number(value) + 1}`;
		case 'operator':
			return OPERATOR_SYMBOLS[String(value)] ?? String(value);
		case 'target':
			return TARGET_LABELS[String(value)] ?? `a ${names.species?.[String(value)] ?? String(value)}`;
		default:
			return String(value);
	}
}

function trimNumber(value: number): string {
	return Number.isInteger(value) ? String(value) : String(Math.round(value * 100) / 100);
}
