import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch, params }) => {
	const entryPromise = (async () => {
		const res = await fetch(apiUrl(`/api/runs/${params.id}`));
		if (!res.ok) return null;
		return await res.json();
	})();

	return { entryPromise, id: params.id };
};
