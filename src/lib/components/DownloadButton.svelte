<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '@iconify/svelte';
	import {
		assetForOperatingSystem,
		detectOperatingSystem,
		fetchReleases,
		latestRelease,
		platformFor,
		type GithubAsset,
		type OperatingSystem
	} from '$lib/ts/github';

	interface Props {
		size?: 'default' | 'large';
	}

	let { size = 'default' }: Props = $props();

	let operatingSystem = $state<OperatingSystem>('unknown');
	let asset = $state<GithubAsset | null>(null);

	let platform = $derived(platformFor(operatingSystem));
	let href = $derived(asset?.browser_download_url ?? '/downloads');
	let label = $derived(asset ? `Download for ${platform.label}` : 'Downloads');

	onMount(async () => {
		operatingSystem = detectOperatingSystem();
		if (operatingSystem === 'unknown') return;

		const { releases } = await fetchReleases();
		asset = assetForOperatingSystem(latestRelease(releases), operatingSystem);
	});
</script>

<a {href} class="btn btn-primary" class:btn-lg={size === 'large'} download={asset ? '' : undefined}>
	<Icon icon={asset ? platform.icon : 'ph:download-simple-bold'} width="18" />
	{label}
</a>
