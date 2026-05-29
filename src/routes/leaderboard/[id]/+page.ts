import type { PageLoad } from './$types';
import type { LeaderboardEntry } from '$lib/ts/leaderboard';
import { apiUrl } from '$lib/ts/api';
import { error } from '@sveltejs/kit';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ params, fetch }) => {
	try {
		const res = await fetch(apiUrl(`/api/leaderboard/${params.id}`));
		if (!res.ok) {
			error(404, 'Entry not found');
		}
		const entry: LeaderboardEntry = await res.json();
		return { entry };
	} catch (err) {
		console.error('Error fetching leaderboard entry:', err);
		error(500, 'Server error');
	}
};
