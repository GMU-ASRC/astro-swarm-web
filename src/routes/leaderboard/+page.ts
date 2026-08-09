import type { PageLoad } from './$types';
import { apiUrl } from '$lib/ts/api';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
	try {
		const res = await fetch(apiUrl('/api/evaluations/players'));
		if (res.ok) {
			return { players: await res.json(), apiError: false };
		}
		console.error('Failed to fetch success rate leaderboard:', await res.text());
		return { players: [], apiError: true };
	} catch (err) {
		console.error('Error fetching success rate leaderboard:', err);
		return { players: [], apiError: true };
	}
};
