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

<div>
	<h1 class="font-game text-2xl text-star-white mb-6">Simulator Runs</h1>
	{#if message}<div class="mb-4 text-sky-200 text-sm">{message}</div>{/if}

	<div class="overflow-x-auto border border-sky-500/20 bg-sky-500/5">
		<table class="w-full text-left border-collapse">
			<thead>
				<tr class="border-b border-sky-500/20 bg-sky-500/10 text-sky-400 font-game text-xs tracking-wider">
					<th class="p-4">TITLE</th>
					<th class="p-4">AUTHOR</th>
					<th class="p-4">DURATION</th>
					<th class="p-4">DATE</th>
					<th class="p-4 text-right">ACTIONS</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr><td colspan="5" class="p-4 text-center text-sky-200 animate-pulse">Loading simulator runs...</td></tr>
				{:else}
					{#each pagedRuns as row}
						<tr class="border-b border-sky-500/10 hover:bg-sky-500/10 transition-colors">
							<td class="p-4 text-star-white">{row.title}</td>
							<td class="p-4 text-text-muted">{row.author}</td>
							<td class="p-4 text-text-muted">{row.duration_seconds}s</td>
							<td class="p-4 text-text-muted text-sm">{new Date(row.created_at).toLocaleString()}</td>
							<td class="p-4 text-right whitespace-nowrap">
								<a href={`/admin/runs/${row.id}`} class="px-3 py-1 border border-sky-400/40 text-sky-200 text-xs font-game tracking-wider hover:bg-sky-500/15 transition-colors">
									VIEW
								</a>
								<button onclick={() => remove(row.id, row.title)} class="ml-2 px-3 py-1 border border-red-400/40 text-red-300 text-xs font-game tracking-wider hover:bg-red-500/15 transition-colors">
									DELETE
								</button>
							</td>
						</tr>
					{:else}
						<tr><td colspan="5" class="p-4 text-center text-text-muted">No simulator runs found.</td></tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>

	<Pagination bind:page total={runs.length} {pageSize} />
</div>
