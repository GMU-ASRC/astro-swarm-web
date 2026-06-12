export interface SimSpecies {
	id: string;
	name: string;
	color: string;
}


export interface SimRunItem {
	id: string;
	title: string;
	description: string;
	author: string;
	thumbnail_filename: string | null;
	file_size: number;
	species: SimSpecies[];
	robot_count: number;
	frame_count: number;
	duration_seconds: number;
	download_count: number;
	created_at: string;
}

export function formatRelativeTime(dateStr: string): string {
	const diff = Date.now() - new Date(dateStr).getTime();
	const minutes = Math.floor(diff / 60000);
	const hours = Math.floor(minutes / 60);
	const days = Math.floor(hours / 24);

	if (days > 0) return `${days}d ago`;
	if (hours > 0) return `${hours}h ago`;
	if (minutes > 0) return `${minutes}m ago`;
	return 'just now';
}

export function formatDuration(seconds: number): string {
	if (seconds < 60) return `${seconds}s`;
	const m = Math.floor(seconds / 60);
	const s = Math.round(seconds % 60);
	return `${m}m ${s}s`;
}
