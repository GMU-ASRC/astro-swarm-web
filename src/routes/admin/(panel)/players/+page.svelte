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
	let players = $state<any[]>([]);
	let loading = $state(true);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.playersPromise.then((rows: any[]) => {
			if (!active) return;
			players = rows;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	let searchQuery = $state('');
	let sortOrder = $state('rating_desc');

	const filtered = $derived(
		players
			.filter((row: any) => {
				if (searchQuery.trim() === '') return true;
				const query = searchQuery.toLowerCase();
				const name = (row.username ?? '').toLowerCase();
				const id = (row.player_id ?? '').toLowerCase();
				return name.includes(query) || id.includes(query);
			})
			.sort((a: any, b: any) => {
				if (sortOrder === 'rating_desc') return (b.rating ?? -1) - (a.rating ?? -1);
				if (sortOrder === 'rating_asc') return (a.rating ?? -1) - (b.rating ?? -1);
				if (sortOrder === 'rate_desc') return (b.overall_success ?? -1) - (a.overall_success ?? -1);
				if (sortOrder === 'rate_asc') return (a.overall_success ?? -1) - (b.overall_success ?? -1);
				if (sortOrder === 'entries_desc') return b.entries - a.entries;
				if (sortOrder === 'name') return (a.username ?? '').localeCompare(b.username ?? '');
				return 0;
			})
	);

	const selected = new SvelteSet<string>();
	const selectedRows = $derived(filtered.filter((row: any) => selected.has(row.player_id)));
	const allFilteredSelected = $derived(
		filtered.length > 0 && filtered.every((row: any) => selected.has(row.player_id))
	);
	const selectedEntries = $derived(
		selectedRows.reduce((total: number, row: any) => total + (row.entries ?? 0), 0)
	);

	function toggleRow(playerId: string) {
		if (selected.has(playerId)) selected.delete(playerId);
		else selected.add(playerId);
	}

	function toggleAll() {
		if (allFilteredSelected) selected.clear();
		else for (const row of filtered) selected.add(row.player_id);
	}

	let page = $state(1);
	const pageSize = 25;
	const pagedPlayers = $derived(filtered.slice((page - 1) * pageSize, page * pageSize));

	$effect(() => {
		searchQuery;
		sortOrder;
		page = 1;
		selected.clear();
	});

	const prompt = createPrompt();

	function rate(value: number | null): string {
		return value != null ? `${value}%` : '—';
	}

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : '—';
	}

	function askRemovePlayer(playerId: string, name: string, entries: number) {
		prompt.ask({
			title: 'Delete player',
			message: `Delete "${name}" and all ${entries} of their ${entries === 1 ? 'entry' : 'entries'}? This cannot be undone.`,
			warning:
				entries >= MANY_ENTRIES ? `That deletes ${entries} entries in one go.` : '',
			confirmLabel: 'Delete',
			danger: true,
			run: () => removePlayer(playerId, name)
		});
	}

	function askRemoveSelected() {
		const count = selectedRows.length;
		prompt.ask({
			title: 'Delete players',
			message: `Delete ${count} selected ${count === 1 ? 'player' : 'players'} and all ${selectedEntries} of their ${selectedEntries === 1 ? 'entry' : 'entries'}? This cannot be undone.`,
			warning:
				count >= MANY_ENTRIES || selectedEntries >= MANY_ENTRIES
					? `That removes ${count} ${count === 1 ? 'player' : 'players'} and ${selectedEntries} ${selectedEntries === 1 ? 'entry' : 'entries'} at once.`
					: '',
			confirmLabel: 'Delete',
			danger: true,
			run: removeSelected
		});
	}

	async function removeSelected() {
		const ids = selectedRows.map((row: any) => row.player_id);
		message = `Deleting ${ids.length} players...`;
		const result = await runBulk(
			ids,
			data.adminKey,
			'DELETE',
			(id) => `/api/evaluations/players/${id}`
		);
		players = players.filter((p: any) => !result.succeeded.includes(p.player_id));
		selected.clear();
		message = bulkMessage('Deleted', 'players', result);
	}

	async function removePlayer(playerId: string, name: string) {
		try {
			const res = await fetch(apiUrl(`/api/evaluations/players/${playerId}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok) {
				players = players.filter((p: any) => p.player_id !== playerId);
				message = `Deleted ${name} and all their entries.`;
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<h1>Players</h1>
<p>View a commander's profile and manage the entries they have submitted.</p>

{#if message}<div class="message">{message}</div>{/if}

<div class="admin-filters">
	<div class="filter-field grow">
		<label for="pl-search">Search</label>
		<input id="pl-search" type="text" placeholder="Username or player ID..." bind:value={searchQuery} />
	</div>
	<div class="filter-field">
		<label for="pl-sort">Sort by</label>
		<select id="pl-sort" bind:value={sortOrder}>
			<option value="rate_desc">Rate (High to Low)</option>
			<option value="rate_asc">Rate (Low to High)</option>
			<option value="entries_desc">Entries (Most)</option>
			<option value="name">Username (A-Z)</option>
		</select>
	</div>
</div>

{#if !loading}
	<p class="filter-summary">Showing {filtered.length} of {players.length} players</p>
{/if}

<BulkActionBar
	count={selectedRows.length}
	noun="player"
	nounPlural="players"
	onclear={() => selected.clear()}
>
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
						aria-label="Select all players"
						checked={allFilteredSelected}
						indeterminate={selectedRows.length > 0 && !allFilteredSelected}
						onchange={toggleAll}
					/>
				</th>
				<th>Username</th>
				<th>Player ID</th>
				<th>Rating</th>
				<th>Levels</th>
				<th>Average Rate</th>
				<th>Best Rate</th>
				<th>Entries</th>
				<th>XP</th>
				<th>Last Active</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#if loading}
				<tr><td colspan="11">Loading players...</td></tr>
			{:else}
				{#each pagedPlayers as row}
					<tr class:selected={selected.has(row.player_id)}>
						<td class="select-cell">
							<input
								type="checkbox"
								aria-label={`Select ${row.username}`}
								checked={selected.has(row.player_id)}
								onchange={() => toggleRow(row.player_id)}
							/>
						</td>
						<td>{row.username}</td>
						<td><code title={row.player_id}>{row.player_id.slice(0, 8)}</code></td>
						<td>{row.rating ?? '—'}</td>
						<td>{row.levels_played}/{row.levels_total}</td>
						<td>{rate(row.overall_success)}</td>
						<td>{rate(row.best_success)}</td>
						<td>{row.entries}</td>
						<td>{row.total_xp}</td>
						<td>{when(row.last_active)}</td>
						<td>
							<div class="actions">
								<a
									class="admin-btn icon-btn"
									href={`/admin/players/${row.player_id}`}
									title="View"
									aria-label="View"
								>
									<Icon icon="ph:eye-bold" width="16" />
								</a>
								<button
									class="admin-btn-danger icon-btn"
									title="Delete"
									aria-label="Delete"
									onclick={() => askRemovePlayer(row.player_id, row.username, row.entries)}
								>
									<Icon icon="ph:trash-bold" width="16" />
								</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="11">No players found.</td></tr>
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
</style>
