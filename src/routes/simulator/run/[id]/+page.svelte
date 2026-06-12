<script lang="ts">
	import Icon from '@iconify/svelte';
	import { formatRelativeTime, formatDuration, type SimRunItem } from '$lib/ts/simulator';
	import { apiUrl } from '$lib/ts/api';

	import { onMount, onDestroy } from 'svelte';

	interface PageData {
		run: SimRunItem | null;
		config?: any;
		error: string | null;
	}

	let { data }: { data: PageData } = $props();

	const run = $derived(data.run);
	const config = $derived(data.config);
	const error = $derived(data.error);

	let blocklyDiv: HTMLDivElement | undefined = $state();
	let workspace: any;

	onMount(async () => {
		if (config) {
			try {
				const Blockly = await import('blockly/core');
				await import('blockly/blocks');
				const En = await import('blockly/msg/en');
				Blockly.setLocale(En.default || En);

				if (blocklyDiv) {
					workspace = Blockly.inject(blocklyDiv, {
						readOnly: true,
						scrollbars: true,
						theme: Blockly.Themes?.Dark || undefined
					});
					Blockly.serialization.workspaces.load(config, workspace);
				}
			} catch (e) {
				console.error("Failed to load blockly config:", e);
			}
		}
	});

	onDestroy(() => {
		if (workspace) {
			workspace.dispose();
		}
	});
</script>

<svelte:head>
	<title>{run ? `${run.title} — AstroSwarm` : 'Run Not Found — AstroSwarm'}</title>
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-10">
		<a href="/simulator" class="inline-flex items-center gap-2 font-sim text-xs text-text-muted hover:text-star-white transition-colors duration-150 no-underline mb-6">
			<Icon icon="ph:arrow-left" width="16" />
			Back to Gallery
		</a>
		
		{#if error || !run}
			<div class="py-12 px-6 text-center border border-dashed border-accent-blue/20 bg-panel-bg/40">
				<div class="flex justify-center mb-4 opacity-40">
					<Icon icon="ph:warning-circle" width="48" color="var(--color-text-muted)" />
				</div>
				<h1 class="font-sim text-xl font-bold text-star-white mb-2">Run Not Found</h1>
				<p class="font-sim text-sm text-text-muted m-0">{error || "This run does not exist or has been removed."}</p>
			</div>
		{:else}
			<h1 class="font-sim text-[clamp(1.8rem,4vw,2.5rem)] font-bold text-star-white leading-tight mb-2">
				{run.title}
			</h1>
			<p class="font-sim text-sm text-text-muted/80 leading-relaxed mb-6">
				Uploaded by <span class="text-accent-cyan font-medium">{run.author}</span> • {formatRelativeTime(run.created_at)}
			</p>
			
			<div class="mb-8 border-2 border-accent-blue/30 bg-panel-bg shadow-xl">
				<!-- svelte-ignore a11y_media_has_caption -->
				<video 
					class="w-full aspect-video bg-black/50 block" 
					controls 
					preload="metadata"
					poster="{apiUrl(`/api/runs/${run.id}/thumbnail`)}"
				>
					<source src="{apiUrl(`/api/runs/${run.id}/video`)}" type="video/mp4" />
					Your browser does not support the video tag.
				</video>
			</div>

			<div class="grid grid-cols-[1fr_300px] max-md:grid-cols-1 gap-8 items-start">
				<!-- Details -->
				<div class="flex flex-col gap-6">
					{#if run.description}
						<div>
							<h3 class="font-sim text-sm font-semibold text-star-white mb-2 tracking-wide uppercase">Description</h3>
							<p class="font-sim text-sm text-text-muted leading-relaxed whitespace-pre-wrap m-0">{run.description}</p>
						</div>
					{/if}
					
					<div>
						<h3 class="font-sim text-sm font-semibold text-star-white mb-3 tracking-wide uppercase">Species Information</h3>
						<div class="grid grid-cols-2 max-sm:grid-cols-1 gap-3">
							{#each run.species as sp}
								<div class="bg-panel-bg/70 border border-accent-blue/20 p-3 flex items-center gap-3">
									<div class="w-4 h-4 shadow-sm" style="background: {sp.color}; box-shadow: 0 0 8px {sp.color}"></div>
									<span class="font-sim text-sm font-medium text-star-white">{sp.name}</span>
								</div>
							{/each}
						</div>
					</div>
				</div>

				<!-- Stats sidebar -->
				<div class="bg-panel-bg/70 border border-accent-blue/20 p-5 flex flex-col gap-5">
					<a
						href="{apiUrl(`/api/runs/${run.id}/download`)}"
						class="flex items-center justify-center gap-2 font-sim text-sm font-semibold py-3 px-4 bg-accent-blue hover:bg-btn-hover-bg text-star-white transition-colors duration-150 no-underline shadow-[0_0_15px_rgba(36,89,184,0.3)]"
						download
					>
						<Icon icon="ph:download-simple" width="18" />
						Download Zip
					</a>
					
					<div class="grid grid-cols-2 gap-4">
						<div>
							<div class="font-sim text-[0.65rem] text-text-muted/60 uppercase tracking-widest mb-1">Robots</div>
							<div class="font-sim text-base font-semibold text-star-white">{run.robot_count}</div>
						</div>
						<div>
							<div class="font-sim text-[0.65rem] text-text-muted/60 uppercase tracking-widest mb-1">Duration</div>
							<div class="font-sim text-base font-semibold text-star-white">{formatDuration(run.duration_seconds)}</div>
						</div>
						<div>
							<div class="font-sim text-[0.65rem] text-text-muted/60 uppercase tracking-widest mb-1">Frame Count</div>
							<div class="font-sim text-base font-semibold text-star-white">{run.frame_count || Math.round(run.duration_seconds * 10)}</div>
						</div>
						<div>
							<div class="font-sim text-[0.65rem] text-text-muted/60 uppercase tracking-widest mb-1">Downloads</div>
							<div class="font-sim text-base font-semibold text-star-white">{run.download_count}</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Blockly Workspace -->
			{#if config}
				<div class="mt-8 border-2 border-accent-blue/30 bg-panel-bg shadow-xl">
					<div class="bg-black/40 border-b border-accent-blue/20 p-3 px-5 flex items-center justify-between">
						<h3 class="font-sim text-sm font-semibold text-star-white m-0 tracking-wide uppercase flex items-center gap-2">
							<Icon icon="ph:code-block" width="18" />
							Algorithm Source
						</h3>
					</div>
					<div class="w-full h-[500px] relative bg-black/20" bind:this={blocklyDiv}></div>
				</div>
			{/if}
		{/if}
	</div>
</div>
