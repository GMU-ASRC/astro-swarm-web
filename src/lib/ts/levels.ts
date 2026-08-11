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
		subtitle: 'Pilot',
		piloted: true,
		variant: 'red',
		summary:
			'You fly the evader yourself against the best submitted Level 2 algorithm. Each entry is a single recorded flight rather than a benchmark.',
		rateLabel: 'goal reached'
	},
	{
		number: 4,
		id: 'farp4',
		slug: '4',
		name: 'Level 4',
		subtitle: 'Swarm',
		piloted: true,
		variant: 'purple',
		summary:
			'Two milling swarms and one human-flown leader. Merge the groups, walk the mill onto the planet and leave without breaking it. Each entry is a recorded flight scored on a milling loss.',
		rateLabel: 'swarm delivered'
	}
];

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
