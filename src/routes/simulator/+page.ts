import type { PageLoad } from './$types';
import { apiUrl } from '$lib/ts/api';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
	try {
		const runsRes = await fetch(apiUrl('/api/runs?page_size=50'));
		const runs = runsRes.ok ? (await runsRes.json()).items : [];

		return { runs, apiError: false };
	} catch {
		return { runs: [], apiError: true };
	}
};
