<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();
	let profile = $state<Record<string, any> | null>(null);
	let entries = $state<any[]>([]);
	let loading = $state(true);
	let message = $state('');
	let renameValue = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.profilePromise.then((row: any) => {
			if (!active) return;
			profile = row;
			entries = row?.recent_entries ?? [];
			renameValue = row?.username ?? '';
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	function rate(value: number | null | undefined): string {
		return value != null ? `${value}%` : '—';
	}

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : '—';
	}

	function levelName(levelNumber: number): string {
		return levelNumber >= 3 ? `Level ${levelNumber} · Evasion` : `Level ${levelNumber} · Defense`;
	}

	async function rename() {
		const username = renameValue.trim();
		if (!username || username === profile?.username) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/players/${data.id}`), {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json', 'X-API-Key': data.adminKey },
				body: JSON.stringify({ username })
			});
			if (res.ok) {
				const body = await res.json();
				if (profile) profile.username = body.username;
				entries = entries.map((e: any) => ({ ...e, username: body.username }));
				message = `Renamed to ${body.username} across ${body.updated} entries.`;
			} else {
				message = `Failed to rename: ${res.status}`;
			}
		} catch (err) {
			message = `Rename failed: ${err}`;
		}
	}

	async function cancelEntry(id: string) {
		if (!confirm('Cancel this running evaluation?')) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${id}/cancel`), {
				method: 'POST',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 202) {
				entries = entries.map((e: any) => (e.id === id ? { ...e, status: 'cancelled' } : e));
				message = 'Evaluation cancelled.';
			} else {
				message = `Failed to cancel: ${res.status}`;
			}
		} catch (err) {
			message = `Cancel failed: ${err}`;
		}
	}

	async function resimulateEntry(id: string) {
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${id}/resimulate`), {
				method: 'POST',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 202) {
				entries = entries.map((e: any) => (e.id === id ? { ...e, status: 'queued', progress: 0 } : e));
				message = 'Evaluation re-queued.';
			} else {
				message = `Failed to resimulate: ${res.status}`;
			}
		} catch (err) {
			message = `Resimulate failed: ${err}`;
		}
	}

	async function removeEntry(id: string) {
		if (!confirm('Delete this entry? This cannot be undone.')) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/${id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 204) {
				entries = entries.filter((e: any) => e.id !== id);
				message = 'Entry deleted.';
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}

	async function removePlayer() {
		const name = profile?.username ?? 'this player';
		if (!confirm(`Delete "${name}" and every entry they submitted? This cannot be undone.`)) return;
		try {
			const res = await fetch(apiUrl(`/api/evaluations/players/${data.id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok) {
				await goto('/admin/players');
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<p><a href="/admin/players">← All players</a></p>

{#if message}<div class="message">{message}</div>{/if}

{#if loading}
	<p>Loading player...</p>
{:else if !profile}
	<p>Player not found.</p>
{:else}
	<h1>{profile.username}</h1>
	<p class="meta">{profile.player_id}</p>
	<p class="meta">Last active {when(profile.last_active)}</p>

	<div class="actions">
		<a class="admin-btn" href={`/leaderboard/${profile.player_id}`}>Public profile</a>
		<button class="admin-btn-danger" onclick={removePlayer}>Delete player and entries</button>
	</div>

	<div class="stat-grid">
		<div class="stat">
			<div class="label">Rank</div>
			<div>#{profile.overall_rank ?? '—'} of {profile.total_players}</div>
		</div>
		<div class="stat">
			<div class="label">Average Rate</div>
			<div>{rate(profile.overall_success)}</div>
		</div>
		<div class="stat">
			<div class="label">Best Rate</div>
			<div>{rate(profile.best_success)}</div>
		</div>
		<div class="stat">
			<div class="label">Entries</div>
			<div>{profile.entries}</div>
		</div>
		<div class="stat">
			<div class="label">Total XP</div>
			<div>{profile.total_xp}</div>
		</div>
	</div>

	<h2>Rename</h2>
	<div class="actions">
		<input type="text" maxlength="30" bind:value={renameValue} />
		<button onclick={rename} disabled={!renameValue.trim() || renameValue === profile.username}>
			Apply to all entries
		</button>
	</div>

	<h2>Per-level stats</h2>
	{#if profile.levels.length === 0}
		<p>No level stats yet.</p>
	{:else}
		<div class="admin-table-wrap">
			<table>
				<thead>
					<tr>
						<th>Level</th>
						<th>Success Rate</th>
						<th>Rank</th>
						<th>XP</th>
					</tr>
				</thead>
				<tbody>
					{#each profile.levels as level}
						<tr>
							<td>{levelName(level.level_number)}</td>
							<td>{rate(level.success_rate)}</td>
							<td>#{level.rank ?? '—'} of {level.players}</td>
							<td>{level.xp}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<h2>Entries</h2>
	<div class="admin-table-wrap">
		<table>
			<thead>
				<tr>
					<th>ID</th>
					<th>Level</th>
					<th>Defenders</th>
					<th>Version</th>
					<th>Status</th>
					<th>Trials</th>
					<th>Rate</th>
					<th>XP</th>
					<th>Created</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				{#each entries as entry}
					<tr>
						<td><code title={entry.id}>{entry.id.slice(0, 8)}</code></td>
						<td>{entry.level_id || 'farp'}</td>
						<td>{entry.defender_count ?? '—'}</td>
						<td>{entry.game_version ?? 'v0.0.4'}</td>
						<td><span class="pill status-{entry.status}">{entry.status.toUpperCase()}</span></td>
						<td>{entry.trials ?? '—'}</td>
						<td>{rate(entry.success_rate)}</td>
						<td>{entry.xp_awarded ?? '—'}</td>
						<td>{when(entry.created_at)}</td>
						<td>
							<div class="actions">
								<a class="admin-btn" href={`/admin/evaluations/${entry.id}`}>View</a>
								{#if entry.status === 'queued' || entry.status === 'running'}
									<button onclick={() => cancelEntry(entry.id)}>Cancel</button>
								{:else}
									<button onclick={() => resimulateEntry(entry.id)}>Resimulate</button>
								{/if}
								<button class="admin-btn-danger" onclick={() => removeEntry(entry.id)}>Delete</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="10">No entries.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
