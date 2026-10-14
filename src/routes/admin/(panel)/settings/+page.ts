import { apiUrl } from '$lib/ts/api';
import type { AdminSettings } from '$lib/ts/adminSettings';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) => {
	const settingsPromise: Promise<AdminSettings | null> = (async () => {
		const res = await fetch(apiUrl('/api/evaluations/settings'));
		return res.ok ? await res.json() : null;
	})();

	return { settingsPromise };
};
