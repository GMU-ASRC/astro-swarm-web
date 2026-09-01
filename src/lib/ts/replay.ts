export type ReplayTone = 'win' | 'loss' | 'timeout';

export interface ReplayCell {
	id: string | number;
	label: string | number;
	tone: ReplayTone;
	title: string;
	selected: boolean;
	replayable?: boolean;
	select: () => void;
}

export interface ReplayGroup {
	label: string;
	cells: ReplayCell[];
}

export function toneOf(outcome: string): ReplayTone {
	if (outcome === 'win') return 'win';
	if (outcome === 'lose') return 'loss';
	return 'timeout';
}
