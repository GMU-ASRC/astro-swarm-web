import type { PageLoad } from './$types';
import type { LevelLeaderboardRow, LevelSweepEntry, PlayerListItem } from '$lib/ts/evaluation';
import { apiUrl } from '$lib/ts/api';
import { levelBySlug } from '$lib/ts/levels';
import { error } from '@sveltejs/kit';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = ({ params, fetch }) => {
	const level = levelBySlug(params.level);
	if (!level) {
		error(404, 'Level not found');
	}

	const playersPromise = (async () => {
		try {
			const res = await fetch(apiUrl('/api/evaluations?exclude_cancelled=1'));
			if (res.ok) {
				return { players: (await res.json()) as PlayerListItem[], apiError: false };
			}
			console.error('Failed to fetch players:', await res.text());
			return { players: [] as PlayerListItem[], apiError: true };
		} catch (err) {
			console.error('Error fetching players:', err);
			return { players: [] as PlayerListItem[], apiError: true };
		}
	})();

	const leaderboardPromise = (async () => {
		try {
			const res = await fetch(apiUrl(`/api/evaluations/level-leaderboard?level_id=${level.id}`));
			if (res.ok) return (await res.json()) as LevelLeaderboardRow[];
			console.error('Failed to fetch level leaderboard:', await res.text());
		} catch (err) {
			console.error('Error fetching level leaderboard:', err);
		}
		return [] as LevelLeaderboardRow[];
	})();

	const comparisonPromise = (async () => {
		try {
			const res = await fetch(apiUrl(`/api/evaluations/level-sweep?level_id=${level.id}`));
			if (res.ok) return (await res.json()) as LevelSweepEntry[];
			console.error('Failed to fetch level comparison:', await res.text());
		} catch (err) {
			console.error('Error fetching level comparison:', err);
		}
		return [] as LevelSweepEntry[];
	})();

	return { level, playersPromise, leaderboardPromise, comparisonPromise };
};
