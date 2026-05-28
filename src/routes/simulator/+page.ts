import type { PageLoad } from './$types';
import { apiUrl } from '$lib/ts/api';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
	try {
		const [configsRes, runsRes] = await Promise.all([
			fetch(apiUrl('/api/configs?page_size=50')),
			fetch(apiUrl('/api/runs?page_size=50'))
		]);

		const configs = configsRes.ok ? (await configsRes.json()).items : [];
		const runs = runsRes.ok ? (await runsRes.json()).items : [];

		return { configs, runs, apiError: false };
	} catch {
		return { configs: [], runs: [], apiError: true };
	}
};
