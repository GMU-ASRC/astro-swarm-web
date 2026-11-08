export interface TimingSummary {
	key: string;
	count: number;
	average_ms: number;
	p50_ms: number;
	p95_ms: number;
	max_ms: number;
	last_at: string;
}

export interface DailyCount {
	date: string;
	submitted: number;
	done: number;
	players: number;
}

export interface LevelStat {
	level_id: string;
	entries: number;
	players: number;
	done: number;
	best_rate: number | null;
	average_rate: number | null;
	last_entry: string | null;
}

export interface AdminStats {
	generated_at: string;
	entries: {
		total: number;
		last_24h: number;
		last_7d: number;
		players: number;
		active_players: number;
		benchmarks: number;
		shared_results: number;
		average_success: number | null;
		turnaround_s_7d: number | null;
		benchmark_runs: number;
		by_status: Record<string, number>;
	};
	daily: DailyCount[];
	levels: LevelStat[];
	workers: { total: number; enabled: number; online: number; busy: number };
	content: Record<string, number>;
	timings: { since: string; requests: TimingSummary[]; page_loads: TimingSummary[] };
}

export const STATUS_COLORS: Record<string, string> = {
	done: 'var(--color-win)',
	running: 'var(--color-brand)',
	queued: 'var(--color-brand-hover)',
	failed: 'var(--color-loss)',
	error: 'var(--color-loss)',
	cancelled: 'var(--color-faint)'
};

export const STATUS_ORDER = ['done', 'running', 'queued', 'failed', 'error', 'cancelled'];
