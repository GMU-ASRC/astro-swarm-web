import type { PageLoad } from './$types';
import type { PlayerListItem } from '$lib/ts/evaluation';
import { apiUrl } from '$lib/ts/api';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
	let players: PlayerListItem[] = [];
	let apiError = false;

	try {
		const res = await fetch(apiUrl('/api/evaluations'));
		if (res.ok) {
			players = await res.json();
		} else {
			apiError = true;
			console.error('Failed to fetch players:', await res.text());
		}
	} catch (err) {
		apiError = true;
		console.error('Error fetching players:', err);
	}

	return { players, apiError };
};
