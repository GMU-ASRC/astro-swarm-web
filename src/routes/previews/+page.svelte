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
		return filename.replace(/\.[^.]+$/, '').replace(/[-_]/g, ' ');
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<svelte:head>
	<title>Screenshots — AstroSwarm</title>
	<meta name="description" content="Screenshot previews of AstroSwarm gameplay." />
</svelte:head>

<div class="page">
	<div class="shell shell-wide page-head">
		<h1 class="page-title">Screenshots</h1>
		<p class="page-lede">Captures from the game as it takes shape. Click any shot to view it full size.</p>
	</div>

	<div class="shell shell-wide gallery-wrap">
		{#if images.length === 0}
			<div class="notice">No screenshots have been published yet.</div>
		{:else}
			<div class="gallery">
				{#each images as image, index}
					<button class="shot" onclick={() => openLightbox(index)}>
						<img
							src="/previews/{image}"
							alt={labelFromFilename(image)}
							loading="lazy"
						/>
						<span class="shot-label">{labelFromFilename(image)}</span>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>

{#if activeIndex !== null}
	<div
		class="lightbox"
		onclick={(event) => event.target === event.currentTarget && closeLightbox()}
		onkeydown={(event) => {
			if (event.key === 'Escape') closeLightbox();
		}}
		role="dialog"
		aria-modal="true"
		aria-label="Image preview"
		tabindex="-1"
	>
		<div class="lightbox-frame">
			<button class="lightbox-close btn btn-sm btn-ghost" onclick={closeLightbox}>
				<Icon icon="ph:x-bold" width="14" />
				Close
			</button>

			<img
				class="lightbox-image"
				src="/previews/{images[activeIndex]}"
				alt={labelFromFilename(images[activeIndex])}
			/>

			<p class="lightbox-caption">
				{labelFromFilename(images[activeIndex])} — {activeIndex + 1} of {images.length}
			</p>

			{#if images.length > 1}
				<button class="lightbox-nav prev btn btn-sm" onclick={goPrev} aria-label="Previous image">
					<Icon icon="ph:caret-left-bold" width="16" />
				</button>
				<button class="lightbox-nav next btn btn-sm" onclick={goNext} aria-label="Next image">
					<Icon icon="ph:caret-right-bold" width="16" />
				</button>
			{/if}
		</div>
	</div>
{/if}

<style>
	.gallery-wrap {
		padding-bottom: 6rem;
	}

	.gallery {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(20rem, 1fr));
		gap: 0.75rem;
	}

	.shot {
		position: relative;
		display: block;
		padding: 0;
		aspect-ratio: 16 / 9;
		overflow: hidden;
		background: var(--color-surface);
		border: 1px solid var(--color-line);
		border-radius: var(--radius-panel);
		cursor: pointer;
		transition: border-color 0.15s;
	}

	.shot:hover {
		border-color: var(--color-line-strong);
	}

	.shot img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
		image-rendering: pixelated;
	}

	.shot-label {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 0;
		padding: 1.5rem 0.85rem 0.6rem;
		background: linear-gradient(to top, rgba(9, 11, 16, 0.92), transparent);
		color: var(--color-body);
		font-size: 0.78rem;
		text-align: left;
		opacity: 0;
		transition: opacity 0.15s;
	}

	.shot:hover .shot-label {
		opacity: 1;
	}

	.lightbox {
		position: fixed;
		inset: 0;
		z-index: 200;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		background: rgba(9, 11, 16, 0.96);
	}

	.lightbox-frame {
		position: relative;
		width: 100%;
		max-width: 68rem;
	}

	.lightbox-close {
		position: absolute;
		right: 0;
		top: -2.75rem;
	}

	.lightbox-image {
		display: block;
		width: 100%;
		height: auto;
		border: 1px solid var(--color-line-strong);
		border-radius: var(--radius-panel);
		image-rendering: pixelated;
	}

	.lightbox-caption {
		margin-top: 0.85rem;
		text-align: center;
		font-size: 0.8rem;
		color: var(--color-faint);
	}
</style>
