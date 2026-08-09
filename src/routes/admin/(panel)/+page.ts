import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	const { adminKey } = await parent();

	const storagePromise = (async () => {
		const res = await fetch(apiUrl('/api/admin/storage'), {
			headers: { 'X-API-Key': adminKey }
		});
		if (!res.ok) return null;
		return await res.json();
	})();

	return { storagePromise };
};
