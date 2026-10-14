<script lang="ts">
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import type { SaveSettings } from '$lib/ts/adminSettings';

	let { seed, saving, onsave }: { seed: number; saving: boolean; onsave: SaveSettings } = $props();

	let seedInput = $state(0);
	let edited = $state(false);
	let confirming = $state(false);

	$effect(() => {
		if (!edited) seedInput = seed;
	});

	const changed = $derived(Number.isInteger(seedInput) && seedInput > 0 && seedInput !== seed);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!changed) return;
		const message = 'Base seed saved. Existing entries were graded under the previous seed.';
		if (await onsave({ seed: seedInput }, message)) edited = false;
	}

	async function regenerate() {
		confirming = false;
		if (await onsave({ regenerate_seeds: true }, 'Drew a new random base seed.')) edited = false;
	}
</script>

<form class="settings-panel" onsubmit={submit}>
	<div class="field-row">
		<label class="field">
			<span class="field-label">Base seed</span>
			<input
				type="number"
				step="1"
				min="1"
				max="2147483647"
				bind:value={seedInput}
				oninput={() => (edited = true)}
			/>
		</label>
		<button type="submit" disabled={saving || !changed}>Save seed</button>
		<button type="button" class="admin-btn-danger" disabled={saving} onclick={() => (confirming = true)}>
			Regenerate
		</button>
	</div>
	<p class="meta">
		Every trial's defender scatter, spawn bearings and match RNG are derived from this value, so the
		same entry and seed always reproduce the same result. Changing it only affects entries submitted
		or re-simulated afterwards.
	</p>
</form>

{#if confirming}
	<ConfirmDialog
		title="Regenerate the base seed?"
		message="A new random seed changes every derived seed."
		warning="New entries will no longer be comparable with existing ones until those are re-simulated."
		confirmLabel="Regenerate"
		danger
		onconfirm={regenerate}
		oncancel={() => (confirming = false)}
	/>
{/if}
