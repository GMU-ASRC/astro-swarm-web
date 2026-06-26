<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();

	let workers = $state<any[]>([]);
	let loading = $state(true);

	const headers = { 'X-API-Key': data.adminKey };

	async function refresh() {
		try {
			const res = await fetch(apiUrl('/api/workers'), { headers });
			if (res.ok) {
				workers = await res.json();
			}
		} catch (err) {
			// transient network error; keep showing the last known list
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		refresh();
		const timer = setInterval(refresh, 3000);
		return () => clearInterval(timer);
	});

	function statusStyle(status: string): string {
		if (status === 'idle') return 'color:#15803d;border-color:#86c9a0;background:#e7f6ee';
		if (status === 'busy') return 'color:#1d4ed8;border-color:#9cb6ef;background:#e8eefb';
		if (status === 'disconnected') return 'color:#b45309;border-color:#e3c08a;background:#fbf1df';
		return 'color:#6b7280;border-color:#d4d4d8;background:#f4f4f5';
	}

	function lastSeen(iso: string | null): string {
		if (!iso) return '—';
		return new Date(iso).toLocaleString();
	}
</script>

<h1>Workers</h1>
<p class="meta">
	Worker nodes pull queued evaluation jobs and run the Godot dedicated server. Workers auto-connect on
	startup. Click a worker to view it and edit its settings.
</p>

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Name</th>
				<th>Host</th>
				<th>Status</th>
				<th>Max jobs</th>
				<th>Current job</th>
				<th>Last seen</th>
			</tr>
		</thead>
		<tbody>
			{#if loading}
				<tr><td colspan="6">Loading workers...</td></tr>
			{:else}
				{#each workers as worker}
					<tr
						class="row-link"
						onclick={() => goto(`/admin/workers/${worker.id}`)}
						title="View worker"
					>
						<td>{worker.name}</td>
						<td>{worker.hostname || '—'}</td>
						<td><span class="pill" style={statusStyle(worker.status)}>{worker.status}</span></td>
						<td>{worker.max_jobs}</td>
						<td>{worker.current_job_id ? `${worker.current_job_id.slice(0, 8)}…` : '—'}</td>
						<td>{lastSeen(worker.last_seen)}</td>
					</tr>
				{:else}
					<tr><td colspan="6">No workers connected. Start a worker pointed at this server.</td></tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<style>
	.row-link {
		cursor: pointer;
	}
</style>
