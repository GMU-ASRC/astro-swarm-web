<script lang="ts">
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();
	let evaluations = $state(data.evaluations);
	let message = $state('');

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

<div>
	<h1 class="font-game text-2xl text-star-white mb-6">Evaluations</h1>
	{#if message}<div class="mb-4 text-sky-200 text-sm">{message}</div>{/if}

	<div class="overflow-x-auto border border-sky-500/20 bg-sky-500/5">
		<table class="w-full text-left border-collapse">
			<thead>
				<tr class="border-b border-sky-500/20 bg-sky-500/10 text-sky-400 font-game text-xs tracking-wider">
					<th class="p-4">USERNAME</th>
					<th class="p-4">LEVEL</th>
					<th class="p-4">STATUS</th>
					<th class="p-4">RATE</th>
					<th class="p-4">DATE</th>
					<th class="p-4 text-right">ACTIONS</th>
				</tr>
			</thead>
			<tbody>
				{#each evaluations as row}
					<tr class="border-b border-sky-500/10 hover:bg-sky-500/10 transition-colors">
						<td class="p-4 text-star-white">{row.username}</td>
						<td class="p-4 text-text-muted">{row.level_id || 'farp'}</td>
						<td class="p-4">
							<span class="text-[10px] font-game tracking-wider px-2 py-1 border 
								{row.status === 'done' ? 'text-green-300 border-green-400/40 bg-green-400/10' : 
								 row.status === 'failed' || row.status === 'error' ? 'text-red-300 border-red-400/40 bg-red-400/10' : 
								 'text-sky-300 border-sky-400/40 bg-sky-400/10'}">
								{row.status.toUpperCase()}
							</span>
						</td>
						<td class="p-4 text-text-muted">{row.success_rate != null ? `${row.success_rate}%` : '—'}</td>
						<td class="p-4 text-text-muted text-sm">{new Date(row.created_at).toLocaleString()}</td>
						<td class="p-4 text-right">
							<button onclick={() => remove(row.id, row.username)} class="px-3 py-1 border border-red-400/40 text-red-300 text-xs font-game tracking-wider hover:bg-red-500/15 transition-colors">
								DELETE
							</button>
						</td>
					</tr>
				{:else}
					<tr><td colspan="6" class="p-4 text-center text-text-muted">No evaluations found.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
