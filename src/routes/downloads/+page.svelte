<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '@iconify/svelte';
	import {
		assetForOperatingSystem,
		detectOperatingSystem,
		formatBytes,
		formatCount,
		formatDate,
		getPlatform,
		latestRelease,
		platformFor,
		playableAssets,
		releaseDownloads,
		totalDownloads,
		type GithubRelease,
		type OperatingSystem
	} from '$lib/ts/github';

	interface PageData {
		releases: GithubRelease[];
		error: string | null;
	}

	let { data }: { data: PageData } = $props();

	let operatingSystem = $state<OperatingSystem>('unknown');

	const releases = $derived(data.releases);
	const latest = $derived(latestRelease(releases));
	const detectedPlatform = $derived(platformFor(operatingSystem));
	const detectedAsset = $derived(assetForOperatingSystem(latest, operatingSystem));
	const latestAssets = $derived(playableAssets(latest));
	const downloadsAllTime = $derived(totalDownloads(releases));

	onMount(() => {
		operatingSystem = detectOperatingSystem();
	});
</script>

<svelte:head>
	<title>Downloads — AstroSwarm</title>
	<meta name="description" content="Download the latest release of AstroSwarm." />
</svelte:head>

<div class="page">
	<div class="shell page-head">
		<h1 class="page-title">Downloads</h1>
		<p class="page-lede">
			Builds are published straight from the AstroSwarm repository. Grab the latest release for your
			platform, or pick an older tag from the list below.
		</p>
		{#if downloadsAllTime > 0}
			<p class="download-total">
				<Icon icon="ph:download-simple-bold" width="15" />
				{formatCount(downloadsAllTime)} downloads all time
			</p>
		{/if}
	</div>

	<div class="shell downloads">
		{#if data.error}
			<div class="notice notice-error">{data.error}</div>
		{:else if releases.length === 0}
			<div class="notice">No releases have been published yet. Check back soon.</div>
		{:else}
			{#if latest}
				<div class="card latest">
					<div class="latest-head">
						<div>
							<p class="latest-label">Latest release</p>
							<h2 class="latest-name">{latest.name || latest.tag_name}</h2>
							<p class="latest-meta">
								{latest.tag_name} · published {formatDate(latest.published_at)}
							</p>
						</div>

						{#if detectedAsset}
							<a href={detectedAsset.browser_download_url} class="btn btn-primary btn-lg" download>
								<Icon icon={detectedPlatform.icon} width="18" />
								Download for {detectedPlatform.label}
							</a>
						{/if}
					</div>

					{#if detectedAsset}
						<p class="latest-detected">
							Detected {detectedPlatform.label} · {detectedAsset.name} ({formatBytes(detectedAsset.size)})
						</p>
					{/if}

					{#if latestAssets.length > 0}
						<div class="asset-row">
							{#each latestAssets as asset (asset.id)}
								{@const platform = getPlatform(asset.name)}
								<a
									href={asset.browser_download_url}
									class="btn btn-sm"
									class:btn-ghost={asset.id !== detectedAsset?.id}
									title={`${asset.name} · ${formatCount(asset.download_count ?? 0)} downloads`}
									download
								>
									<Icon icon={platform.icon} width="15" />
									{platform.label}
									<span class="asset-size">{formatBytes(asset.size)}</span>
								</a>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<h2 class="section-title notes-heading">Before you play</h2>
			<p class="notes-lede">
				AstroSwarm builds are not code-signed, so both Windows and macOS flag them on first launch.
				The steps below let the game through.
			</p>

			<div class="note-list">
				<div class="card note" class:note-detected={operatingSystem === 'windows'}>
					<div class="note-head">
						<Icon icon="ph:windows-logo-fill" width="18" />
						<span class="note-title">Windows Defender SmartScreen</span>
						{#if operatingSystem === 'windows'}
							<span class="badge badge-brand">Your platform</span>
						{/if}
					</div>
					<p class="note-body">
						Windows shows a blue "Windows protected your PC" screen when you run the game. Click
						<strong>More info</strong>, then <strong>Run anyway</strong>. If Defender quarantines the
						download instead, open <strong>Windows Security &gt; Virus &amp; threat protection &gt;
						Protection history</strong>, find AstroSwarm, and choose <strong>Allow on device</strong>.
					</p>
				</div>

				<div class="card note" class:note-detected={operatingSystem === 'macos'}>
					<div class="note-head">
						<Icon icon="ph:apple-logo-fill" width="18" />
						<span class="note-title">macOS Privacy &amp; Security</span>
						{#if operatingSystem === 'macos'}
							<span class="badge badge-brand">Your platform</span>
						{/if}
					</div>
					<p class="note-body">
						macOS blocks the game with "AstroSwarm cannot be opened because it is from an
						unidentified developer". Open <strong>System Settings &gt; Privacy &amp; Security</strong>,
						scroll to the Security section, and click <strong>Open Anyway</strong> next to the
						AstroSwarm message, then confirm. You only need to do this once per build.
					</p>
				</div>
			</div>

			<h2 class="section-title all-heading">All releases</h2>

			<div class="release-list">
				{#each releases as release (release.id)}
					<div class="card release">
						<div class="release-head">
							<div class="release-title">
								<span class="release-name">{release.name || release.tag_name}</span>
								<span class="badge">{release.tag_name}</span>
								{#if release.prerelease}
									<span class="badge badge-warn">Pre-release</span>
								{:else if release.id === latest?.id}
									<span class="badge badge-brand">Latest</span>
								{/if}
							</div>
							<span class="release-meta">
								<span class="release-date">
									<Icon icon="ph:download-simple-bold" width="14" />
									{formatCount(releaseDownloads(release))} downloads
								</span>
								<span class="release-date">
									<Icon icon="ph:calendar-blank" width="14" />
									{formatDate(release.published_at)}
								</span>
							</span>
						</div>

						{#if release.body?.trim()}
							<p class="release-notes">{release.body.trim()}</p>
						{/if}

						<div class="asset-row">
							{#each playableAssets(release) as asset (asset.id)}
								{@const platform = getPlatform(asset.name)}
								<a
									href={asset.browser_download_url}
									class="btn btn-sm btn-ghost"
									title={`${asset.name} · ${formatCount(asset.download_count ?? 0)} downloads`}
									download
								>
									<Icon icon={platform.icon} width="15" />
									{platform.label}
									<span class="asset-size">{formatBytes(asset.size)}</span>
								</a>
							{/each}

							<a
								href={release.html_url}
								target="_blank"
								rel="noreferrer"
								class="btn btn-sm btn-ghost"
							>
								<Icon icon="ph:github-logo-fill" width="15" />
								Release notes
							</a>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.downloads {
		padding-bottom: 6rem;
	}

	.latest {
		padding: 1.75rem;
		border-color: var(--color-line-strong);
	}

	.latest-head {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1.25rem;
	}

	.latest-label {
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--color-brand);
	}

	.latest-name {
		margin-top: 0.5rem;
		font-size: 1.5rem;
		font-weight: 600;
	}

	.latest-meta {
		margin-top: 0.35rem;
		font-size: 0.85rem;
		color: var(--color-faint);
	}

	.latest-detected {
		margin-top: 1rem;
		font-size: 0.8rem;
		color: var(--color-faint);
	}

	.asset-row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-top: 1.25rem;
		padding-top: 1.25rem;
		border-top: 1px solid var(--color-line);
	}

	.asset-size {
		color: var(--color-faint);
		font-size: 0.75rem;
	}

	.all-heading {
		margin-top: 3.5rem;
		margin-bottom: 1rem;
	}

	.notes-heading {
		margin-top: 3.5rem;
	}

	.notes-lede {
		margin-top: 0.6rem;
		margin-bottom: 1rem;
		max-width: 44rem;
		font-size: 0.875rem;
		line-height: 1.65;
		color: var(--color-dim);
	}

	.note-list {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(20rem, 1fr));
		gap: 0.75rem;
	}

	.note {
		padding: 1.35rem;
	}

	.note-detected {
		border-color: var(--color-line-strong);
	}

	.note-head {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.6rem;
		color: var(--color-heading);
	}

	.note-title {
		font-size: 1rem;
		font-weight: 600;
	}

	.note-body {
		margin-top: 0.85rem;
		font-size: 0.875rem;
		line-height: 1.7;
		color: var(--color-dim);
	}

	.note-body strong {
		color: var(--color-heading);
		font-weight: 600;
	}

	.release-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.release {
		padding: 1.5rem;
	}

	.release-head {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.release-title {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.6rem;
	}

	.release-name {
		font-size: 1.05rem;
		font-weight: 600;
		color: var(--color-heading);
	}

	.release-meta {
		display: inline-flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 1rem;
	}

	.release-date {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.8rem;
		color: var(--color-faint);
	}

	.download-total {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		margin-top: 1rem;
		font-size: 0.85rem;
		color: var(--color-faint);
	}

	.release-notes {
		margin-top: 1rem;
		font-size: 0.875rem;
		line-height: 1.65;
		color: var(--color-dim);
		white-space: pre-wrap;
		display: -webkit-box;
		-webkit-line-clamp: 4;
		line-clamp: 4;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
