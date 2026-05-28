<script lang="ts">
	import '$lib/css/previews.css';
	import Icon from '@iconify/svelte';

	let { data } = $props();

	const images = $derived(data.images);

	let activeIndex = $state<number | null>(null);

	function openLightbox(index: number) {
		activeIndex = index;
	}

	function closeLightbox() {
		activeIndex = null;
	}

	function goNext() {
		if (activeIndex === null) return;
		activeIndex = (activeIndex + 1) % images.length;
	}

	function goPrev() {
		if (activeIndex === null) return;
		activeIndex = (activeIndex - 1 + images.length) % images.length;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (activeIndex === null) return;
		if (event.key === 'ArrowRight') goNext();
		if (event.key === 'ArrowLeft') goPrev();
		if (event.key === 'Escape') closeLightbox();
	}

	function labelFromFilename(filename: string) {
		return filename.replace(/\.[^.]+$/, '').replace(/[-_]/g, ' ').toUpperCase();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<svelte:head>
	<title>Previews — AstroSwarm</title>
	<meta name="description" content="Screenshot previews of AstroSwarm gameplay and simulator." />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20">
	<div class="max-w-275 mx-auto px-8 max-sm:px-5 pt-12 pb-10">
		<h1 class="font-game text-[clamp(2rem,4vw,3rem)] text-star-white tracking-[0.06em] m-0">
			SCREENSHOTS
		</h1>
	</div>

	<div class="h-px mx-8 max-sm:mx-5" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

	{#if images.length === 0}
		<div class="max-w-275 mx-8 max-sm:mx-5 mt-10 py-16 text-center border-2 border-dashed border-btn-border">
			<p class="font-game text-base tracking-widest text-text-muted m-0">NO PREVIEWS YET</p>
		</div>
	{:else}
		<div class="max-w-275 mx-auto px-8 max-sm:px-5 pt-10 pb-24 grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] max-sm:grid-cols-2 gap-0.75 max-sm:gap-0.5">
			{#each images as image, i}
				<button
					class="group relative overflow-hidden bg-page-bg border-2 border-btn-border cursor-pointer aspect-video transition-[border-color] duration-150 hover:border-btn-hover-border"
					onclick={() => openLightbox(i)}
				>
					<img
						src="/previews/{image}"
						alt={labelFromFilename(image)}
						loading="lazy"
						class="w-full h-full object-cover block [image-rendering:pixelated] transition-transform duration-200 group-hover:scale-[1.04]"
					/>
					<div class="absolute bottom-0 left-0 right-0 px-3 py-2 opacity-0 group-hover:opacity-100 transition-opacity duration-200"
						style="background: linear-gradient(to top, rgba(2,4,9,0.9), transparent)">
						<span class="font-game text-xs tracking-[0.08em] text-text-muted">
							{labelFromFilename(image)}
						</span>
					</div>
				</button>
			{/each}
		</div>
	{/if}
</div>

{#if activeIndex !== null}
	<div
		class="fixed inset-0 z-200 flex items-center justify-center p-8"
		style="background: rgba(2,4,9,0.94)"
		onclick={(e) => e.target === e.currentTarget && closeLightbox()}
		onkeydown={(e) => { if (e.key === 'Escape') closeLightbox(); }}
		role="dialog"
		aria-modal="true"
		aria-label="Image preview"
		tabindex="-1"
	>
		<div class="relative w-full max-w-[min(1100px,100%)]">
			<button
				class="absolute right-0 -top-9 font-game text-[0.85rem] tracking-[0.12em] text-text-muted bg-none border-none cursor-pointer p-0 transition-colors hover:text-accent-cyan"
				onclick={closeLightbox}
			>
				[ CLOSE ]
			</button>

			<img
				class="w-full h-auto block border-2 border-btn-hover-border [image-rendering:pixelated]"
				src="/previews/{images[activeIndex]}"
				alt={labelFromFilename(images[activeIndex])}
			/>

			<p class="font-game text-xs tracking-widest text-text-muted mt-3 text-center">
				{labelFromFilename(images[activeIndex])} — {activeIndex + 1} / {images.length}
			</p>

			{#if images.length > 1}
				<button
					class="lightbox-nav absolute top-1/2 -translate-y-1/2 bg-btn-bg border-2 border-btn-border text-btn-text font-game text-base px-3 py-2 cursor-pointer transition-[background,border-color] duration-150 hover:bg-btn-hover-bg hover:border-btn-hover-border"
					onclick={goPrev}
				>
					<Icon icon="ph:caret-left" width="16" />
				</button>
				<button
					class="lightbox-nav absolute top-1/2 -translate-y-1/2 bg-btn-bg border-2 border-btn-border text-btn-text font-game text-base px-3 py-2 cursor-pointer transition-[background,border-color] duration-150 hover:bg-btn-hover-bg hover:border-btn-hover-border"
					onclick={goNext}
				>
					<Icon icon="ph:caret-right" width="16" />
				</button>
			{/if}
		</div>
	</div>
{/if}
