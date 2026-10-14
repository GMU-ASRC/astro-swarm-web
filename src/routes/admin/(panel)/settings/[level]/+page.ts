import { redirect } from '@sveltejs/kit';
import { levelNumberOf } from '$lib/ts/adminSettings';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ params }) => {
	redirect(308, `/admin/settings#level-${levelNumberOf(params.level)}`);
};
