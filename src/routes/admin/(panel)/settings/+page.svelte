<script lang="ts">
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();
	let settings = $state<any>(null);
	let loading = $state(true);
	let saving = $state(false);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.settingsPromise.then((row) => {
			if (!active) return;
			settings = row;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	async function save(body: Record<string, unknown>, successMessage: string) {
		saving = true;
		message = '';
		try {
			const res = await fetch(apiUrl('/api/evaluations/settings'), {
				method: 'PUT',
				headers: { 'X-API-Key': data.adminKey, 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});
			if (!res.ok) {
				message = `Failed to save: ${res.status}`;
				return;
			}
			const updated = await res.json();
			if (settings && updated.levels != null) settings.levels = updated.levels;
			message = successMessage;
		} catch (err) {
			message = `Save failed: ${err}`;
		} finally {
			saving = false;
		}
	}

	function toggleLevel(levelId: string, enabled: boolean) {
		save({ level_id: levelId, enabled }, `Saved level settings.`);
	}

	let versionInput = $state('');

	$effect(() => {
		if (settings && versionInput === '') versionInput = settings.required_game_version ?? '';
	});

	async function saveVersion() {
		const version = versionInput.trim();
		if (!version) return;
		await save({ required_game_version: version }, `Only ${version} can submit entries now.`);
		if (settings) settings.required_game_version = version;
	}

	function levelSlug(levelId: string): string {
		const m = (levelId ?? '').match(/\d+/);
		return `level-${m ? m[0] : '1'}`;
	}
</script>

<h1>Settings</h1>
<p class="meta">Every entry is benchmarked with these static values so that different entries are tested under identical conditions.</p>

{#if loading}
	<p>Loading settings...</p>
{:else if !settings}
	<p>Could not load settings.</p>
{:else}
	{#if message}<div class="message">{message}</div>{/if}
	<p class="meta">Max parallel jobs is configured per worker on the <a href="/admin/workers">Workers</a> page.</p>

	<h2>Game version</h2>
	<p class="meta">
		Only this build may submit entries. Every other version is rejected with a 426, which keeps an
		older client from filing entries under level ids that have since moved.
	</p>
	<div class="actions">
		<input type="text" maxlength="20" bind:value={versionInput} />
		<button onclick={saveVersion} disabled={saving || versionInput.trim() === settings.required_game_version}>
			Require this version
		</button>
	</div>

	<h2>Level settings</h2>
	<p class="meta">Enable or disable levels. A disabled level rejects new benchmark submissions.</p>
	{#each settings.levels ?? [] as level}
		<div class="level-block">
			<div class="actions">
				<strong>{level.name}</strong>
				<label class="toggle">
					<input
						type="checkbox"
						checked={level.enabled}
						onchange={(e) => toggleLevel(level.id, e.currentTarget.checked)}
						disabled={saving}
					/>
					<span>{level.enabled ? 'Enabled' : 'Disabled'}</span>
				</label>
			</div>
			<p class="meta">
				Enemy start, ring sweep, per-trial seeds and other benchmark parameters live on the
				<a href={`/admin/settings/${levelSlug(level.id)}`}>{level.name} settings</a> page.
			</p>
		</div>
	{/each}
{/if}
