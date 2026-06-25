import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const res = await fetch(apiUrl('/api/leaderboard'));
	const leaderboard = res.ok ? await res.json() : [];
	return { leaderboard };
};
