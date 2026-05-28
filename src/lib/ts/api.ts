import { PUBLIC_API_URL } from '$env/static/public';

export const apiBase = PUBLIC_API_URL ?? '';

export function apiUrl(path: string): string {
	return `${apiBase}${path}`;
}
