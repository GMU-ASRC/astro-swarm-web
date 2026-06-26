<script lang="ts">
	import { onDestroy } from 'svelte';
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();

	let workers = $state<any[]>([]);
	let loading = $state(true);
	let message = $state('');
	let edits = $state<Record<string, number>>({});

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

	async function setConnected(id: string, name: string, connected: boolean) {
		const action = connected ? 'connect' : 'disconnect';
		try {
			const res = await fetch(apiUrl(`/api/workers/${id}/${action}`), { method: 'POST', headers });
			if (res.ok) {
				message = `${connected ? 'Connected' : 'Disconnected'} ${name}.`;
				await refresh();
			} else {
				message = `Failed to ${action}: ${res.status}`;
			}
		} catch (err) {
			message = `${action} failed: ${err}`;
		}
	}

	async function saveMaxJobs(id: string, name: string) {
		const value = edits[id];
		if (value == null || value < 1) return;
		try {
			const res = await fetch(apiUrl(`/api/workers/${id}/max-jobs`), {
				method: 'POST',
				headers: { ...headers, 'Content-Type': 'application/json' },
				body: JSON.stringify({ max_jobs: value })
			});
			if (res.ok) {
				delete edits[id];
				message = `Set ${name} max parallel jobs to ${value}.`;
				await refresh();
			} else {
				message = `Failed to set max jobs: ${res.status}`;
			}
		} catch (err) {
			message = `Set max jobs failed: ${err}`;
		}
	}

	async function remove(id: string, name: string) {
		if (!confirm(`Remove worker "${name}"? Any job it is running will be requeued.`)) return;
		try {
			const res = await fetch(apiUrl(`/api/workers/${id}`), { method: 'DELETE', headers });
			if (res.ok || res.status === 204) {
				workers = workers.filter((w) => w.id !== id);
				message = `Removed ${name}.`;
			} else {
				message = `Failed to remove: ${res.status}`;
			}
		} catch (err) {
			message = `Remove failed: ${err}`;
		}
	}

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
	startup; use Disconnect to stop a worker from taking new jobs (its current job is requeued).
</p>
{#if message}<div class="message">{message}</div>{/if}

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
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#if loading}
				<tr><td colspan="7">Loading workers...</td></tr>
			{:else}
				{#each workers as worker}
					<tr>
						<td>{worker.name}</td>
						<td>{worker.hostname || '—'}</td>
						<td><span class="pill" style={statusStyle(worker.status)}>{worker.status}</span></td>
						<td>
							<div class="actions">
								<input
									type="number"
									min="1"
									max="64"
									value={edits[worker.id] ?? worker.max_jobs}
									oninput={(e) => (edits[worker.id] = Number(e.currentTarget.value))}
									style="width:4.5rem"
								/>
								{#if edits[worker.id] != null && edits[worker.id] !== worker.max_jobs}
									<button onclick={() => saveMaxJobs(worker.id, worker.name)}>Save</button>
								{/if}
							</div>
						</td>
						<td>
							{#if worker.current_job_id}
								<a href={`/admin/evaluations/${worker.current_job_id}`}>{worker.current_job_id.slice(0, 8)}…</a>
							{:else}
								—
							{/if}
						</td>
						<td>{lastSeen(worker.last_seen)}</td>
						<td>
							<div class="actions">
								{#if worker.enabled}
									<button onclick={() => setConnected(worker.id, worker.name, false)}>Disconnect</button>
								{:else}
									<button onclick={() => setConnected(worker.id, worker.name, true)}>Connect</button>
								{/if}
								<button class="admin-btn-danger" onclick={() => remove(worker.id, worker.name)}>Remove</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="7">No workers connected. Start a worker pointed at this server.</td></tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>
