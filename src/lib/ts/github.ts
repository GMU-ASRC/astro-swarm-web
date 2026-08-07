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
	draft: boolean;
	published_at: string;
	assets: GithubAsset[];
	html_url: string;
}

export const REPO = 'GMU-ASRC/astro-swarm';

export interface ReleaseFetchResult {
	releases: GithubRelease[];
	error: string | null;
}

export async function fetchReleases(fetchFn: typeof fetch = fetch): Promise<ReleaseFetchResult> {
	try {
		const response = await fetchFn(`https://api.github.com/repos/${REPO}/releases`, {
			headers: { Accept: 'application/vnd.github+json' }
		});

		if (!response.ok) {
			return { releases: [], error: `GitHub API returned ${response.status}` };
		}

		const all: GithubRelease[] = await response.json();
		return { releases: all.filter((release) => !release.draft), error: null };
	} catch {
		return { releases: [], error: 'Failed to fetch releases.' };
	}
}

export type OperatingSystem = 'windows' | 'macos' | 'linux' | 'unknown';

export interface PlatformInfo {
	os: OperatingSystem;
	label: string;
	icon: string;
}

const PLATFORMS: Record<OperatingSystem, PlatformInfo> = {
	windows: { os: 'windows', label: 'Windows', icon: 'ph:windows-logo-fill' },
	macos: { os: 'macos', label: 'macOS', icon: 'ph:apple-logo-fill' },
	linux: { os: 'linux', label: 'Linux', icon: 'ph:linux-logo-fill' },
	unknown: { os: 'unknown', label: 'Download', icon: 'ph:download-simple-bold' }
};

export function platformFor(os: OperatingSystem): PlatformInfo {
	return PLATFORMS[os];
}

export function getPlatform(filename: string): PlatformInfo {
	const lower = filename.toLowerCase();
	if (lower.includes('windows') || lower.includes('win64') || lower.endsWith('.exe')) return PLATFORMS.windows;
	if (lower.includes('macos') || lower.includes('mac') || lower.includes('osx') || lower.endsWith('.dmg')) return PLATFORMS.macos;
	if (lower.includes('linux') || lower.endsWith('.appimage') || lower.endsWith('.x86_64')) return PLATFORMS.linux;
	if (lower.endsWith('.zip')) return { os: 'unknown', label: 'ZIP', icon: 'ph:file-zip' };
	if (lower.endsWith('.tar.gz') || lower.endsWith('.tar.xz')) return { os: 'unknown', label: 'TAR', icon: 'ph:file-archive' };
	return PLATFORMS.unknown;
}

export function detectOperatingSystem(): OperatingSystem {
	if (typeof navigator === 'undefined') return 'unknown';

	const hinted = (navigator as { userAgentData?: { platform?: string } }).userAgentData?.platform ?? '';
	const source = `${hinted} ${navigator.platform ?? ''} ${navigator.userAgent ?? ''}`.toLowerCase();

	if (source.includes('android')) return 'unknown';
	if (source.includes('iphone') || source.includes('ipad')) return 'macos';
	if (source.includes('win')) return 'windows';
	if (source.includes('mac') || source.includes('darwin')) return 'macos';
	if (source.includes('linux') || source.includes('x11') || source.includes('cros')) return 'linux';
	return 'unknown';
}

export function assetForOperatingSystem(release: GithubRelease | null, os: OperatingSystem): GithubAsset | null {
	if (!release || os === 'unknown') return null;
	return release.assets.find((asset) => getPlatform(asset.name).os === os) ?? null;
}

export function latestRelease(releases: GithubRelease[]): GithubRelease | null {
	return releases.find((release) => !release.prerelease) ?? releases[0] ?? null;
}

export function formatBytes(bytes: number): string {
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function formatDate(dateStr: string): string {
	return new Date(dateStr).toLocaleDateString('en-US', {
		year: 'numeric',
		month: 'short',
		day: 'numeric'
	});
}
