import { apiUrl } from '$lib/ts/api';

export type PageLoadKind = 'first' | 'navigation';

const MAX_REPORTED_MS = 120000;

export function reportPageLoad(route: string | null, kind: PageLoadKind, totalMs: number) {
	if (!route || !Number.isFinite(totalMs) || totalMs < 0 || totalMs > MAX_REPORTED_MS) return;
	const body = JSON.stringify({ route, kind, total_ms: Math.round(totalMs * 10) / 10 });
	const url = apiUrl('/api/metrics/page-load');
	try {
		if (navigator.sendBeacon?.(url, body)) return;
		fetch(url, { method: 'POST', body, keepalive: true }).catch(() => {});
	} catch {
		return;
	}
}
