import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) => {
	const leaderboardPromise = (async () => {
		const res = await fetch(apiUrl('/api/leaderboard'));
		return res.ok ? await res.json() : [];
	})();

	return { leaderboardPromise };
};
