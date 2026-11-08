<script lang="ts">
	import Icon from '@iconify/svelte';
	import { apiUrl } from '$lib/ts/api';
	import { createPrompt } from '$lib/ts/prompt.svelte';
	import { formatBytes, formatCount, formatWhen } from '$lib/ts/adminFormat';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	let { storage = $bindable(), adminKey }: { storage: Record<string, any> | null; adminKey: string } = $props();

	let reclaiming = $state('');
	let message = $state('');
	const prompt = createPrompt();

	let databaseBytes = $derived(storage?.database_bytes ?? 0);
	let uploadShare = $derived(storage?.total_bytes ? (100 * storage.uploads_bytes) / storage.total_bytes : 0);

	function share(bytes: number): number {
		return databaseBytes ? (100 * bytes) / databaseBytes : 0;
	}

	async function refreshStorage(measure = false) {
		try {
			const res = await fetch(apiUrl(measure ? '/api/admin/storage?measure=1' : '/api/admin/storage'), {
				headers: { 'X-API-Key': adminKey }
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
				headers: { 'Content-Type': 'application/json', 'X-API-Key': adminKey },
				body: JSON.stringify({ table })
			});
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				message = `Reclaimed ${formatBytes(body.freed_bytes)} from ${table} (${formatBytes(body.before_bytes)} to ${formatBytes(body.after_bytes)}).`;
				await refreshStorage();
			} else {
				message = `Reclaim failed: ${body.error ?? res.status}`;
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

{#if message}<div class="message">{message}</div>{/if}

{#if storage}

<div class="split card-row">
	<div class="split-bar" role="img" aria-label="Database and uploads share of storage">
		<div class="split-database" style:width={`${100 - uploadShare}%`}></div>
		<div class="split-uploads" style:width={`${uploadShare}%`}></div>
	</div>
	<div class="split-legend">
		<span><span class="swatch database"></span>Database <strong>{formatBytes(storage.database_bytes)}</strong></span>
		<span><span class="swatch uploads"></span>Uploads <strong>{formatBytes(storage.uploads_bytes)}</strong> in {formatCount(storage.upload_files)} files</span>
		<span class="measured">
			Uploads measured {formatWhen(storage.uploads_measured_at)}{storage.uploads_measuring ? ', measuring again now' : ''}
			<button class="admin-btn" onclick={() => refreshStorage(true)}>Measure now</button>
		</span>
	</div>
</div>

{#if storage.database_error}
	<div class="message message-error">Database size unavailable: {storage.database_error}</div>
{/if}
{#if storage.uploads_error}
	<div class="message message-error">{storage.uploads_error}: {storage.upload_dir}</div>
{/if}

{#if storage.tables?.length > 0}
	<div class="admin-table-wrap">
		<table>
			<thead>
				<tr>
					<th>Table</th>
					<th>Rows</th>
					<th>Size</th>
					<th>Share of database</th>
					<th>Dead rows</th>
					<th>Last vacuum</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each storage.tables as table}
					<tr>
						<td><code>{table.name}</code></td>
						<td>{formatCount(table.rows)}</td>
						<td>{formatBytes(table.bytes)}</td>
						<td>
							<div class="share-cell">
								<span class="share-track"><span class="share-fill" style:width={`${share(table.bytes)}%`}></span></span>
								<span>{share(table.bytes).toFixed(1)}%</span>
							</div>
						</td>
						<td>{formatCount(table.dead_rows ?? 0)}</td>
						<td>{formatWhen(table.last_vacuum)}</td>
						<td>
							<div class="actions">
								<button
									class="icon-btn"
									title={table.reclaimable === false ? 'Too large to rewrite while the site is running' : 'Reclaim disk space'}
									aria-label="Reclaim disk space"
									disabled={reclaiming !== '' || table.reclaimable === false}
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

<style>
	.card-row {
		margin-bottom: 1rem;
		padding: 1rem 1.1rem;
		border: 1px solid var(--color-line);
		background: var(--color-surface);
	}

	.split-bar {
		display: flex;
		height: 0.8rem;
		overflow: hidden;
		border-radius: 999px;
		background: var(--color-surface-raised);
	}

	.split-database,
	.swatch.database {
		background: var(--color-brand);
	}

	.split-uploads,
	.swatch.uploads {
		background: var(--color-gold);
	}

	.split-legend {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.6rem 1.5rem;
		margin-top: 0.85rem;
		color: var(--color-dim);
		font-size: 0.8rem;
	}

	.split-legend strong {
		color: var(--color-heading);
	}

	.swatch {
		display: inline-block;
		width: 0.6rem;
		height: 0.6rem;
		margin-right: 0.4rem;
		border-radius: 2px;
	}

	.measured {
		display: inline-flex;
		align-items: center;
		gap: 0.6rem;
		margin-left: auto;
	}

	.share-cell {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		min-width: 9rem;
		font-variant-numeric: tabular-nums;
	}

	.share-track {
		flex: 1;
		height: 4px;
		border-radius: 2px;
		background: var(--color-surface-raised);
	}

	.share-fill {
		display: block;
		height: 100%;
		border-radius: 2px;
		background: var(--color-brand);
	}
</style>
