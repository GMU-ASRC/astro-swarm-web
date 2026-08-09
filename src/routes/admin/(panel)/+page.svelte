<script lang="ts">
	let { data } = $props();
	let storage = $state<Record<string, any> | null>(null);
	let loading = $state(true);

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
					</tr>
				</thead>
				<tbody>
					{#each storage.tables as table}
						<tr>
							<td><code>{table.name}</code></td>
							<td>{table.rows}</td>
							<td>{formatBytes(table.bytes)}</td>
							<td>{share(table.bytes)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}
