<script lang="ts">
	import { apiUrl } from '$lib/ts/api';
	import Pagination from '$lib/components/Pagination.svelte';

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
	let sortOrder = $state('rate_desc');

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
				if (sortOrder === 'rate_desc') return (b.overall_success ?? -1) - (a.overall_success ?? -1);
				if (sortOrder === 'rate_asc') return (a.overall_success ?? -1) - (b.overall_success ?? -1);
				if (sortOrder === 'entries_desc') return b.entries - a.entries;
				if (sortOrder === 'name') return (a.username ?? '').localeCompare(b.username ?? '');
				return 0;
			})
	);

	let page = $state(1);
	const pageSize = 25;
	const pagedPlayers = $derived(filtered.slice((page - 1) * pageSize, page * pageSize));

	$effect(() => {
		searchQuery;
		sortOrder;
		page = 1;
	});

	function rate(value: number | null): string {
		return value != null ? `${value}%` : '—';
	}

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : '—';
	}

	async function removePlayer(playerId: string, name: string, entries: number) {
		if (!confirm(`Delete "${name}" and all ${entries} of their entries? This cannot be undone.`)) return;
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

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Username</th>
				<th>Player ID</th>
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
				<tr><td colspan="8">Loading players...</td></tr>
			{:else}
				{#each pagedPlayers as row}
					<tr>
						<td>{row.username}</td>
						<td><code title={row.player_id}>{row.player_id.slice(0, 8)}</code></td>
						<td>{rate(row.overall_success)}</td>
						<td>{rate(row.best_success)}</td>
						<td>{row.entries}</td>
						<td>{row.total_xp}</td>
						<td>{when(row.last_active)}</td>
						<td>
							<div class="actions">
								<a class="admin-btn" href={`/admin/players/${row.player_id}`}>View</a>
								<button
									class="admin-btn-danger"
									onclick={() => removePlayer(row.player_id, row.username, row.entries)}
								>
									Delete
								</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="8">No players found.</td></tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<Pagination bind:page total={filtered.length} {pageSize} />
