import type { PageLoad } from './$types';
import { fetchReleases } from '$lib/ts/github';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
	return fetchReleases(fetch);
};
