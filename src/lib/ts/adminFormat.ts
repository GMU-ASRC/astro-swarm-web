export function formatBytes(bytes: number | null | undefined): string {
	if (bytes == null) return '—';
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function formatCount(value: number | null | undefined): string {
	return value == null ? '—' : value.toLocaleString();
}

export function formatPercent(value: number | null | undefined): string {
	return value == null ? '—' : `${value.toFixed(1)}%`;
}

export function formatSeconds(seconds: number | null | undefined): string {
	if (seconds == null) return '—';
	if (seconds < 60) return `${Math.round(seconds)}s`;
	if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${Math.round(seconds % 60)}s`;
	return `${Math.floor(seconds / 3600)}h ${Math.round((seconds % 3600) / 60)}m`;
}

export function formatMs(ms: number | null | undefined): string {
	if (ms == null) return '—';
	if (ms < 1000) return `${ms.toFixed(ms < 10 ? 1 : 0)} ms`;
	return `${(ms / 1000).toFixed(2)} s`;
}

export function formatWhen(iso: string | null | undefined): string {
	return iso ? new Date(iso).toLocaleString() : 'never';
}

export function formatAgo(iso: string | null | undefined): string {
	if (!iso) return 'never';
	const seconds = (Date.now() - new Date(iso).getTime()) / 1000;
	if (seconds < 60) return 'just now';
	if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
	if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
	return `${Math.floor(seconds / 86400)}d ago`;
}
