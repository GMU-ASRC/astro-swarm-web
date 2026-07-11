export interface EvalSummary {
	trials: number;
	success_rate: number;
	outcomes: string[];
	detection_times?: number[];
	capture_times?: number[];
	sweep?: { n: number; success_rate: number }[];
}

export interface PlacementInfo {
	x: number;
	y: number;
	rot: number;
}

export interface PlayerEvaluation {
	id: string;
	player_id: string;
	username: string;
	level_id: string;
	level_number?: number;
	is_attack?: boolean;
	game_version?: string;
	defender_count?: number;
	algorithm: any[];
	placements: PlacementInfo[];
	status: string;
	progress: number;
	trials: number;
	results: EvalSummary;
	attacker_rate?: number | null;
	defender_rate?: number | null;
	xp_awarded?: number | null;
	error: string | null;
	created_at: string;
	completed_at: string | null;
}

export interface PlayerListItem {
	id: string;
	player_id: string;
	username: string;
	level_id: string;
	level_number?: number;
	is_attack?: boolean;
	game_version?: string;
	defender_count?: number;
	status: string;
	progress: number;
	trials: number;
	success_rate: number | null;
	attacker_rate?: number | null;
	defender_rate?: number | null;
	xp_awarded?: number | null;
	created_at: string;
	completed_at: string | null;
}

export interface BaselineResult {
	success_rate: number | null;
	samples: number;
}

export interface ReplayRunInfo {
	trial: number;
	outcome: string;
}

export interface ReplayIndex {
	fps: number;
	defenders: number;
	planet: [number, number, number] | null;
	arena: [number, number] | null;
	runs: ReplayRunInfo[];
}

export interface Replay {
	trial?: number;
	n?: number;
	outcome: string;
	detection_time?: number;
	capture_time?: number;
	fps: number;
	defenders: number;
	view: number;
	fov: number;
	planet: [number, number, number];
	arena: [number, number];
	frames: number[][];
}
