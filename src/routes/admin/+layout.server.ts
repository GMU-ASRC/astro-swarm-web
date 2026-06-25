import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

export const prerender = false;

export const load: LayoutServerLoad = async ({ cookies, url }) => {
	const adminKey = cookies.get('astroswarm_session');

	if (!adminKey && !url.pathname.startsWith('/admin/login')) {
		throw redirect(303, '/admin/login');
	}

	return {
		adminKey
	};
};
