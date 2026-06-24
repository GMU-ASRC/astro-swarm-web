export interface EvalPoint {
	n: number;
	success_rate: number;
}

export interface EvalSummary {
	trials: number;
	success_rate: number;
	outcomes: string[];
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
	algorithm: any[];
	placements: PlacementInfo[];
	status: string;
	progress: number;
	trials: number;
	results: EvalSummary;
	error: string | null;
	created_at: string;
	completed_at: string | null;
}

export interface PlayerListItem {
	id: string;
	player_id: string;
	username: string;
	status: string;
	progress: number;
	trials: number;
	success_rate: number | null;
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
	trial: number;
	outcome: string;
	fps: number;
	defenders: number;
	view: number;
	fov: number;
	planet: [number, number, number];
	arena: [number, number];
	frames: number[][];
}
