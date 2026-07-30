import type { PageLoad } from './$types';
import type { SurviveMatch } from '$lib/ts/survive';
import { apiUrl } from '$lib/ts/api';
import { error } from '@sveltejs/kit';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ params, fetch }) => {
	let match: SurviveMatch;
	try {
		const res = await fetch(apiUrl(`/api/survive/matches/${params.id}`));
		if (!res.ok) {
			error(404, 'Match not found');
		}
		match = await res.json();
	} catch (err) {
		console.error('Error fetching survive match:', err);
		error(500, 'Server error');
	}

	return { match };
};
