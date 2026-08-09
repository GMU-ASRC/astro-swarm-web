import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch, params }) => {
	const profilePromise = (async () => {
		const res = await fetch(apiUrl(`/api/evaluations/players/${params.id}`));
		if (!res.ok) return null;
		return await res.json();
	})();

	return { profilePromise, id: params.id };
};
