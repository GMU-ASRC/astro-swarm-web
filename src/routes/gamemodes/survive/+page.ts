import type { PageLoad } from './$types';
import type { SurviveMatchListItem } from '$lib/ts/survive';
import { apiUrl } from '$lib/ts/api';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = ({ fetch }) => {
	const matchesPromise = (async () => {
		try {
			const res = await fetch(apiUrl('/api/survive/matches'));
			if (res.ok) {
				return { matches: (await res.json()) as SurviveMatchListItem[], apiError: false };
			}
			console.error('Failed to fetch survive matches:', await res.text());
			return { matches: [] as SurviveMatchListItem[], apiError: true };
		} catch (err) {
			console.error('Error fetching survive matches:', err);
			return { matches: [] as SurviveMatchListItem[], apiError: true };
		}
	})();

	return { matchesPromise };
};
