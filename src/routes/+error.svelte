<script lang="ts">
	interface Props {
		status: number;
		error: { message?: string };
	}

	let { status, error }: Props = $props();

	const titles: Record<number, string> = {
		404: 'SIGNAL LOST',
		403: 'ACCESS DENIED',
		500: 'SWARM MALFUNCTION',
		503: 'SWARM OFFLINE',
	};

	const title = $derived(titles[status] ?? 'TRANSMISSION FAILED');
</script>

<svelte:head>
	<title>Error {status} — AstroSwarm</title>
</svelte:head>

<div class="relative z-1 min-h-screen flex flex-col items-center justify-center text-center px-8 max-sm:px-5 pt-20 pb-16">
	<p class="font-game text-[clamp(8rem,28vw,16rem)] text-star-white/[0.04] leading-none select-none pointer-events-none absolute">
		{status}
	</p>

	<div class="relative z-10 flex flex-col items-center">
		<p class="font-game text-accent-cyan text-[0.65rem] tracking-widest mb-5">
			ERROR {status}
		</p>

		<h1
			class="font-game text-[clamp(1.8rem,5vw,3.5rem)] text-star-white tracking-[0.04em] mb-4 leading-tight"
			style="text-shadow: 0 0 40px rgba(56,189,248,0.35), 0 0 80px rgba(36,89,184,0.2)"
		>
			{title}
		</h1>

		{#if error?.message}
			<p class="font-game text-[0.6rem] text-text-muted/60 tracking-widest mb-10 max-w-xs">
				{error.message}
			</p>
		{:else}
			<p class="font-game text-[0.6rem] text-text-muted/60 tracking-widest mb-10">
				THE SWARM ENCOUNTERED AN UNEXPECTED CONDITION
			</p>
		{/if}

		<div class="flex flex-col sm:flex-row gap-4 items-center justify-center w-full max-w-xs sm:max-w-none">
			<a
				href="/"
				class="font-game text-[1rem] tracking-[0.06em] py-2.75 px-4.5 bg-btn-press-bg text-star-white border-2 border-btn-press-border no-underline inline-block w-full sm:w-auto text-center transition-[background,border-color] duration-150 hover:bg-btn-hover-bg hover:border-btn-hover-border"
			>
				RETURN HOME
			</a>
			<button
				onclick={() => history.back()}
				class="font-game text-[1rem] tracking-[0.06em] py-2.75 px-4.5 bg-btn-bg text-btn-text border-2 border-btn-border w-full sm:w-auto text-center transition-[background,border-color,color] duration-150 hover:bg-btn-hover-bg hover:border-btn-hover-border hover:text-star-white cursor-pointer"
			>
				GO BACK
			</button>
		</div>
	</div>
</div>
