import { apiUrl } from '$lib/ts/api';

export interface AdminLevel {
	id: string;
	name: string;
	enabled: boolean;
}

export interface DerivedSeed {
	name: string;
	formula: string;
	value: string | number;
}

export interface TrialSeed {
	trial: number;
	seed: number;
}

export interface RunLimits {
	max_seconds: number;
	max_ships: number;
	frame_slots: number | null;
}

export interface AdminSettings {
	seed: number;
	sweep_max: number;
	sweep_trials: number;
	match_cap_seconds: number;
	goal_tail_seconds: number;
	enemy_start_x: number;
	enemy_start_y: number;
	arena_width: number;
	arena_height: number;
	planet_x: number;
	planet_y: number;
	sweep_trial_seeds: TrialSeed[];
	derived_seeds: DerivedSeed[];
	levels: AdminLevel[];
	pilot_level_ids: string[];
	pilot_run_limits: Record<string, RunLimits>;
	pilot_max_fps: number;
	pilot_max_xp: number;
	required_game_version: string;
	assault_levels: string[];
	attrition_levels: string[];
	stream_levels: string[];
	assault_trials: number;
	assault_clean_streak: number;
	assault_sweep_max: number;
	assault_sweep_trials: number;
	stream_sweep_max: number;
	siege_evaders: number;
}

export type SaveSettings = (body: Record<string, unknown>, successMessage: string) => Promise<boolean>;

export async function putSettings(
	adminKey: string,
	body: Record<string, unknown>
): Promise<Partial<AdminSettings>> {
	const res = await fetch(apiUrl('/api/evaluations/settings'), {
		method: 'PUT',
		headers: { 'X-API-Key': adminKey, 'Content-Type': 'application/json' },
		body: JSON.stringify(body)
	});
	const payload = await res.json().catch(() => null);
	if (!res.ok) throw new Error(payload?.error ?? `HTTP ${res.status}`);
	return payload ?? {};
}

export type LevelKind = 'stream' | 'assault' | 'recorded';

export interface LevelProfile {
	number: number;
	kind: LevelKind;
	attrition: boolean;
	arrival: string;
	endsWhen: string;
	budget: string;
}

export const KIND_LABELS: Record<LevelKind, string> = {
	stream: 'Stream',
	assault: 'Assault',
	recorded: 'Recorded run'
};

export function levelNumberOf(levelId: string): number {
	const digits = (levelId ?? '').match(/\d+/);
	return digits ? Number(digits[0]) : 1;
}

function kindOf(levelId: string, settings: AdminSettings): LevelKind {
	if (settings.pilot_level_ids.includes(levelId)) return 'recorded';
	if (settings.stream_levels.includes(levelId)) return 'stream';
	return 'assault';
}

function arrivalOf(number: number, settings: AdminSettings): string {
	switch (number) {
		case 1:
			return '10 evaders, 3 s apart, removed once seen';
		case 2:
			return '2 × D evaders, one every 15 / D s, after a 10 s head start';
		case 3:
			return '2 waves of D evaders, each wave all at once';
		case 4:
			return 'One at a time, each from a fresh bearing';
		case 5:
			return `${settings.siege_evaders} at once off the arena edges`;
		case 6:
			return 'The player flies the evader';
		case 7:
			return 'The player herds two milling swarms';
		case 8:
			return 'The player splits one force across two planets';
	}
	return '—';
}

function endsWhenOf(number: number, kind: LevelKind, attrition: boolean, limits?: RunLimits): string {
	if (kind === 'recorded') return limits ? `Recorded flight, up to ${limits.max_seconds} s` : 'Recorded flight';
	if (kind === 'stream') return 'Every evader is stopped or reaches the planet';
	if (number === 3) return 'Both waves are resolved, or the clock stops';
	if (number === 5) return 'Every evader is resolved';
	return attrition ? 'The line is spent, or the clock stops' : 'The clock stops';
}

function budgetOf(kind: LevelKind, settings: AdminSettings): string {
	if (kind === 'recorded') return '1 render job';
	const trials = `${settings.assault_trials} trials`;
	const perSize = `${settings.assault_sweep_trials} per size`;
	if (kind === 'stream') return `${trials} + sweep n = 1–${settings.stream_sweep_max}, ${perSize}`;
	return `${trials} + sweep up to n = ${settings.assault_sweep_max}, ${perSize}, stops after ${settings.assault_clean_streak} clean`;
}

export function levelProfile(level: AdminLevel, settings: AdminSettings): LevelProfile {
	const number = levelNumberOf(level.id);
	const kind = kindOf(level.id, settings);
	const attrition = settings.attrition_levels.includes(level.id);
	return {
		number,
		kind,
		attrition,
		arrival: arrivalOf(number, settings),
		endsWhen: endsWhenOf(number, kind, attrition, settings.pilot_run_limits[level.id]),
		budget: budgetOf(kind, settings)
	};
}
