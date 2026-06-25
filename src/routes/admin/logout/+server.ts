import { redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ cookies }) => {
	cookies.delete('astroswarm_session', { path: '/' });
	throw redirect(303, '/admin/login');
};
