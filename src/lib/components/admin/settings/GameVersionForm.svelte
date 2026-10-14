<script lang="ts">
	import type { SaveSettings } from '$lib/ts/adminSettings';

	let {
		currentVersion,
		saving,
		onsave
	}: { currentVersion: string; saving: boolean; onsave: SaveSettings } = $props();

	let versionInput = $state('');
	let edited = $state(false);

	$effect(() => {
		if (!edited) versionInput = currentVersion;
	});

	const trimmed = $derived(versionInput.trim());
	const changed = $derived(trimmed !== '' && trimmed !== currentVersion);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!changed) return;
		if (await onsave({ required_game_version: trimmed }, `Only ${trimmed} can submit entries now.`)) {
			edited = false;
		}
	}
</script>

<form class="settings-panel" onsubmit={submit}>
	<div class="current">
		<span class="field-label">Required build</span>
		<span class="pill status-done">{currentVersion}</span>
	</div>
	<div class="field-row">
		<label class="field">
			<span class="field-label">New version</span>
			<input type="text" maxlength="20" placeholder="v0.0.9" bind:value={versionInput} oninput={() => (edited = true)} />
		</label>
		<button type="submit" disabled={saving || !changed}>Require this version</button>
	</div>
	<p class="meta">
		The game sends <code>v</code> plus <code>config/version</code> from <code>project.godot</code>.
		Any other build is refused with a 426.
	</p>
</form>

<style>
	.current {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 1rem;
	}
</style>
