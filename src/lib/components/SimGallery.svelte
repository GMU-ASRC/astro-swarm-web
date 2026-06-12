<script lang="ts">
	import Icon from '@iconify/svelte';
	import { formatRelativeTime, formatDuration, type SimRunItem } from '$lib/ts/simulator';
	import { apiUrl } from '$lib/ts/api';

	interface Props {
		runs: SimRunItem[];
		apiError: boolean;
	}

	let { runs, apiError }: Props = $props();

	const visibleRuns = $derived(runs);
</script>

<div class="flex flex-col gap-0">
	<div class="flex items-center justify-between mb-5">
		<h2 class="font-sim text-lg font-semibold text-star-white m-0">Community Gallery</h2>
		<div class="font-sim text-xs text-text-muted">
			Runs ({runs.length})
		</div>
	</div>

	{#if apiError}
		<div class="py-12 px-6 text-center border border-dashed border-accent-blue/20">
			<div class="flex justify-center mb-4 opacity-40">
				<Icon icon="ph:warning-circle" width="48" color="var(--color-text-muted)" />
			</div>
			<p class="font-sim text-sm text-text-muted mb-1">Could not connect to the API.</p>
			<p class="font-sim text-xs text-text-muted/50">Check that the server is running and try again.</p>
		</div>
	{:else}
		{#if visibleRuns.length === 0}
			<div class="py-12 px-6 text-center border border-dashed border-accent-blue/20">
				<div class="flex justify-center mb-4 opacity-40">
					<Icon icon="ph:film-strip" width="48" color="var(--color-text-muted)" />
				</div>
				<p class="font-sim text-sm text-text-muted">No run recordings shared yet. Be the first!</p>
			</div>
		{:else}
			<div class="grid grid-cols-2 max-lg:grid-cols-1 gap-4">
				{#each visibleRuns as run (run.id)}
					<a href="/simulator/run/{run.id}" class="group relative bg-panel-bg/70 border border-accent-blue/20 overflow-hidden flex flex-col transition-[border-color,transform] duration-150 hover:border-accent-cyan/50 hover:-translate-y-1 text-current no-underline">
						<!-- Thumbnail Area -->
						<div class="relative w-full aspect-video bg-black/60 border-b border-accent-blue/20 overflow-hidden">
							{#if run.thumbnail_filename}
								<img src={apiUrl(`/api/runs/${run.id}/thumbnail`)} alt="{run.title} thumbnail" class="w-full h-full object-cover opacity-80 group-hover:opacity-100 transition-opacity duration-300" loading="lazy" />
							{:else}
								<div class="absolute inset-0 flex items-center justify-center opacity-30">
									<Icon icon="ph:film-strip" width="32" />
								</div>
							{/if}
							<!-- Duration badge -->
							<div class="absolute bottom-2 right-2 bg-black/80 px-2 py-0.5 font-sim text-[0.65rem] font-medium text-star-white rounded-sm border border-white/10">
								{formatDuration(run.duration_seconds)}
							</div>
							
							<!-- Play icon overlay on hover -->
							<div class="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-300 bg-black/30 backdrop-blur-[2px]">
								<div class="w-12 h-12 rounded-full bg-accent-cyan/90 text-black flex items-center justify-center pl-1 shadow-[0_0_15px_rgba(42,212,255,0.5)]">
									<Icon icon="ph:play-fill" width="24" />
								</div>
							</div>
						</div>
						
						<!-- Details Area -->
						<div class="p-4 flex flex-col flex-1">
							<div class="flex justify-between items-start gap-2 mb-2">
								<p class="font-sim text-sm font-semibold text-star-white m-0 line-clamp-1 flex-1">{run.title}</p>
								<div class="flex items-center gap-1 text-text-muted/60 bg-black/30 px-1.5 py-0.5 rounded-sm">
									<Icon icon="ph:download-simple" width="12" />
									<span class="font-sim text-[0.65rem]">{run.download_count}</span>
								</div>
							</div>
							
							<div class="flex flex-wrap gap-1.5 mb-3">
								{#each run.species as sp}
									<span class="w-2.5 h-2.5 rounded-full shadow-sm" style="background: {sp.color}; box-shadow: 0 0 4px {sp.color}88" title={sp.name}></span>
								{/each}
							</div>
							
							<div class="mt-auto flex justify-between items-center font-sim text-[0.7rem] text-text-muted/50 border-t border-white/5 pt-3">
								<span class="truncate pr-2">{run.author}</span>
								<span class="whitespace-nowrap">{formatRelativeTime(run.created_at)}</span>
							</div>
						</div>
					</a>
				{/each}
			</div>
		{/if}
	{/if}
</div>
