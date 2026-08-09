<script lang="ts">
	import Pagination from '$lib/components/Pagination.svelte';

	let { data } = $props();
	let players = $state<any[]>([]);
	let loading = $state(true);

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

	const filtered = $derived(
		players.filter((row: any) => {
			if (searchQuery.trim() === '') return true;
			const query = searchQuery.toLowerCase();
			const name = (row.username ?? '').toLowerCase();
			const id = (row.player_id ?? '').toLowerCase();
			return name.includes(query) || id.includes(query);
		})
	);

	let page = $state(1);
	const pageSize = 25;
	const pagedPlayers = $derived(filtered.slice((page - 1) * pageSize, page * pageSize));

	$effect(() => {
		searchQuery;
		page = 1;
	});

	function rate(value: number | null): string {
		return value != null ? `${value}%` : '—';
	}

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : '—';
	}
</script>

<h1>Success Rate Leaderboard</h1>
<p>Standings across every benchmarked level, ranked by average success rate.</p>

<div class="admin-filters">
	<div class="filter-field grow">
		<label for="lb-search">Search</label>
		<input id="lb-search" type="text" placeholder="Username or player ID..." bind:value={searchQuery} />
	</div>
</div>

{#if !loading}
	<p class="filter-summary">Showing {filtered.length} of {players.length} commanders</p>
{/if}

<div class="admin-table-wrap">
	<table>
		<thead>
			<tr>
				<th>Rank</th>
				<th>Username</th>
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
				<tr><td colspan="8">Loading leaderboard...</td></tr>
			{:else}
				{#each pagedPlayers as row}
					<tr>
						<td>#{row.rank}</td>
						<td>{row.username}</td>
						<td>{rate(row.overall_success)}</td>
						<td>{rate(row.best_success)}</td>
						<td>{row.entries}</td>
						<td>{row.total_xp}</td>
						<td>{when(row.last_active)}</td>
						<td>
							<div class="actions">
								<a class="admin-btn" href={`/admin/players/${row.player_id}`}>Manage</a>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="8">No commanders found.</td></tr>
				{/each}
			{/if}
		</tbody>
	</table>
</div>

<Pagination bind:page total={filtered.length} {pageSize} />
