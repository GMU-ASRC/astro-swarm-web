import type { PageLoad } from './$types';
import type { PlayerEvaluation, BaselineResult } from '$lib/ts/evaluation';
import { apiUrl } from '$lib/ts/api';
import { error } from '@sveltejs/kit';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ params, fetch }) => {
	let evaluation: PlayerEvaluation;
	try {
		const res = await fetch(apiUrl(`/api/evaluations/${params.id}`));
		if (!res.ok) {
			error(404, 'Player not found');
		}
		evaluation = await res.json();
	} catch (err) {
		console.error('Error fetching evaluation:', err);
		error(500, 'Server error');
	}

	let baseline: BaselineResult = { results: [], samples: 0 };
	try {
		const res = await fetch(apiUrl('/api/evaluations/baseline'));
		if (res.ok) {
			baseline = await res.json();
		}
	} catch (err) {
		console.error('Error fetching baseline:', err);
	}

	return { evaluation, baseline };
};
