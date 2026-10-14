<script lang="ts">
	import {
		KIND_LABELS,
		levelProfile,
		type AdminLevel,
		type AdminSettings,
		type SaveSettings
	} from '$lib/ts/adminSettings';

	let {
		settings,
		saving,
		onsave
	}: { settings: AdminSettings; saving: boolean; onsave: SaveSettings } = $props();

	const rows = $derived(
		settings.levels.map((level) => ({ level, profile: levelProfile(level, settings) }))
	);
	const enabledCount = $derived(settings.levels.filter((level) => level.enabled).length);

	async function toggle(level: AdminLevel, input: HTMLInputElement) {
		const enabled = input.checked;
		const verb = enabled ? 'enabled' : 'disabled';
		if (!(await onsave({ level_id: level.id, enabled }, `${level.name} ${verb}.`))) {
			input.checked = !enabled;
		}
	}
</script>

<p class="filter-summary">
	{enabledCount} of {settings.levels.length} levels accept submissions. D is the number of defenders
	in a run.
</p>

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Level</th>
				<th>Type</th>
				<th>Evaders</th>
				<th>Ends when</th>
				<th>Benchmark budget</th>
				<th>Accepting entries</th>
			</tr>
		</thead>
		<tbody>
			{#each rows as { level, profile } (level.id)}
				<tr id={`level-${profile.number}`} class:disabled-row={!level.enabled}>
					<td>
						<div class="level-name">
							<strong>{level.name}</strong>
							<code>{level.id}</code>
						</div>
					</td>
					<td>
						<span class="pill" class:status-queued={profile.kind !== 'recorded'}>
							{KIND_LABELS[profile.kind]}
						</span>
						{#if profile.attrition}<span class="pill status-failed">Attrition</span>{/if}
					</td>
					<td>{profile.arrival}</td>
					<td>{profile.endsWhen}</td>
					<td class="budget">{profile.budget}</td>
					<td class="switch-cell">
						<label class="switch">
							<input
								type="checkbox"
								checked={level.enabled}
								disabled={saving}
								onchange={(event) => toggle(level, event.currentTarget)}
							/>
							<span class="track"></span>
							<span>{level.enabled ? 'On' : 'Off'}</span>
						</label>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<style>
	.level-name {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		min-width: 12rem;
	}

	.level-name strong {
		color: var(--color-heading);
		font-weight: 600;
	}

	.level-name code {
		align-self: flex-start;
		font-size: 0.75rem;
	}

	.switch-cell {
		white-space: nowrap;
	}

	td .pill + .pill {
		margin-left: 0.35rem;
	}

	.budget {
		color: var(--color-dim);
		font-size: 0.8rem;
		min-width: 14rem;
	}

	tr:target {
		background: color-mix(in srgb, var(--color-brand) 10%, transparent);
	}

	.disabled-row td:not(:last-child) {
		opacity: 0.55;
	}
</style>
