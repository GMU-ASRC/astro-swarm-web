import type { PageLoad } from './$types';
import { fetchTrainingLeaderboard } from '$lib/ts/training';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = ({ fetch }) => {
	return { leaderboardPromise: fetchTrainingLeaderboard(fetch) };
};
