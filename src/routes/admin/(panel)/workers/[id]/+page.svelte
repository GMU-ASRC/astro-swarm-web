<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();

	const workerId = $derived($page.params.id);
	const headers = { 'X-API-Key': data.adminKey };

	let worker = $state<any>(null);
	let loading = $state(true);
	let notFound = $state(false);
	let message = $state('');

	let name = $state('');
	let maxJobs = $state(1);
	let editingName = $state(false);
	let editingJobs = $state(false);

	async function refresh() {
		try {
			const res = await fetch(apiUrl(`/api/workers/${workerId}`), { headers });
			if (res.status === 404) {
				notFound = true;
				worker = null;
				return;
			}
			if (res.ok) {
				worker = await res.json();
				if (!editingName) name = worker.name;
				if (!editingJobs) maxJobs = worker.max_jobs;
			}
		} catch (err) {
			// keep last known state on transient errors
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		refresh();
		const timer = setInterval(refresh, 3000);
		return () => clearInterval(timer);
	});

	async function saveSettings() {
		try {
			const res = await fetch(apiUrl(`/api/workers/${workerId}/settings`), {
				method: 'POST',
				headers: { ...headers, 'Content-Type': 'application/json' },
				body: JSON.stringify({ name, max_jobs: maxJobs })
			});
			if (res.ok) {
				worker = await res.json();
				editingName = false;
				editingJobs = false;
				message = 'Saved worker settings.';
			} else {
				message = `Failed to save: ${res.status}`;
			}
		} catch (err) {
			message = `Save failed: ${err}`;
		}
	}

	async function setConnected(connected: boolean) {
		const action = connected ? 'connect' : 'disconnect';
		try {
			const res = await fetch(apiUrl(`/api/workers/${workerId}/${action}`), { method: 'POST', headers });
			if (res.ok) {
				worker = await res.json();
				message = connected ? 'Worker connected.' : 'Worker disconnected; its job was requeued.';
			} else {
				message = `Failed to ${action}: ${res.status}`;
			}
		} catch (err) {
			message = `${action} failed: ${err}`;
		}
	}

	async function remove() {
		if (!confirm(`Remove worker "${worker.name}"? Any job it is running will be requeued.`)) return;
		try {
			const res = await fetch(apiUrl(`/api/workers/${workerId}`), { method: 'DELETE', headers });
			if (res.ok || res.status === 204) {
				await goto('/admin/workers');
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

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : '—';
	}

	const hasStats = $derived(worker?.system_stats && Object.keys(worker.system_stats).length > 0);
</script>

<p><a href="/admin/workers">← All workers</a></p>

{#if message}<div class="message">{message}</div>{/if}

{#if loading && !worker}
	<p>Loading worker...</p>
{:else if notFound || !worker}
	<p>Worker not found.</p>
{:else}
	<h1>{worker.name}</h1>
	<p class="meta">
		<span class="pill" style={statusStyle(worker.status)}>{worker.status}</span>
		· {worker.hostname || 'unknown host'}
	</p>
	<p class="meta">{worker.id}</p>

	<div class="actions">
		{#if worker.enabled}
			<button onclick={() => setConnected(false)}>Disconnect</button>
		{:else}
			<button onclick={() => setConnected(true)}>Connect</button>
		{/if}
		<button class="admin-btn-danger" onclick={remove}>Remove</button>
	</div>

	<h2>Settings</h2>
	<div class="stat-grid">
		<div class="stat">
			<div class="label">Name</div>
			<input
				type="text"
				bind:value={name}
				oninput={() => (editingName = true)}
				style="width:100%"
			/>
		</div>
		<div class="stat">
			<div class="label">Max parallel jobs</div>
			<input
				type="number"
				min="1"
				max="64"
				bind:value={maxJobs}
				oninput={() => (editingJobs = true)}
				style="width:5rem"
			/>
		</div>
	</div>
	<p class="meta">Max parallel jobs is how many Godot processes this worker runs at once for a single evaluation. Higher values use more CPU and memory on the worker machine.</p>
	<div class="actions">
		<button onclick={saveSettings} disabled={!editingName && !editingJobs}>Save settings</button>
	</div>

	<h2>Status</h2>
	<div class="admin-table-wrap">
		<table>
			<tbody>
				<tr><th>Connection</th><td>{worker.enabled ? 'Enabled' : 'Disconnected by admin'}</td></tr>
				<tr><th>Online</th><td>{worker.online ? 'Yes' : 'No'}</td></tr>
				<tr><th>Last seen</th><td>{when(worker.last_seen)}</td></tr>
				<tr><th>First seen</th><td>{when(worker.created_at)}</td></tr>
			</tbody>
		</table>
	</div>

	<h2>System</h2>
	{#if hasStats}
		<div class="stat-grid">
			<div class="stat">
				<div class="label">CPU</div>
				<div class="value">{worker.system_stats.cpu_percent ?? '—'}%{#if worker.system_stats.cpu_count} · {worker.system_stats.cpu_count} cores{/if}</div>
			</div>
			<div class="stat">
				<div class="label">Memory</div>
				<div class="value">{worker.system_stats.memory_percent ?? '—'}%{#if worker.system_stats.memory_total_mb} · {worker.system_stats.memory_used_mb} / {worker.system_stats.memory_total_mb} MB{/if}</div>
			</div>
			<div class="stat">
				<div class="label">Disk</div>
				<div class="value">{worker.system_stats.disk_percent ?? '—'}%{#if worker.system_stats.disk_total_gb} · {worker.system_stats.disk_used_gb} / {worker.system_stats.disk_total_gb} GB{/if}</div>
			</div>
			<div class="stat">
				<div class="label">Load (1m)</div>
				<div class="value">{worker.system_stats.load_avg_1m ?? '—'}</div>
			</div>
		</div>
	{:else}
		<p class="meta">No system stats reported yet. The worker needs psutil installed and a recent heartbeat.</p>
	{/if}

	<h2>Active jobs ({worker.jobs?.length ?? 0})</h2>
	{#if worker.jobs && worker.jobs.length > 0}
		<div class="admin-table-wrap">
			<table>
				<thead>
					<tr><th>Evaluation</th><th>Player</th><th>Level</th><th>Shard</th><th>Progress</th><th>Updated</th></tr>
				</thead>
				<tbody>
					{#each worker.jobs as job}
						<tr>
							<td><a href={`/admin/evaluations/${job.evaluation_id}`}>{job.evaluation_id}</a></td>
							<td>{job.username ?? '—'}</td>
							<td>{job.level_id ?? '—'}</td>
							<td>#{job.shard_index}</td>
							<td>{job.done_units} / {job.total_units}</td>
							<td>{when(job.last_update)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="meta">This worker is not running any jobs right now.</p>
	{/if}
{/if}
