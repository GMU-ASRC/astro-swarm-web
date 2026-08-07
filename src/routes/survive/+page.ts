import type { PageLoad } from './$types';
import { redirect } from '@sveltejs/kit';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = () => {
	redirect(308, '/gamemodes/survive');
};
