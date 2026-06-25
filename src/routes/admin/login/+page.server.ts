import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { apiUrl } from '$lib/ts/api';

export const prerender = false;

export const load: PageServerLoad = async ({ cookies }) => {
	if (cookies.get('astroswarm_session')) {
		throw redirect(303, '/admin');
	}
};

export const actions: Actions = {
	default: async ({ request, cookies, fetch }) => {
		const data = await request.formData();
		const apiKey = data.get('apiKey');

		if (!apiKey || typeof apiKey !== 'string') {
			return fail(400, { error: 'API key is required' });
		}

		try {
			// Verify API key by making a dummy GET request.
			// Actually the baseline endpoint is public, let's just make a DELETE to a nonexistent id to verify Auth.
			const res = await fetch(apiUrl('/api/evaluations/test-auth'), {
				method: 'DELETE',
				headers: { 'X-API-Key': apiKey }
			});

			if (res.status === 401 || res.status === 403) {
				return fail(401, { error: 'Invalid API Key' });
			}
			
			// Valid!
			cookies.set('astroswarm_session', apiKey, {
				path: '/',
				httpOnly: true,
				sameSite: 'lax',
				secure: process.env.NODE_ENV === 'production',
				maxAge: 60 * 60 * 24 * 7 // 1 week
			});

		} catch (err) {
			return fail(500, { error: 'Failed to connect to backend API' });
		}

		throw redirect(303, '/admin');
	}
};
