import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) => {
	const playersPromise = (async () => {
		const res = await fetch(apiUrl('/api/evaluations/players'));
		return res.ok ? await res.json() : [];
	})();

	return { playersPromise };
};
