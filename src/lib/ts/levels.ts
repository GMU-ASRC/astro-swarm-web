import type { ShipVariant } from '$lib/ts/ships';

export interface LevelInfo {
	number: number;
	id: string;
	slug: string;
	name: string;
	subtitle: string;
	piloted: boolean;
	variant: ShipVariant;
	summary: string;
	rateLabel: string;
}

export const FARP_LEVELS: LevelInfo[] = [
	{
		number: 1,
		id: 'farp1',
		slug: '1',
		name: 'Level 1',
		subtitle: 'Defense',
		piloted: false,
		variant: 'blue',
		summary:
			'Ring the planet with defenders and program how they sweep for the incoming evader. Every submission is replayed headlessly over many trials.',
		rateLabel: 'capture rate'
	},
	{
		number: 2,
		id: 'farp2',
		slug: '2',
		name: 'Level 2',
		subtitle: 'Defense',
		piloted: false,
		variant: 'gold',
		summary:
			'The same defensive problem under tighter conditions. Submissions are benchmarked the same way, so Level 1 and Level 2 scores stay comparable.',
		rateLabel: 'capture rate'
	},
	{
		number: 3,
		id: 'farp3',
		slug: '3',
		name: 'Level 3',
		subtitle: 'Waves',
		piloted: false,
		variant: 'green',
		summary:
			'Several evaders come in off the ring, one after another or all at once. A defender that touches one destroys it and carries on. A run only counts if every evader is stopped.',
		rateLabel: 'waves held'
	},
	{
		number: 4,
		id: 'farp4',
		slug: '4',
		name: 'Level 4',
		subtitle: 'Trade',
		piloted: false,
		variant: 'red',
		summary:
			'The same waves, but a capture destroys the defender as well as the evader. Every kill costs a body, so the line thins as the wave goes on.',
		rateLabel: 'waves held'
	},
	{
		number: 5,
		id: 'farp5',
		slug: '5',
		name: 'Level 5',
		subtitle: 'Pilot',
		piloted: true,
		variant: 'red',
		summary:
			'You fly the evader yourself against the best submitted Level 2 algorithm. Each entry is a single recorded flight rather than a benchmark.',
		rateLabel: 'goal reached'
	},
	{
		number: 6,
		id: 'farp6',
		slug: '6',
		name: 'Level 6',
		subtitle: 'Swarm',
		piloted: true,
		variant: 'purple',
		summary:
			'Two milling swarms and one human-flown leader. Merge the groups, walk the mill onto the planet and leave without breaking it. Each entry is a recorded flight scored on a milling loss.',
		rateLabel: 'swarm delivered'
	}
];

export const PILOT_LEVELS = [5, 6];
export const WAVE_LEVELS = [3, 4];
export const SWARM_LEVEL = 6;

export function canonicalLevelId(levelId: string): string {
	if (!levelId || levelId === 'farp') return 'farp1';
	return levelId;
}

export function levelBySlug(slug: string): LevelInfo | undefined {
	return FARP_LEVELS.find((level) => level.slug === slug);
}

export function levelById(levelId: string): LevelInfo | undefined {
	const canonical = canonicalLevelId(levelId);
	return FARP_LEVELS.find((level) => level.id === canonical);
}

export function levelNumber(levelId: string | null | undefined): number {
	const digits = (levelId ?? '').replace(/\D/g, '');
	return digits === '' ? 1 : Number(digits);
}

export function isPilot(level: number): boolean {
	return PILOT_LEVELS.includes(level);
}

export function isSwarm(level: number): boolean {
	return level === SWARM_LEVEL;
}

export function isWave(level: number): boolean {
	return WAVE_LEVELS.includes(level);
}

export function levelName(level: number): string {
	if (isSwarm(level)) return `Level ${level} · Swarm`;
	if (isPilot(level)) return `Level ${level} · Evasion`;
	if (isWave(level)) return `Level ${level} · Waves`;
	return `Level ${level} · Defense`;
}

export function rateWord(level: number): string {
	return isPilot(level) ? 'evasion' : 'detection';
}
