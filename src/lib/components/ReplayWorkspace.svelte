<script lang="ts">
	import FarpReplay from '$lib/components/FarpReplay.svelte';
	import type { Replay } from '$lib/ts/evaluation';
	import type { ReplayGroup } from '$lib/ts/replay';

	let {
		groups,
		replay,
		meta = '',
		mode = 'defence',
		empty = 'Pick a run to replay it.'
	}: {
		groups: ReplayGroup[];
		replay: Replay | null;
		meta?: string;
		mode?: 'defence' | 'swarm';
		empty?: string;
	} = $props();
</script>

<div class="workspace">
	<div class="picker">
		{#each groups as group}
			{#if group.cells.length > 0}
				<div class="group">
					<p class="group-label">{group.label}</p>
					<div class="cells">
						{#each group.cells as cell (cell.id)}
							<button
								type="button"
								title={cell.title}
								aria-label={cell.title}
								onclick={cell.select}
								class="cell tone-{cell.tone}"
								class:selected={cell.selected}
							>
								{cell.label}
							</button>
						{/each}
					</div>
				</div>
			{/if}
		{/each}
	</div>

	<div class="stage">
		{#if replay}
			<FarpReplay {replay} {mode} />
			{#if meta}
				<p class="stage-meta">{meta}</p>
			{/if}
		{:else}
			<div class="stage-empty">{empty}</div>
		{/if}
	</div>
</div>

<style>
	.workspace {
		display: grid;
		grid-template-columns: minmax(10rem, 14rem) minmax(0, 1fr);
		gap: 1.5rem;
		align-items: start;
		margin-top: 1.25rem;
	}

	@media (max-width: 60rem) {
		.workspace {
			grid-template-columns: minmax(0, 1fr);
		}
	}

	.picker {
		display: flex;
		flex-direction: column;
		gap: 1.1rem;
		max-height: 30rem;
		overflow-y: auto;
		padding-right: 0.35rem;
	}

	.group-label {
		margin-bottom: 0.5rem;
		font-size: 0.68rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--color-faint);
	}

	.cells {
		display: flex;
		flex-wrap: wrap;
		gap: 0.3rem;
	}

	.cell {
		width: 2.1rem;
		height: 2.1rem;
		border: 2px solid transparent;
		border-radius: 0;
		color: #ffffff;
		font-size: 0.68rem;
		cursor: pointer;
		transition:
			filter 0.15s,
			border-color 0.15s;
	}

	.cell:hover {
		filter: brightness(1.25);
	}

	.cell.selected {
		border-color: var(--color-heading);
	}

	.tone-win {
		background: color-mix(in srgb, var(--color-win) 70%, #000000);
	}

	.tone-loss {
		background: color-mix(in srgb, var(--color-loss) 70%, #000000);
	}

	.tone-timeout {
		background: color-mix(in srgb, var(--color-warn) 70%, #000000);
	}

	.stage {
		position: sticky;
		top: 1rem;
	}

	.stage-meta {
		margin-top: 0.7rem;
		font-size: 0.75rem;
		color: var(--color-faint);
	}

	.stage-empty {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 14rem;
		border: 1px dashed var(--color-line);
		border-radius: 4px;
		font-size: 0.8rem;
		color: var(--color-faint);
	}
</style>
