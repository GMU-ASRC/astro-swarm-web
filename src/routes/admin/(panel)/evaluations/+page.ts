import { apiUrl } from '$lib/ts/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const res = await fetch(apiUrl('/api/evaluations'));
	const evaluations = res.ok ? await res.json() : [];
	return { evaluations };
};
