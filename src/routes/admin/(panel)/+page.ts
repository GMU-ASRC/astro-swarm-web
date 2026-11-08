import { apiUrl } from '$lib/ts/api';
import type { AdminStats } from '$lib/ts/adminStats';
import type { PageLoad } from './$types';

async function adminJson<T>(fetcher: typeof fetch, path: string, adminKey: string): Promise<T | null> {
	try {
		const res = await fetcher(apiUrl(path), { headers: { 'X-API-Key': adminKey } });
		return res.ok ? ((await res.json()) as T) : null;
	} catch {
		return null;
	}
}

export const load: PageLoad = async ({ fetch, parent }) => {
	const { adminKey } = await parent();
	return {
		storagePromise: adminJson<Record<string, any>>(fetch, '/api/admin/storage', adminKey),
		statsPromise: adminJson<AdminStats>(fetch, '/api/admin/stats', adminKey)
	};
};
