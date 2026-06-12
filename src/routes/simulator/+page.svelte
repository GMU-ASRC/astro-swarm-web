<script lang="ts">
	import '$lib/css/simulator.css';
	import SimGallery from '$lib/components/SimGallery.svelte';
	import type { SimRunItem } from '$lib/ts/simulator';

	interface PageData {
		runs: SimRunItem[];
		apiError: boolean;
	}

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Simulator — AstroSwarm</title>
	<meta name="description" content="Upload and share AstroSwarm simulator configs and run recordings with the community." />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-10">
		<h1 class="font-sim text-[clamp(1.8rem,4vw,3rem)] font-bold text-star-white leading-tight mb-3">
			Swarm Config Gallery
		</h1>
		<p class="font-sim text-base text-text-muted leading-relaxed max-w-150">
			Browse simulator configs and run recordings from the AstroSwarm community.
			Download setups from other players and load them directly into your game.
		</p>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	<div class="max-w-275 mx-auto px-8 max-sm:px-5 py-12 max-md:py-8">
		<SimGallery runs={data.runs} apiError={data.apiError} />
	</div>
</div>
