import type { ChartConfiguration } from 'chart.js';

const GRID = '#e5e7eb';
const TEXT = '#374151';

export const PLAYER_COLORS = ['#a16207', '#0d9488'];

export interface SurvivePlayer {
	slot: number;
	name: string;
	evaders_through: number;
	defenders_at_end: number;
	herded_total: number;
	freezes_used: number;
	actions: number;
	apm_average: number;
	apm_peak: number;
}

export interface SurviveMatchListItem {
	id: string;
	username: string;
	game_version: string;
	outcome: string;
	winner: string | null;
	duration: number;
	player1: string;
	player2: string;
	evaders_player1: number;
	evaders_player2: number;
	apm_player1: number;
	apm_player2: number;
	created_at: string;
}

export interface SurviveMatch extends SurviveMatchListItem {
	player_id: string;
	players: SurvivePlayer[];
	apm_bucket_seconds: number;
	apm_times: number[];
	apm_series_player1: number[];
	apm_series_player2: number[];
	evaders_spawned: number;
	wild_remaining: number;
	first_target: number;
}

export function clockLabel(seconds: number): string {
	const mm = Math.floor(seconds / 60);
	const ss = Math.floor(seconds % 60);
	return `${mm}:${String(ss).padStart(2, '0')}`;
}

export function apmChartConfig(match: SurviveMatch): ChartConfiguration {
	const labels = match.apm_times.map(clockLabel);
	const series = [
		{ name: match.player1, data: match.apm_series_player1 },
		{ name: match.player2, data: match.apm_series_player2 }
	];

	return {
		type: 'line',
		data: {
			labels,
			datasets: series.map((entry, index) => ({
				label: entry.name,
				data: entry.data,
				borderColor: PLAYER_COLORS[index],
				backgroundColor: PLAYER_COLORS[index],
				pointRadius: 0,
				pointHoverRadius: 5,
				borderWidth: 2,
				tension: 0.25
			}))
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			interaction: { mode: 'index', intersect: false },
			plugins: {
				title: {
					display: true,
					text: `Actions per minute (${match.apm_bucket_seconds}s samples)`,
					color: TEXT,
					font: { size: 15 }
				},
				legend: { display: true, labels: { color: TEXT, usePointStyle: true } },
				tooltip: { enabled: true }
			},
			scales: {
				y: {
					beginAtZero: true,
					title: { display: true, text: 'APM', color: TEXT },
					ticks: { color: TEXT },
					grid: { color: GRID }
				},
				x: {
					title: { display: true, text: 'Match time', color: TEXT },
					ticks: { color: TEXT, maxTicksLimit: 12 },
					grid: { color: GRID }
				}
			}
		}
	};
}
