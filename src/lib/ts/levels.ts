import type { ShipVariant } from '$lib/ts/ships';

export interface LevelInfo {
	number: number;
	id: string;
	slug: string;
	name: string;
	subtitle: string;
	tagline: string;
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
		tagline: 'Place up to twenty defenders and spot ten incoming evaders.',
		piloted: false,
		variant: 'blue',
		summary:
			'Place up to twenty defenders around the planet and program how they sweep. Ten evaders come in one after another, and any evader a defender spots vanishes. Scored on the share spotted, with the average circliness of the defenders mill by mill.',
		rateLabel: 'spot rate'
	},
	{
		number: 2,
		id: 'farp2',
		slug: '2',
		name: 'Level 2',
		subtitle: 'Scatter',
		tagline: 'Scatter a swarm of defenders and capture two evaders per defender by touch.',
		piloted: false,
		variant: 'gold',
		summary:
			'Field 5, 25, 50, 100, 250 or 500 defenders scattered at random in a larger arena. After a ten second head start, two evaders per defender come in one after another and each has to be captured by touch. Scored on the share captured, with the average circliness of the defenders mill by mill.',
		rateLabel: 'capture rate'
	},
	{
		number: 3,
		id: 'farp3',
		slug: '3',
		name: 'Level 3',
		subtitle: 'Waves',
		tagline: 'Hold two waves, each sending one evader per defender at once.',
		piloted: false,
		variant: 'green',
		summary:
			'Field 5, 25, 50, 100, 250 or 500 defenders. Two waves come in, each sending one evader per defender all at once around the ring. A defender that catches one survives and takes the next, so nothing is spent. The score is the share of evaders destroyed.',
		rateLabel: 'capture success rate'
	},
	{
		number: 4,
		id: 'farp4',
		slug: '4',
		name: 'Level 4',
		subtitle: 'Attrition',
		tagline: 'Every capture costs a defender, so make each trade count.',
		piloted: false,
		variant: 'red',
		summary:
			'Five to twenty scattered defenders face evaders one at a time, each from a fresh bearing, and a capture destroys the defender as well as the evader. Every kill costs a body, and the run ends when the line is spent. The level plays five waves; the benchmark keeps sending them until nothing is left to stop them.',
		rateLabel: 'capture success rate'
	},
	{
		number: 5,
		id: 'farp5',
		slug: '5',
		name: 'Level 5',
		subtitle: 'Siege',
		tagline: 'Five evaders strike at once and five defenders must stop them all.',
		piloted: false,
		variant: 'purple',
		summary:
			'No waves. Five evaders spawn at once around the edges of the arena and all drive at the planet, arriving in a stagger. Attrition is on, so five defenders have to make five clean trades.',
		rateLabel: 'capture success rate'
	},
	{
		number: 6,
		id: 'farp6',
		slug: '6',
		name: 'Level 6',
		subtitle: 'Pilot',
		tagline: 'Fly the evader yourself against the best Level 2 algorithm.',
		piloted: true,
		variant: 'red',
		summary:
			'You fly the evader yourself against the best submitted Level 2 algorithm. Each entry is a single recorded flight rather than a benchmark.',
		rateLabel: 'goal reached'
	},
	{
		number: 7,
		id: 'farp7',
		slug: '7',
		name: 'Level 7',
		subtitle: 'Swarm',
		tagline: 'Lead two milling swarms together and walk the mill onto the planet.',
		piloted: true,
		variant: 'purple',
		summary:
			'Two milling swarms and one human-flown leader. Merge the groups, walk the mill onto the planet and leave without breaking it. Each entry is a recorded flight scored on a milling loss.',
		rateLabel: 'swarm delivered'
	},
	{
		number: 8,
		id: 'farp8',
		slug: '8',
		name: 'Level 8',
		subtitle: 'Supply',
		tagline: 'Fly a shuttle to split defenders between two planets under attack.',
		piloted: true,
		variant: 'gold',
		summary:
			'Two planets on two separate seeds. Ten defenders start on planet A and planet B starts empty, and both are attacked by five evaders each. The defenders run the submitted algorithm during the deployment window too, and the player flies a shuttle their sensors read as an ally, steering them with their own sees-ally rule and calling them across between the planets before the window closes, then both assaults play out back to back on that split with the shuttle still flying. Each entry is a recorded flight scored on the share of evaders stopped across the pair.',
		rateLabel: 'evaders stopped'
	}
];

export const PILOT_LEVELS = [6, 7, 8];
export const ASSAULT_LEVELS = [1, 2, 3, 4, 5];
export const STREAM_LEVELS = [1, 2];
export const ATTRITION_LEVELS = [4, 5];
export const SWARM_LEVEL = 7;
export const SUPPLY_LEVEL = 8;

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

export function isSupply(level: number): boolean {
	return level === SUPPLY_LEVEL;
}

export function isAssault(level: number): boolean {
	return ASSAULT_LEVELS.includes(level);
}

export function isStream(level: number): boolean {
	return STREAM_LEVELS.includes(level);
}

export function hasAttrition(level: number): boolean {
	return ATTRITION_LEVELS.includes(level);
}

export function levelName(level: number): string {
	if (isSupply(level)) return `Level ${level} · Supply`;
	if (isSwarm(level)) return `Level ${level} · Swarm`;
	if (isPilot(level)) return `Level ${level} · Evasion`;
	if (level === 5) return `Level ${level} · Siege`;
	if (isStream(level)) return `Level ${level} · ${level === 2 ? 'Scatter' : 'Defense'}`;
	if (isAssault(level)) return `Level ${level} · Waves`;
	return `Level ${level} · Defense`;
}

export function rateWord(level: number): string {
	if (isSupply(level)) return 'detection';
	return isPilot(level) ? 'evasion' : 'detection';
}
