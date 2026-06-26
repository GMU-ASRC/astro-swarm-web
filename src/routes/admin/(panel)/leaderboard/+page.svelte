<script lang="ts">
	import { apiUrl } from '$lib/ts/api';
	import Pagination from '$lib/components/Pagination.svelte';

	let { data } = $props();
	let leaderboard = $state<any[]>([]);
	let loading = $state(true);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.leaderboardPromise.then((rows) => {
			if (!active) return;
			leaderboard = rows;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	let page = $state(1);
	const pageSize = 25;
	const pagedLeaderboard = $derived(leaderboard.slice((page - 1) * pageSize, page * pageSize));

	function formatTime(seconds: number): string {
		if (seconds == null) return '—';
		return Math.round(seconds) + 's';
	}

	async function remove(id: string, name: string) {
		if (!confirm(`Delete leaderboard entry for "${name}"? This cannot be undone.`)) return;
		try {
			const res = await fetch(apiUrl(`/api/leaderboard/${id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 204) {
				leaderboard = leaderboard.filter((e: any) => e.id !== id);
				message = `Deleted entry for ${name}.`;
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<h1>Leaderboard</h1>
{#if message}<div class="message">{message}</div>{/if}

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Username</th>
				<th>Time (s)</th>
				<th>Date</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#if loading}
				<tr><td colspan="4">Loading leaderboard...</td></tr>
			{:else}
				{#each pagedLeaderboard as row}
					<tr>
						<td>{row.username}</td>
						<td>{formatTime(row.time_seconds)}</td>
						<td>{new Date(row.created_at).toLocaleString()}</td>
						<td>
							<div class="actions">
								<a class="admin-btn" href={`/admin/leaderboard/${row.id}`}>View</a>
								<button class="admin-btn-danger" onclick={() => remove(row.id, row.username)}>Delete</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="4">No leaderboard entries found.</td></tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<Pagination bind:page total={leaderboard.length} {pageSize} />
