import { apiUrl } from '$lib/ts/api';

export type BulkResult = {
	succeeded: string[];
	failed: string[];
};

export async function runBulk(
	ids: string[],
	adminKey: string,
	method: 'POST' | 'DELETE',
	path: (id: string) => string
): Promise<BulkResult> {
	const succeeded: string[] = [];
	const failed: string[] = [];
	for (const id of ids) {
		try {
			const res = await fetch(apiUrl(path(id)), {
				method,
				headers: { 'X-API-Key': adminKey }
			});
			if (res.ok || res.status === 202 || res.status === 204) succeeded.push(id);
			else failed.push(id);
		} catch {
			failed.push(id);
		}
	}
	return { succeeded, failed };
}

export function bulkMessage(pastTense: string, noun: string, result: BulkResult): string {
	const summary = `${pastTense} ${result.succeeded.length} ${noun}`;
	if (result.failed.length === 0) return `${summary}.`;
	return `${summary}, ${result.failed.length} failed.`;
}
