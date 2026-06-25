<script lang="ts">
	import { goto } from '$app/navigation';
	import AlgorithmView from '$lib/components/AlgorithmView.svelte';
	import { apiUrl } from '$lib/ts/api';

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
		if (!confirm('Delete this leaderboard entry? This cannot be undone.')) return;
		try {
			const res = await fetch(apiUrl(`/api/leaderboard/${data.id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 204) {
				await goto('/admin/leaderboard');
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<div class="max-w-4xl">
	<a href="/admin/leaderboard" class="text-sm text-sky-300 hover:text-sky-200">← All leaderboard entries</a>

	{#if message}<div class="mt-4 text-red-300 text-sm">{message}</div>{/if}

	{#if loading}
		<div class="mt-6 p-6 border-2 border-sky-500/20 bg-sky-500/5 text-sky-200 font-game tracking-wider text-center animate-pulse">
			Loading entry...
		</div>
	{:else if !entry}
		<div class="mt-6 p-6 border-2 border-red-500/30 bg-red-500/10 text-red-200 font-game tracking-wider text-center">
			Entry not found.
		</div>
	{:else}
		<div class="mt-3 flex flex-wrap items-start justify-between gap-4">
			<div>
				<h1 class="font-game text-3xl text-star-white" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
					{entry.username}
				</h1>
				<p class="text-sm text-text-muted mt-2">
					Best time {entry.time_seconds}s · submitted {new Date(entry.created_at).toLocaleString()}
				</p>
				<p class="text-[11px] text-text-muted/60 font-mono mt-1 break-all">{entry.id}</p>
			</div>
			<div class="flex gap-2">
				<a
					href={apiUrl(`/api/leaderboard/${entry.id}/export`)}
					class="px-4 py-2 border-2 border-sky-400/40 text-sky-200 font-game text-sm tracking-wider hover:bg-sky-500/15 transition-colors"
				>
					DOWNLOAD ZIP
				</a>
				<button
					onclick={remove}
					class="px-4 py-2 border-2 border-red-500/30 text-red-300 font-game text-sm tracking-wider hover:bg-red-500/15 transition-colors"
				>
					DELETE
				</button>
			</div>
		</div>

		<div class="h-px my-6" style="background: linear-gradient(to right, rgba(36,89,184,0.4), transparent)"></div>

		<div class="grid grid-cols-2 sm:grid-cols-3 gap-4 mb-8">
			<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5">
				<div class="text-xs text-text-muted tracking-wider font-game">TIME</div>
				<div class="text-2xl text-green-300 mt-1">{entry.time_seconds}s</div>
			</div>
		</div>

		<h2 class="font-sim text-xl text-star-white mb-4">Algorithm</h2>
		{#if entry.algorithm && entry.algorithm.length > 0}
			<div class="p-5 border-2 border-sky-500/20 bg-page-bg/60 overflow-x-auto">
				<AlgorithmView scripts={entry.algorithm} />
			</div>
		{:else}
			<div class="p-4 border-2 border-sky-500/20 bg-sky-500/5 text-text-muted text-sm">
				No algorithm data recorded for this entry.
			</div>
		{/if}
	{/if}
</div>
