import { browser } from '$app/environment';
import { redirect } from '@sveltejs/kit';
import { apiUrl } from '$lib/ts/api';
import type { LayoutLoad } from './$types';

export const ssr = false;
export const prerender = false;

export const sessionKey = 'astroswarm_session';

export const load: LayoutLoad = async ({ url, fetch }) => {
	if (!browser) {
		return { adminKey: '' };
	}

	const adminKey = localStorage.getItem(sessionKey) ?? '';
	const onLogin = url.pathname.startsWith('/admin/login');

	if (!adminKey) {
		if (!onLogin) {
			throw redirect(303, '/admin/login');
		}
		return { adminKey: '' };
	}

	if (!onLogin) {
		let unauthorized = false;
		try {
			const res = await fetch(apiUrl('/api/admin/me'), { headers: { 'X-API-Key': adminKey } });
			unauthorized = res.status === 401 || res.status === 403;
		} catch {
			// Network/transient errors: let the page load and surface errors itself.
			unauthorized = false;
		}
		if (unauthorized) {
			localStorage.removeItem(sessionKey);
			throw redirect(303, '/admin/login');
		}
	}

	return { adminKey };
};
