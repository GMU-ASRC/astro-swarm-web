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

<p><a href="/admin/leaderboard">← All leaderboard entries</a></p>

{#if message}<div class="message">{message}</div>{/if}

{#if loading}
	<p>Loading entry...</p>
{:else if !entry}
	<p>Entry not found.</p>
{:else}
	<h1>{entry.username}</h1>
	<p class="meta">Best time {entry.time_seconds}s · submitted {new Date(entry.created_at).toLocaleString()}</p>
	<p class="meta">{entry.id}</p>

	<div class="actions">
		<a class="admin-btn" href={apiUrl(`/api/leaderboard/${entry.id}/export`)}>Download ZIP</a>
		<button class="admin-btn-danger" onclick={remove}>Delete</button>
	</div>

	<div class="stat-grid">
		<div class="stat">
			<div class="label">Time</div>
			<div>{entry.time_seconds}s</div>
		</div>
	</div>

	<h2>Algorithm</h2>
	{#if entry.algorithm && entry.algorithm.length > 0}
		<div class="admin-table-wrap">
			<AlgorithmView scripts={entry.algorithm} />
		</div>
	{:else}
		<p>No algorithm data recorded for this entry.</p>
	{/if}
{/if}
