import type { PageLoad } from './$types';
import type { SimulatorEntryListItem } from '$lib/ts/simulator';
import { apiUrl } from '$lib/ts/api';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = ({ fetch }) => {
	const entriesPromise = (async () => {
		try {
			const res = await fetch(apiUrl('/api/simulator/entries?limit=200'));
			if (res.ok) {
				return { entries: (await res.json()) as SimulatorEntryListItem[], apiError: false };
			}
			console.error('Failed to fetch simulator entries:', await res.text());
			return { entries: [] as SimulatorEntryListItem[], apiError: true };
		} catch (err) {
			console.error('Error fetching simulator entries:', err);
			return { entries: [] as SimulatorEntryListItem[], apiError: true };
		}
	})();

	return { entriesPromise };
};
