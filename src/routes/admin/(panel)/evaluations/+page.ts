import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) => {
	const evaluationsPromise = (async () => {
		const res = await fetch(apiUrl('/api/evaluations'));
		return res.ok ? await res.json() : [];
	})();

	return { evaluationsPromise };
};
