<script lang="ts">
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import type { SimulatorEntry } from '$lib/ts/simulator';
	import { blockNamesFor, speciesConfigRows } from '$lib/ts/simulator';

	let { entry }: { entry: SimulatorEntry } = $props();

	const ARENA_TAB = '__arena__';
	const ARENA_COLOR = '#c94f80';

	let selectedTab = $state('');

	const names = $derived(blockNamesFor(entry));
	const activeTab = $derived(selectedTab || entry.species[0]?.id || ARENA_TAB);
	const activeSpecies = $derived(entry.species.find((species) => species.id === activeTab));
	const configRows = $derived(activeSpecies ? speciesConfigRows(activeSpecies.config) : []);
	const scripts = $derived(activeSpecies ? (entry.behaviors[activeSpecies.id] ?? []) : entry.arena_program);
</script>

<div class="logic">
	<div class="tabs" role="tablist">
		{#each entry.species as species}
			<button
				type="button"
				role="tab"
				class="tab"
				class:active={activeTab === species.id}
				aria-selected={activeTab === species.id}
				style={`--tab-color:${species.color}`}
				onclick={() => (selectedTab = species.id)}
			>
				{species.name}
			</button>
		{/each}
		<button
			type="button"
			role="tab"
			class="tab"
			class:active={activeTab === ARENA_TAB}
			aria-selected={activeTab === ARENA_TAB}
			style={`--tab-color:${ARENA_COLOR}`}
			onclick={() => (selectedTab = ARENA_TAB)}
		>
			Arena program
		</button>
	</div>

	<div class="panel card">
		{#if activeSpecies}
			<div class="panel-head">
				<span class="swatch" style={`background:${activeSpecies.color}`}></span>
				<h3 class="panel-title">{activeSpecies.name}</h3>
			</div>
			{#if configRows.length > 0}
				<dl class="config">
					{#each configRows as row}
						<div class="config-row">
							<dt>{row.label}</dt>
							<dd>{row.value}</dd>
						</div>
					{/each}
				</dl>
			{/if}
		{:else}
			<div class="panel-head">
				<span class="swatch" style={`background:${ARENA_COLOR}`}></span>
				<h3 class="panel-title">Arena program</h3>
			</div>
			<p class="panel-note">
				Runs once for the whole arena rather than once per robot, and drives the spawn zones.
			</p>
		{/if}

		<AlgorithmView
			{scripts}
			{names}
			empty={activeSpecies ? 'This species has no logic blocks.' : 'This entry has no arena program.'}
		/>

		{#if entry.variables.length > 0}
			<div class="variables">
				<span class="variables-label">Variables</span>
				{#each entry.variables as variable}
					<span class="badge">{variable.name} : {variable.type}</span>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.logic {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.tabs {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
	}

	.tab {
		padding: 0.4rem 0.9rem;
		border: 1px solid var(--tab-color);
		border-radius: 3px;
		background: color-mix(in srgb, var(--tab-color) 10%, transparent);
		color: var(--color-heading);
		font-size: 0.8rem;
		cursor: pointer;
	}

	.tab:hover {
		background: color-mix(in srgb, var(--tab-color) 22%, transparent);
	}

	.tab.active {
		background: var(--tab-color);
		color: #ffffff;
	}

	.panel {
		display: flex;
		flex-direction: column;
		gap: 1.1rem;
		padding: 1.4rem;
	}

	.panel-head {
		display: flex;
		align-items: center;
		gap: 0.55rem;
	}

	.panel-title {
		font-size: 1rem;
		font-weight: 600;
	}

	.panel-note {
		font-size: 0.82rem;
		color: var(--color-dim);
	}

	.swatch {
		width: 0.8rem;
		height: 0.8rem;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.config {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
		gap: 0.6rem;
		margin: 0;
	}

	.config-row {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}

	.config-row dt {
		font-size: 0.7rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--color-faint);
	}

	.config-row dd {
		margin: 0;
		font-size: 0.9rem;
		font-variant-numeric: tabular-nums;
		color: var(--color-heading);
	}

	.variables {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.75rem;
	}

	.variables-label {
		margin-right: 0.3rem;
		color: var(--color-faint);
	}
</style>
