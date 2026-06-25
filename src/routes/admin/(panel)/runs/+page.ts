import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) => {
	const runsPromise = (async () => {
		const res = await fetch(apiUrl('/api/runs?page=1&page_size=100'));
		const data = res.ok ? await res.json() : { items: [] };
		return data.items || [];
	})();

	return { runsPromise };
};
