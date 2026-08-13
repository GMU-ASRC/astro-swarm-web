export const PILOT_LEVELS = [5, 6];
export const WAVE_LEVELS = [3, 4];
export const SWARM_LEVEL = 6;

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

export function rateLabel(level: number): string {
	return isPilot(level) ? 'evasion' : 'detection';
}
