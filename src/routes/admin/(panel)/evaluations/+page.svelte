<script lang="ts">
	import { apiUrl } from '$lib/ts/api';
	import Pagination from '$lib/components/Pagination.svelte';

	let { data } = $props();
	let evaluations = $state<any[]>([]);
	let loading = $state(true);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.evaluationsPromise.then((rows) => {
			if (!active) return;
			evaluations = rows;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	let page = $state(1);
	const pageSize = 25;
	const pagedEvaluations = $derived(evaluations.slice((page - 1) * pageSize, page * pageSize));

	async function remove(id: string, name: string) {
		if (!confirm(`Delete evaluation for "${name}"? This cannot be undone.`)) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 204) {
				evaluations = evaluations.filter((e: any) => e.id !== id);
				message = `Deleted evaluation for ${name}.`;
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<h1>Evaluations</h1>
{#if message}<div class="message">{message}</div>{/if}

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Username</th>
				<th>Level</th>
				<th>Status</th>
				<th>Rate</th>
				<th>Date</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#if loading}
				<tr><td colspan="6">Loading evaluations...</td></tr>
			{:else}
				{#each pagedEvaluations as row}
					<tr>
						<td>{row.username}</td>
						<td>{row.level_id || 'farp'}</td>
						<td><span class="pill">{row.status.toUpperCase()}</span></td>
						<td>{row.success_rate != null ? `${row.success_rate}%` : '—'}</td>
						<td>{new Date(row.created_at).toLocaleString()}</td>
						<td>
							<div class="actions">
								<a class="admin-btn" href={`/admin/evaluations/${row.id}`}>View</a>
								<button class="admin-btn-danger" onclick={() => remove(row.id, row.username)}>Delete</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="6">No evaluations found.</td></tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<Pagination bind:page total={evaluations.length} {pageSize} />
