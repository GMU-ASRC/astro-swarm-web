import type { PageLoad } from './$types';
import type { GithubRelease } from '$lib/ts/github';

const REPO = 'GMU-ASRC/astro-swarm';

export const ssr = false;
export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`https://api.github.com/repos/${REPO}/releases`, {
			headers: { Accept: 'application/vnd.github+json' }
		});

		if (!response.ok) {
			return { releases: [] as GithubRelease[], error: `GitHub API returned ${response.status}` };
		}

		const all: GithubRelease[] = await response.json();
		const releases = all.filter((r) => !r.draft);

		return { releases, error: null };
	} catch {
		return { releases: [] as GithubRelease[], error: 'Failed to fetch releases.' };
	}
};
