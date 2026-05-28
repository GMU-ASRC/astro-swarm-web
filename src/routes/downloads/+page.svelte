<script lang="ts">
	import Icon from '@iconify/svelte';
	import { getPlatform, formatBytes, formatDate, type GithubRelease } from '$lib/ts/github';

	let { data } = $props();

	const releases = $derived(data.releases as GithubRelease[]);
	const error = $derived(data.error as string | null);
	const latest = $derived(releases.find((r) => !r.prerelease) ?? releases[0] ?? null);
</script>

<svelte:head>
	<title>Downloads — AstroSwarm</title>
	<meta name="description" content="Download the latest release of AstroSwarm." />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20">
	<div class="max-w-275 mx-auto px-8 max-sm:px-5 pt-12 pb-10">
		<h1 class="font-game text-[clamp(2rem,4vw,3rem)] text-star-white tracking-[0.06em] m-0">
			DOWNLOADS
		</h1>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-275 mx-auto px-8 max-sm:px-5 py-10 pb-24 flex flex-col gap-3">

		{#if error}
			<div class="border-2 border-btn-border bg-btn-bg px-6 py-10 text-center">
				<div class="flex justify-center mb-4 opacity-50">
					<Icon icon="ph:warning" width="40" color="var(--color-text-muted)" />
				</div>
				<p class="font-game text-sm tracking-widest text-text-muted m-0">{error}</p>
			</div>

		{:else if releases.length === 0}
			<div class="border-2 border-dashed border-btn-border px-6 py-16 text-center">
				<div class="flex justify-center mb-4 opacity-40">
					<Icon icon="ph:package" width="48" color="var(--color-text-muted)" />
				</div>
				<p class="font-game text-sm tracking-widest text-text-muted m-0">NO RELEASES YET</p>
				<p class="font-game text-[0.6rem] tracking-widest text-text-muted/40 mt-2 mb-0">CHECK BACK SOON</p>
			</div>

		{:else}
			{#each releases as release (release.id)}
				<div class="border-2 border-btn-border bg-btn-bg/60 transition-[border-color] duration-150 hover:border-btn-hover-border"
					class:border-btn-hover-border={release === latest}
					class:bg-btn-bg={release !== latest}
				>
					<div class="px-6 pt-5 pb-4 flex items-start justify-between gap-4 flex-wrap border-b border-btn-border/40">
						<div class="flex items-center gap-3 flex-wrap">
							<span class="font-game text-[1rem] text-star-white tracking-[0.04em]">
								{release.name || release.tag_name}
							</span>
							<span class="font-game text-[0.6rem] tracking-[0.08em] px-2 py-0.5 border border-btn-border text-text-muted">
								{release.tag_name}
							</span>
							{#if release.prerelease}
								<span class="font-game text-[0.55rem] tracking-[0.08em] px-2 py-0.5 border border-accent-cyan/40 text-accent-cyan">
									PRE-RELEASE
								</span>
							{/if}
							{#if release === latest && !release.prerelease}
								<span class="font-game text-[0.55rem] tracking-[0.08em] px-2 py-0.5 bg-btn-press-bg border border-btn-hover-border text-star-white">
									LATEST
								</span>
							{/if}
						</div>
						<div class="flex items-center gap-2 text-text-muted/60">
							<Icon icon="ph:calendar-blank" width="14" color="var(--color-text-muted)" />
							<span class="font-game text-[0.6rem] tracking-[0.06em]">
								{formatDate(release.published_at)}
							</span>
						</div>
					</div>

					{#if release.body?.trim()}
						<div class="px-6 py-4 border-b border-btn-border/40">
							<p class="font-game text-[0.6rem] leading-loose tracking-wider text-text-muted m-0 whitespace-pre-wrap line-clamp-4">
								{release.body.trim()}
							</p>
							<a
								href={release.html_url}
								target="_blank"
								rel="noreferrer"
								class="font-game text-[0.55rem] tracking-[0.08em] text-accent-cyan no-underline hover:text-star-white transition-colors mt-2 inline-block"
							>
								VIEW FULL NOTES →
							</a>
						</div>
					{/if}

					{#if release.assets.length > 0}
						<div class="px-6 py-4 flex flex-wrap gap-2">
							{#each release.assets as asset (asset.id)}
								{@const platform = getPlatform(asset.name)}
								<a
									href={asset.browser_download_url}
									class="flex items-center gap-2 font-game text-[0.65rem] tracking-[0.06em] py-2.75 px-4.5 bg-btn-bg text-btn-text border-2 border-btn-border no-underline transition-[background,border-color,color] duration-150 hover:bg-btn-hover-bg hover:border-btn-hover-border hover:text-star-white"
									download
								>
									<Icon icon={platform.icon} width="16" />
									{platform.label}
									<span class="text-text-muted/50 font-game text-[0.5rem]">
										{formatBytes(asset.size)}
									</span>
								</a>
							{/each}

							<a
								href={release.html_url}
								target="_blank"
								rel="noreferrer"
								class="flex items-center gap-2 font-game text-[0.65rem] tracking-[0.06em] py-2.75 px-4.5 text-text-muted border-2 border-btn-border/40 no-underline transition-[color,border-color] duration-150 hover:text-accent-cyan hover:border-accent-cyan/40"
							>
								<Icon icon="ph:github-logo" width="16" />
								VIEW ON GITHUB
							</a>
						</div>
					{:else}
						<div class="px-6 py-4">
							<a
								href={release.html_url}
								target="_blank"
								rel="noreferrer"
								class="flex items-center gap-2 font-game text-[0.65rem] tracking-[0.06em] py-2.75 px-4.5 text-text-muted border-2 border-btn-border/40 no-underline transition-[color,border-color] duration-150 hover:text-accent-cyan hover:border-accent-cyan/40 w-fit"
							>
								<Icon icon="ph:github-logo" width="16" />
								VIEW ON GITHUB
							</a>
						</div>
					{/if}
				</div>
			{/each}
		{/if}
	</div>
</div>
