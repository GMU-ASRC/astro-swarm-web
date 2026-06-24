export interface EvalPoint {
	n: number;
	success_rate: number;
}

export interface PlayerEvaluation {
	id: string;
	player_id: string;
	username: string;
	algorithm: any[];
	status: string;
	n_max: number;
	trials: number;
	results: EvalPoint[];
	error: string | null;
	created_at: string;
	completed_at: string | null;
}

export interface PlayerListItem {
	id: string;
	player_id: string;
	username: string;
	status: string;
	n_max: number;
	trials: number;
	created_at: string;
	completed_at: string | null;
}

export interface BaselineResult {
	results: EvalPoint[];
	samples: number;
}
