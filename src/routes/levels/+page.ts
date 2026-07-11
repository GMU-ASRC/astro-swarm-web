import type { PageLoad } from './$types';
import type { PlayerListItem } from '$lib/ts/evaluation';
import { apiUrl } from '$lib/ts/api';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = ({ fetch }) => {
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

	return { playersPromise };
};
