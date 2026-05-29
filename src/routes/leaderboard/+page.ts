import type { PageLoad } from './$types';
import type { LeaderboardEntry } from '$lib/ts/leaderboard';
import { apiUrl } from '$lib/ts/api';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
	let entries: LeaderboardEntry[] = [];
	let apiError = false;

	try {
		const res = await fetch(apiUrl('/api/leaderboard'));
		if (res.ok) {
			entries = await res.json();
		} else {
			apiError = true;
			console.error('Failed to fetch leaderboard:', await res.text());
		}
	} catch (err) {
		apiError = true;
		console.error('Error fetching leaderboard:', err);
	}

	return {
		entries,
		apiError
	};
};
