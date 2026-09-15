import type { PageLoad } from './$types';
import type { SimulatorEntry } from '$lib/ts/simulator';
import { apiUrl } from '$lib/ts/api';
import { error } from '@sveltejs/kit';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ params, fetch }) => {
	let response: Response;
	try {
		response = await fetch(apiUrl(`/api/simulator/entries/${params.id}`));
	} catch (err) {
		console.error('Error fetching simulator entry:', err);
		error(500, 'Server error');
	}
	if (!response.ok) {
		error(404, 'Simulator entry not found');
	}
	const entry = (await response.json()) as SimulatorEntry;
	return { entry };
};
