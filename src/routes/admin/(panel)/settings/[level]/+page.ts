import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch, params }) => {
	const settingsPromise = (async () => {
		const res = await fetch(apiUrl('/api/evaluations/settings'));
		return res.ok ? await res.json() : null;
	})();

	return { settingsPromise, levelSlug: params.level };
};
