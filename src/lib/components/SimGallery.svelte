<script lang="ts">
	import Icon from '@iconify/svelte';
	import { formatRelativeTime, formatDuration, type SimConfigItem, type SimRunItem } from '$lib/ts/simulator';

	interface Props {
		configs: SimConfigItem[];
		runs: SimRunItem[];
		apiError: boolean;
	}

	let { configs, runs, apiError }: Props = $props();

	let activeTab = $state<'configs' | 'runs'>('configs');

	const visibleConfigs = $derived(configs);
	const visibleRuns = $derived(runs);
</script>

<div class="flex flex-col gap-0">
	<div class="flex items-center justify-between mb-5">
		<h2 class="font-sim text-lg font-semibold text-star-white m-0">Community Gallery</h2>
		<div class="flex border border-accent-blue/25">
			<button
				class="font-sim text-xs px-3 py-1.5 transition-colors duration-150 cursor-pointer border-none"
				class:bg-accent-blue={activeTab === 'configs'}
				class:text-star-white={activeTab === 'configs'}
				class:bg-transparent={activeTab !== 'configs'}
				class:text-text-muted={activeTab !== 'configs'}
				onclick={() => (activeTab = 'configs')}
			>
				Configs ({configs.length})
			</button>
			<button
				class="font-sim text-xs px-3 py-1.5 transition-colors duration-150 cursor-pointer border-none border-l border-accent-blue/25"
				class:bg-accent-blue={activeTab === 'runs'}
				class:text-star-white={activeTab === 'runs'}
				class:bg-transparent={activeTab !== 'runs'}
				class:text-text-muted={activeTab !== 'runs'}
				onclick={() => (activeTab = 'runs')}
			>
				Runs ({runs.length})
			</button>
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

	{:else if activeTab === 'configs'}
		{#if visibleConfigs.length === 0}
			<div class="py-12 px-6 text-center border border-dashed border-accent-blue/20">
				<div class="flex justify-center mb-4 opacity-40">
					<Icon icon="ph:file-code" width="48" color="var(--color-text-muted)" />
				</div>
				<p class="font-sim text-sm text-text-muted">No configs shared yet. Be the first!</p>
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				{#each visibleConfigs as config (config.id)}
					<div class="bg-panel-bg/70 border border-accent-blue/20 px-5 py-4 grid grid-cols-[1fr_auto] gap-x-4 gap-y-2 items-start transition-[border-color,background] duration-150 hover:border-btn-hover-border/35 hover:bg-panel-bg">
						<div>
							<div class="flex items-center gap-2 mb-1 flex-wrap">
								<p class="font-sim text-sm font-semibold text-star-white m-0">{config.title}</p>
								<span class="font-sim text-xs px-1.5 py-0.5 border border-accent-blue/30 text-text-muted/70 uppercase tracking-wide">
									{config.file_type}
								</span>
							</div>
							{#if config.description}
								<p class="font-sim text-xs text-text-muted m-0 leading-snug">{config.description}</p>
							{/if}
						</div>
						<div class="font-sim text-xs text-text-muted/50 text-right whitespace-nowrap">
							<div>{formatRelativeTime(config.created_at)}</div>
							<div class="mt-1">{config.robot_count} robots</div>
						</div>
						<div class="col-span-full flex items-center justify-between gap-3 flex-wrap mt-1">
							<div class="flex gap-1.5 flex-wrap">
								{#each config.species as sp}
									<span
										class="font-sim text-xs px-2 py-0.5 font-medium"
										style="background: {sp.color}22; color: {sp.color}; border: 1px solid {sp.color}44"
									>
										{sp.name}
									</span>
								{/each}
							</div>
							<div class="flex items-center gap-1 text-text-muted/40">
								<Icon icon="ph:download-simple" width="12" />
								<span class="font-sim text-xs">{config.download_count}</span>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}

	{:else}
		{#if visibleRuns.length === 0}
			<div class="py-12 px-6 text-center border border-dashed border-accent-blue/20">
				<div class="flex justify-center mb-4 opacity-40">
					<Icon icon="ph:film-strip" width="48" color="var(--color-text-muted)" />
				</div>
				<p class="font-sim text-sm text-text-muted">No run recordings shared yet. Be the first!</p>
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				{#each visibleRuns as run (run.id)}
					<div class="bg-panel-bg/70 border border-accent-blue/20 px-5 py-4 grid grid-cols-[1fr_auto] gap-x-4 gap-y-2 items-start transition-[border-color,background] duration-150 hover:border-btn-hover-border/35 hover:bg-panel-bg">
						<div>
							<p class="font-sim text-sm font-semibold text-star-white m-0 mb-1">{run.title}</p>
							{#if run.description}
								<p class="font-sim text-xs text-text-muted m-0 leading-snug">{run.description}</p>
							{/if}
						</div>
						<div class="font-sim text-xs text-text-muted/50 text-right whitespace-nowrap">
							<div>{formatRelativeTime(run.created_at)}</div>
							<div class="mt-1">{run.robot_count} robots</div>
						</div>
						<div class="col-span-full flex items-center justify-between gap-3 flex-wrap mt-1">
							<div class="flex gap-1.5 flex-wrap">
								{#each run.species as sp}
									<span
										class="font-sim text-xs px-2 py-0.5 font-medium"
										style="background: {sp.color}22; color: {sp.color}; border: 1px solid {sp.color}44"
									>
										{sp.name}
									</span>
								{/each}
							</div>
							<div class="flex items-center gap-3 text-text-muted/40">
								<div class="flex items-center gap-1">
									<Icon icon="ph:clock" width="12" />
									<span class="font-sim text-xs">{formatDuration(run.duration_seconds)}</span>
								</div>
								<div class="flex items-center gap-1">
									<Icon icon="ph:download-simple" width="12" />
									<span class="font-sim text-xs">{run.download_count}</span>
								</div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</div>
