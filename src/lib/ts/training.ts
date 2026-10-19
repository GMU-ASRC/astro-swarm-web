import { apiUrl } from '$lib/ts/api';

export interface TrainingScoreRow {
	id: string;
	rank: number;
	player_id: string;
	username: string;
	circliness: number;
	defenders: number;
	game_version: string;
	updated_at: string | null;
}

export async function fetchTrainingLeaderboard(
	fetcher: typeof fetch
): Promise<{ rows: TrainingScoreRow[]; apiError: boolean }> {
	try {
		const res = await fetcher(apiUrl('/api/training/leaderboard'));
		if (res.ok) return { rows: (await res.json()) as TrainingScoreRow[], apiError: false };
		console.error('Failed to fetch training leaderboard:', await res.text());
	} catch (err) {
		console.error('Error fetching training leaderboard:', err);
	}
	return { rows: [], apiError: true };
}

export function circlinessText(value: number): string {
	return value.toFixed(3);
}
