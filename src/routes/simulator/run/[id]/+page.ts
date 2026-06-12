import type { PageLoad } from './$types';
import { apiUrl } from '$lib/ts/api';
import type { SimRunItem } from '$lib/ts/simulator';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch, params }) => {
	try {
		const [runRes, configRes] = await Promise.all([
			fetch(apiUrl(`/api/runs/${params.id}`)),
			fetch(apiUrl(`/api/runs/${params.id}/config`))
		]);

		if (!runRes.ok) {
			if (runRes.status === 404) {
				return { run: null, config: null, error: 'Run not found.' };
			}
			return { run: null, config: null, error: 'Failed to load run details.' };
		}

		const run: SimRunItem = await runRes.json();
		const config = configRes.ok ? await configRes.json() : null;

		return { run, config, error: null };
	} catch (e) {
		return { run: null, config: null, error: 'Network error.' };
	}
};
