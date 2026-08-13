<script lang="ts">
	import Icon from '@iconify/svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { apiUrl } from '$lib/ts/api';
	import { runBulk, bulkMessage } from '$lib/ts/bulk';
	import { createPrompt } from '$lib/ts/prompt.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import BulkActionBar from '$lib/components/BulkActionBar.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	const MANY_ENTRIES = 10;

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

	async function refresh() {
		try {
			const res = await fetch(apiUrl('/api/evaluations'));
			if (res.ok) evaluations = await res.json();
		} catch {
			// keep last known data on transient errors
		}
	}

	const hasActive = $derived(
		evaluations.some((e: any) => e.status === 'running' || e.status === 'queued')
	);

	// Poll for live progress while any evaluation is queued/running.
	$effect(() => {
		if (!hasActive) return;
		const timer = setInterval(refresh, 3000);
		return () => clearInterval(timer);
	});

	function pct(value: number | null | undefined): number {
		return Math.max(0, Math.min(100, Math.round((value ?? 0) * 100)));
	}

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : '—';
	}

	let searchQuery = $state('');
	let statusFilter = $state('all');
	let levelFilter = $state('all');
	let sortOrder = $state('date_desc');
	let startDate = $state('');
	let endDate = $state('');

	const levelOptions = $derived(
		Array.from(new Set(evaluations.map((e: any) => e.level_id || 'farp'))).sort()
	);

	const filtered = $derived(
		evaluations
			.filter((e: any) => {
				if (searchQuery.trim() !== '') {
					const q = searchQuery.toLowerCase();
					const user = (e.username ?? '').toLowerCase();
					const id = (e.id ?? '').toLowerCase();
					if (!user.includes(q) && !id.includes(q)) return false;
				}
				if (statusFilter !== 'all' && e.status !== statusFilter) return false;
				if (levelFilter !== 'all' && (e.level_id || 'farp') !== levelFilter) return false;
				if (startDate) {
					const d = new Date(e.created_at);
					const s = new Date(startDate);
					s.setHours(0, 0, 0, 0);
					if (d.getTime() < s.getTime()) return false;
				}
				if (endDate) {
					const d = new Date(e.created_at);
					const en = new Date(endDate);
					en.setHours(23, 59, 59, 999);
					if (d.getTime() > en.getTime()) return false;
				}
				return true;
			})
			.sort((a: any, b: any) => {
				if (sortOrder === 'date_desc')
					return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
				if (sortOrder === 'date_asc')
					return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
				if (sortOrder === 'rate_desc') return (b.success_rate ?? -1) - (a.success_rate ?? -1);
				if (sortOrder === 'rate_asc') return (a.success_rate ?? -1) - (b.success_rate ?? -1);
				return 0;
			})
	);

	const selected = new SvelteSet<string>();
	const selectedRows = $derived(filtered.filter((e: any) => selected.has(e.id)));
	const allFilteredSelected = $derived(
		filtered.length > 0 && filtered.every((e: any) => selected.has(e.id))
	);
	const resimulatableRows = $derived(
		selectedRows.filter((e: any) => e.status !== 'queued' && e.status !== 'running')
	);

	function toggleRow(id: string) {
		if (selected.has(id)) selected.delete(id);
		else selected.add(id);
	}

	function toggleAll() {
		if (allFilteredSelected) selected.clear();
		else for (const row of filtered) selected.add(row.id);
	}

	let page = $state(1);
	const pageSize = 25;
	const pagedEvaluations = $derived(filtered.slice((page - 1) * pageSize, page * pageSize));

	$effect(() => {
		searchQuery;
		statusFilter;
		levelFilter;
		sortOrder;
		startDate;
		endDate;
		page = 1;
		selected.clear();
	});

	const prompt = createPrompt();

	function manyWarning(count: number, action: string): string {
		return count >= MANY_ENTRIES ? `That is a lot of entries to ${action} at once.` : '';
	}

	async function removeSelected() {
		const ids = selectedRows.map((row: any) => row.id);
		message = `Deleting ${ids.length} evaluations...`;
		const result = await runBulk(ids, data.adminKey, 'DELETE', (id) => `/api/evaluations/${id}`);
		evaluations = evaluations.filter((e: any) => !result.succeeded.includes(e.id));
		selected.clear();
		message = bulkMessage('Deleted', 'evaluations', result);
	}

	async function resimulateSelected() {
		const ids = resimulatableRows.map((row: any) => row.id);
		message = `Queueing ${ids.length} evaluations...`;
		const result = await runBulk(
			ids,
			data.adminKey,
			'POST',
			(id) => `/api/evaluations/${id}/resimulate`
		);
		evaluations = evaluations.map((e: any) =>
			result.succeeded.includes(e.id) ? { ...e, status: 'queued', progress: 0 } : e
		);
		selected.clear();
		message = bulkMessage('Queued', 'evaluations', result);
	}

	function askRemoveSelected() {
		const count = selectedRows.length;
		prompt.ask({
			title: 'Delete evaluations',
			message: `Delete ${count} selected ${count === 1 ? 'evaluation' : 'evaluations'}? Their results and replays cannot be recovered.`,
			warning: manyWarning(count, 'delete'),
			confirmLabel: 'Delete',
			danger: true,
			run: removeSelected
		});
	}

	function askResimulateSelected() {
		const count = resimulatableRows.length;
		const skipped = selectedRows.length - count;
		prompt.ask({
			title: 'Re-simulate evaluations',
			message: `Re-run ${count} selected ${count === 1 ? 'evaluation' : 'evaluations'} with the current simulator build? This overwrites their results and replays.`,
			warning:
				skipped > 0
					? `${skipped} selected ${skipped === 1 ? 'entry is' : 'entries are'} already queued or running and will be skipped.`
					: manyWarning(count, 're-simulate'),
			confirmLabel: 'Re-simulate',
			run: resimulateSelected
		});
	}

	function askCancel(id: string, name: string) {
		prompt.ask({
			title: 'Cancel run',
			message: `Cancel the running evaluation for "${name}"?`,
			confirmLabel: 'Cancel run',
			danger: true,
			run: () => cancel(id, name)
		});
	}

	function askRemove(id: string, name: string) {
		prompt.ask({
			title: 'Delete evaluation',
			message: `Delete the evaluation for "${name}"? Its results and replays cannot be recovered.`,
			confirmLabel: 'Delete',
			danger: true,
			run: () => remove(id, name)
		});
	}

	async function cancel(id: string, name: string) {
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${id}/cancel`), {
				method: 'POST',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 202) {
				evaluations = evaluations.map((e: any) => (e.id === id ? { ...e, status: 'cancelled' } : e));
				message = `Cancelled evaluation for ${name}.`;
			} else {
				message = `Failed to cancel: ${res.status}`;
			}
		} catch (err) {
			message = `Cancel failed: ${err}`;
		}
	}

	async function remove(id: string, name: string) {
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

<div class="admin-filters">
	<div class="filter-field grow">
		<label for="ev-search">Search</label>
		<input id="ev-search" type="text" placeholder="Username or ID..." bind:value={searchQuery} />
	</div>
	<div class="filter-field">
		<label for="ev-status">Status</label>
		<select id="ev-status" bind:value={statusFilter}>
			<option value="all">All</option>
			<option value="queued">Queued</option>
			<option value="running">Running</option>
			<option value="done">Done</option>
			<option value="failed">Failed</option>
			<option value="cancelled">Cancelled</option>
		</select>
	</div>
	<div class="filter-field">
		<label for="ev-level">Level</label>
		<select id="ev-level" bind:value={levelFilter}>
			<option value="all">All</option>
			{#each levelOptions as lvl}
				<option value={lvl}>{lvl}</option>
			{/each}
		</select>
	</div>
	<div class="filter-field">
		<label for="ev-start">From</label>
		<input id="ev-start" type="date" bind:value={startDate} />
	</div>
	<div class="filter-field">
		<label for="ev-end">To</label>
		<input id="ev-end" type="date" bind:value={endDate} />
	</div>
	<div class="filter-field">
		<label for="ev-sort">Sort by</label>
		<select id="ev-sort" bind:value={sortOrder}>
			<option value="date_desc">Date (Newest)</option>
			<option value="date_asc">Date (Oldest)</option>
			<option value="rate_desc">Rate (High to Low)</option>
			<option value="rate_asc">Rate (Low to High)</option>
		</select>
	</div>
</div>

{#if !loading}
	<p class="filter-summary">Showing {filtered.length} of {evaluations.length} evaluations</p>
{/if}

<BulkActionBar
	count={selectedRows.length}
	noun="evaluation"
	nounPlural="evaluations"
	onclear={() => selected.clear()}
>
	<button
		type="button"
		onclick={askResimulateSelected}
		disabled={resimulatableRows.length === 0}
	>
		<Icon icon="ph:arrows-clockwise-bold" width="14" />
		Re-simulate
	</button>
	<button type="button" class="admin-btn-danger" onclick={askRemoveSelected}>
		<Icon icon="ph:trash-bold" width="14" />
		Delete
	</button>
</BulkActionBar>

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th class="select-cell">
					<input
						type="checkbox"
						aria-label="Select all evaluations"
						checked={allFilteredSelected}
						indeterminate={selectedRows.length > 0 && !allFilteredSelected}
						onchange={toggleAll}
					/>
				</th>
				<th>Username</th>
				<th>ID</th>
				<th>Level</th>
				<th>Defenders</th>
				<th>Version</th>
				<th>Status</th>
				<th>Progress</th>
				<th>Trials</th>
				<th>Rate</th>
				<th>Created</th>
				<th>Completed</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#if loading}
				<tr><td colspan="13">Loading evaluations...</td></tr>
			{:else}
				{#each pagedEvaluations as row}
					<tr class:selected={selected.has(row.id)}>
						<td class="select-cell">
							<input
								type="checkbox"
								aria-label={`Select evaluation for ${row.username}`}
								checked={selected.has(row.id)}
								onchange={() => toggleRow(row.id)}
							/>
						</td>
						<td>{row.username}</td>
						<td><code class="eval-id" title={row.id}>{row.id.slice(0, 8)}</code></td>
						<td>{row.level_id || 'farp'}</td>
						<td>{row.defender_count ?? '—'}</td>
						<td>{row.game_version ?? 'v0.0.4'}</td>
						<td><span class="pill status-{row.status}">{row.status.toUpperCase()}</span></td>
						<td class="progress-cell">
							{#if row.status === 'running' || row.status === 'queued'}
								<div class="progress-track" title={`${pct(row.progress)}%`}>
									<div class="progress-fill" style={`width:${pct(row.progress)}%`}></div>
								</div>
								<div class="meta">{pct(row.progress)}%{#if row.stage} · {row.stage}{/if}</div>
							{:else if row.status === 'done'}
								<div class="progress-track"><div class="progress-fill done" style="width:100%"></div></div>
							{:else}
								—
							{/if}
						</td>
						<td>{row.trials ?? '—'}</td>
						<td>{row.success_rate != null ? `${row.success_rate}%` : '—'}</td>
						<td>{when(row.created_at)}</td>
						<td>{when(row.completed_at)}</td>
						<td>
							<div class="actions">
								<a
									class="admin-btn icon-btn"
									href={`/admin/evaluations/${row.id}`}
									title="View"
									aria-label="View"
								>
									<Icon icon="ph:eye-bold" width="16" />
								</a>
								{#if row.status === 'queued' || row.status === 'running'}
									<button
										class="icon-btn"
										title="Cancel run"
										aria-label="Cancel run"
										onclick={() => askCancel(row.id, row.username)}
									>
										<Icon icon="ph:prohibit-bold" width="16" />
									</button>
								{/if}
								<button
									class="admin-btn-danger icon-btn"
									title="Delete"
									aria-label="Delete"
									onclick={() => askRemove(row.id, row.username)}
								>
									<Icon icon="ph:trash-bold" width="16" />
								</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="13">No evaluations found.</td></tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<Pagination bind:page total={filtered.length} {pageSize} />

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
	tbody tr.selected td {
		background: color-mix(in srgb, var(--color-brand) 10%, transparent);
	}
	.eval-id {
		font-size: 0.78em;
		color: var(--color-faint);
	}
	.progress-cell {
		min-width: 12rem;
	}
	.progress-track {
		height: 8px;
		width: 100%;
		background: var(--color-ink);
		border: 1px solid var(--color-line);
		overflow: hidden;
	}
	.progress-fill {
		height: 100%;
		background: var(--color-brand);
		transition: width 0.4s ease;
	}
	.progress-fill.done {
		background: var(--color-win);
	}
</style>
