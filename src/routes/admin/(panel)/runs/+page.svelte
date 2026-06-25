<script lang="ts">
	import { apiUrl } from '$lib/ts/api';
	import Pagination from '$lib/components/Pagination.svelte';

	let { data } = $props();
	let runs = $state<any[]>([]);
	let loading = $state(true);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.runsPromise.then((rows) => {
			if (!active) return;
			runs = rows;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	let page = $state(1);
	const pageSize = 25;
	const pagedRuns = $derived(runs.slice((page - 1) * pageSize, page * pageSize));

	async function remove(id: string, title: string) {
		if (!confirm(`Delete simulator run "${title}"? This cannot be undone.`)) return;
		try {
			const res = await fetch(apiUrl(`/api/runs/${id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 204) {
				runs = runs.filter((e: any) => e.id !== id);
				message = `Deleted run "${title}".`;
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<h1>Simulator Runs</h1>
{#if message}<div class="message">{message}</div>{/if}

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Title</th>
				<th>Author</th>
				<th>Duration</th>
				<th>Date</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#if loading}
				<tr><td colspan="5">Loading simulator runs...</td></tr>
			{:else}
				{#each pagedRuns as row}
					<tr>
						<td>{row.title}</td>
						<td>{row.author}</td>
						<td>{row.duration_seconds}s</td>
						<td>{new Date(row.created_at).toLocaleString()}</td>
						<td>
							<div class="actions">
								<a class="admin-btn" href={`/admin/runs/${row.id}`}>View</a>
								<button class="admin-btn-danger" onclick={() => remove(row.id, row.title)}>Delete</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="5">No simulator runs found.</td></tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<Pagination bind:page total={runs.length} {pageSize} />
