export interface GithubAsset {
	id: number;
	name: string;
	size: number;
	download_count: number;
	browser_download_url: string;
	content_type: string;
}

export interface GithubRelease {
	id: number;
	tag_name: string;
	name: string;
	body: string;
	prerelease: boolean;
	published_at: string;
	assets: GithubAsset[];
	html_url: string;
}

export interface PlatformInfo {
	label: string;
	icon: string;
}

export function getPlatform(filename: string): PlatformInfo {
	const lower = filename.toLowerCase();
	if (lower.includes('windows') || lower.endsWith('.exe')) return { label: 'Windows', icon: 'ph:windows-logo' };
	if (lower.includes('macos') || lower.includes('mac') || lower.includes('osx') || lower.endsWith('.dmg')) return { label: 'macOS', icon: 'ph:apple-logo' };
	if (lower.includes('linux') || lower.endsWith('.appimage')) return { label: 'Linux', icon: 'ph:linux-logo' };
	if (lower.endsWith('.zip')) return { label: 'ZIP', icon: 'ph:file-zip' };
	if (lower.endsWith('.tar.gz') || lower.endsWith('.tar.xz')) return { label: 'TAR', icon: 'ph:file-archive' };
	return { label: 'Download', icon: 'ph:download-simple' };
}

export function formatBytes(bytes: number): string {
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function formatDate(dateStr: string): string {
	return new Date(dateStr).toLocaleDateString('en-US', {
		year: 'numeric',
		month: 'short',
		day: 'numeric',
	});
}
