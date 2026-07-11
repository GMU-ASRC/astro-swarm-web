import type { PageLoad } from './$types';
import { apiUrl } from '$lib/ts/api';
import { error } from '@sveltejs/kit';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ params, fetch }) => {
	const res = await fetch(apiUrl(`/api/evaluations/players/${params.id}`)).catch(() => null);
	if (!res || !res.ok) {
		error(404, 'Player not found');
	}
	return { profile: await res.json() };
};
