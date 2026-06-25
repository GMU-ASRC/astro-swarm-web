<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiUrl } from '$lib/ts/api';
	import EntryDetail from '$lib/components/EntryDetail.svelte';

	let { data } = $props();
	let entry = $state<Record<string, any> | null>(null);
	let loading = $state(true);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.entryPromise.then((row) => {
			if (!active) return;
			entry = row;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	async function remove() {
		if (!confirm('Delete this evaluation? This cannot be undone.')) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${data.id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 204) {
				await goto('/admin/evaluations');
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<div class="max-w-4xl">
	<a href="/admin/evaluations" class="text-sm text-sky-300 hover:text-sky-200">← All evaluations</a>
	<div class="flex items-center justify-between gap-4 mt-3 mb-6">
		<h1 class="font-game text-2xl text-star-white">Evaluation</h1>
		{#if entry}
			<button onclick={remove} class="px-4 py-2 border-2 border-red-500/30 text-red-300 font-game text-sm tracking-wider hover:bg-red-500/15 transition-colors">
				DELETE
			</button>
		{/if}
	</div>

	{#if message}<div class="mb-4 text-sky-200 text-sm">{message}</div>{/if}

	{#if loading}
		<div class="p-6 border-2 border-sky-500/20 bg-sky-500/5 text-sky-200 font-game tracking-wider text-center animate-pulse">
			Loading entry...
		</div>
	{:else if !entry}
		<div class="p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center">
			Entry not found.
		</div>
	{:else}
		<EntryDetail record={entry} />
	{/if}
</div>
