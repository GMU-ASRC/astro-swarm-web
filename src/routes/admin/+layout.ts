import { browser } from '$app/environment';
import { redirect } from '@sveltejs/kit';
import type { LayoutLoad } from './$types';

export const ssr = false;
export const prerender = false;

export const sessionKey = 'astroswarm_session';

export const load: LayoutLoad = ({ url }) => {
	if (!browser) {
		return { adminKey: '' };
	}

	const adminKey = localStorage.getItem(sessionKey) ?? '';

	if (!adminKey && !url.pathname.startsWith('/admin/login')) {
		throw redirect(303, '/admin/login');
	}

	return { adminKey };
};
