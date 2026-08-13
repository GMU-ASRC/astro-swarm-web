<script lang="ts">
	import Icon from '@iconify/svelte';
	import { apiUrl } from '$lib/ts/api';
	import { createPrompt } from '$lib/ts/prompt.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	let { data } = $props();
	let storage = $state<Record<string, any> | null>(null);
	let loading = $state(true);
	let reclaiming = $state('');
	let message = $state('');
	const prompt = createPrompt();

	$effect(() => {
		let active = true;
		loading = true;
		data.storagePromise.then((row: any) => {
			if (!active) return;
			storage = row;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	function formatBytes(bytes: number): string {
		if (bytes == null) return '—';
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
		return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
	}

	function share(bytes: number): string {
		const total = storage?.total_bytes ?? 0;
		if (!total) return '0%';
		return `${((100 * bytes) / total).toFixed(1)}%`;
	}

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : 'never';
	}

	async function refreshStorage() {
		try {
			const res = await fetch(apiUrl('/api/admin/storage'), {
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok) storage = await res.json();
		} catch {
			message = 'Could not refresh storage usage.';
		}
	}

	async function reclaim(table: string) {
		reclaiming = table;
		message = `Reclaiming space in ${table}...`;
		try {
			const res = await fetch(apiUrl('/api/admin/storage/reclaim'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json', 'X-API-Key': data.adminKey },
				body: JSON.stringify({ table })
			});
			if (res.ok) {
				const body = await res.json();
				message = `Reclaimed ${formatBytes(body.freed_bytes)} from ${table} (${formatBytes(body.before_bytes)} to ${formatBytes(body.after_bytes)}).`;
				await refreshStorage();
			} else {
				message = `Reclaim failed: ${res.status}`;
			}
		} catch (err) {
			message = `Reclaim failed: ${err}`;
		}
		reclaiming = '';
	}

	function askReclaim(table: string, deadRows: number) {
		prompt.ask({
			title: 'Reclaim disk space',
			message: `Rewrite ${table} to give its dead rows back to the disk? Re-simulating an entry leaves the old replay behind, and that is what this clears.`,
			warning: `The table is locked for the whole rewrite, so the site cannot read or write ${table} until it finishes.${deadRows ? ` ${deadRows} dead rows are waiting.` : ''}`,
			confirmLabel: 'Reclaim',
			run: () => reclaim(table)
		});
	}
</script>

<h1>Welcome, Admin</h1>
<p>Select a category to manage data, moderate content, and delete old evaluations or simulator runs.</p>

<ul>
	<li><a href="/admin/evaluations">Evaluations</a> — manage player level evaluations.</li>
	<li><a href="/admin/players">Players</a> — view player profiles and manage their entries.</li>
	<li><a href="/admin/leaderboard">Leaderboard</a> — success rate standings.</li>
	<li><a href="/admin/runs">Simulator Runs</a> — manage shared sandbox simulations.</li>
</ul>

<h2>Data Storage</h2>

{#if message}<div class="message">{message}</div>{/if}

{#if loading}
	<p>Loading storage usage...</p>
{:else if !storage}
	<p>Storage usage is unavailable.</p>
{:else}
	<div class="stat-grid">
		<div class="stat">
			<div class="label">Total Stored</div>
			<div>{formatBytes(storage.total_bytes)}</div>
		</div>
		<div class="stat">
			<div class="label">Database</div>
			<div>{formatBytes(storage.database_bytes)}</div>
		</div>
		<div class="stat">
			<div class="label">Uploads</div>
			<div>{formatBytes(storage.uploads_bytes)}</div>
		</div>
		<div class="stat">
			<div class="label">Upload Files</div>
			<div>{storage.upload_files}</div>
		</div>
	</div>

	{#if storage.database_error}
		<div class="message message-error">Database size unavailable: {storage.database_error}</div>
	{/if}
	{#if storage.uploads_error}
		<div class="message message-error">{storage.uploads_error}: {storage.upload_dir}</div>
	{/if}

	{#if storage.tables.length > 0}
		<div class="admin-table-wrap">
			<table>
				<thead>
					<tr>
						<th>Table</th>
						<th>Rows</th>
						<th>Size</th>
						<th>Share</th>
						<th>Dead Rows</th>
						<th>Last Vacuum</th>
						<th>Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each storage.tables as table}
						<tr>
							<td><code>{table.name}</code></td>
							<td>{table.rows}</td>
							<td>{formatBytes(table.bytes)}</td>
							<td>{share(table.bytes)}</td>
							<td>{table.dead_rows ?? 0}</td>
							<td>{when(table.last_vacuum)}</td>
							<td>
								<div class="actions">
									<button
										class="icon-btn"
										title="Reclaim disk space"
										aria-label="Reclaim disk space"
										disabled={reclaiming !== ''}
										onclick={() => askReclaim(table.name, table.dead_rows ?? 0)}
									>
										<Icon icon="ph:broom-bold" width="16" />
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}

{#if prompt.current}
	<ConfirmDialog
		title={prompt.current.title}
		message={prompt.current.message}
		warning={prompt.current.warning}
		confirmLabel={prompt.current.confirmLabel}
		danger={prompt.current.danger}
		onconfirm={prompt.accept}
		oncancel={prompt.dismiss}
	/>
{/if}
