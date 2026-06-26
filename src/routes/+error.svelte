<script lang="ts">
	interface Props {
		status: number;
		error: { message?: string };
	}

	let { status, error }: Props = $props();

	const presets: Record<number, { title: string; blurb: string; accent: string }> = {
		404: {
			title: 'SIGNAL LOST',
			blurb: 'This coordinate is empty space — no swarm node answered the ping.',
			accent: '#38bdf8'
		},
		403: {
			title: 'ACCESS DENIED',
			blurb: 'Your clearance does not authorize entry to this sector.',
			accent: '#fbbf24'
		},
		500: {
			title: 'SWARM MALFUNCTION',
			blurb: 'A node faulted mid-transmission. Engineering has been pinged.',
			accent: '#f87171'
		},
		503: {
			title: 'SWARM OFFLINE',
			blurb: 'The hive is temporarily unreachable. Try again shortly.',
			accent: '#f87171'
		}
	};

	const preset = $derived(
		presets[status] ?? {
			title: 'TRANSMISSION FAILED',
			blurb: 'The swarm encountered an unexpected condition.',
			accent: '#38bdf8'
		}
	);
</script>

<svelte:head>
	<title>Error {status} — AstroSwarm</title>
</svelte:head>

<div
	class="relative z-1 min-h-screen flex flex-col items-center justify-center text-center px-8 max-sm:px-5 pt-20 pb-16"
	style="--accent: {preset.accent}"
>
	<p
		class="font-game text-[clamp(8rem,30vw,18rem)] leading-none select-none pointer-events-none absolute top-1/2 -translate-y-1/2"
		style="color: color-mix(in srgb, var(--accent) 6%, transparent)"
	>
		{status}
	</p>

	<div class="relative z-10 flex flex-col items-center w-full max-w-lg">
		<div class="radar mb-9" style="--accent: {preset.accent}">
			<span class="radar-dot"></span>
		</div>

		<p
			class="font-game text-[0.62rem] tracking-[0.35em] mb-4"
			style="color: var(--accent)"
		>
			ERROR {status}
		</p>

		<h1
			class="font-game text-[clamp(1.9rem,5vw,3.6rem)] text-star-white tracking-[0.04em] mb-5 leading-tight"
			style="text-shadow: 0 0 40px color-mix(in srgb, var(--accent) 45%, transparent), 0 0 90px color-mix(in srgb, var(--accent) 20%, transparent)"
		>
			{preset.title}
		</h1>

		<p class="font-sim text-sm text-text-muted/80 leading-relaxed mb-6 max-w-md">
			{preset.blurb}
		</p>

		{#if error?.message}
			<p
				class="font-mono text-[0.7rem] text-text-muted/70 mb-9 px-4 py-2 border border-star-white/10 bg-star-white/[0.03] break-words max-w-md"
			>
				&gt; {error.message}
			</p>
		{:else}
			<div class="mb-9"></div>
		{/if}

		<div class="flex flex-col sm:flex-row gap-4 items-center justify-center w-full max-w-xs sm:max-w-none">
			<a
				href="/"
				class="font-game text-[1rem] tracking-[0.06em] py-2.75 px-5 bg-btn-press-bg text-star-white border-2 border-btn-press-border no-underline inline-block w-full sm:w-auto text-center transition-[background,border-color] duration-150 hover:bg-btn-hover-bg hover:border-btn-hover-border"
			>
				RETURN HOME
			</a>
			<button
				onclick={() => history.back()}
				class="font-game text-[1rem] tracking-[0.06em] py-2.75 px-5 bg-btn-bg text-btn-text border-2 border-btn-border w-full sm:w-auto text-center transition-[background,border-color,color] duration-150 hover:bg-btn-hover-bg hover:border-btn-hover-border hover:text-star-white cursor-pointer"
			>
				GO BACK
			</button>
		</div>
	</div>
</div>

<style>
	.radar {
		position: relative;
		width: 96px;
		height: 96px;
		border-radius: 50%;
		border: 1px solid color-mix(in srgb, var(--accent) 35%, transparent);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.radar::before,
	.radar::after {
		content: '';
		position: absolute;
		inset: 0;
		border-radius: 50%;
		border: 1px solid var(--accent);
		opacity: 0;
		animation: radar-pulse 2.6s ease-out infinite;
	}

	.radar::after {
		animation-delay: 1.3s;
	}

	.radar-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: var(--accent);
		box-shadow: 0 0 16px var(--accent);
	}

	@keyframes radar-pulse {
		0% {
			transform: scale(0.55);
			opacity: 0.65;
		}
		100% {
			transform: scale(1.8);
			opacity: 0;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.radar::before,
		.radar::after {
			animation: none;
		}
	}
</style>
