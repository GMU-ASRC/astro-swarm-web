<script lang="ts">
	import Icon from '@iconify/svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { goto } from '$app/navigation';
	import { levelName } from '$lib/ts/levels';
	import { apiUrl } from '$lib/ts/api';
	import { runBulk, bulkMessage } from '$lib/ts/bulk';
	import { createPrompt } from '$lib/ts/prompt.svelte';
	import BulkActionBar from '$lib/components/BulkActionBar.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	const MANY_ENTRIES = 10;

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

	const selected = new SvelteSet<string>();
	const selectedEntries = $derived(entries.filter((entry: any) => selected.has(entry.id)));
	const allEntriesSelected = $derived(
		entries.length > 0 && entries.every((entry: any) => selected.has(entry.id))
	);
	const resimulatableEntries = $derived(
		selectedEntries.filter((entry: any) => entry.status !== 'queued' && entry.status !== 'running')
	);

	function toggleRow(id: string) {
		if (selected.has(id)) selected.delete(id);
		else selected.add(id);
	}

	function toggleAll() {
		if (allEntriesSelected) selected.clear();
		else for (const entry of entries) selected.add(entry.id);
	}

	const prompt = createPrompt();

	function rate(value: number | null | undefined): string {
		return value != null ? `${value}%` : '—';
	}

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : '—';
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

	async function removeSelected() {
		const ids = selectedEntries.map((entry: any) => entry.id);
		message = `Deleting ${ids.length} entries...`;
		const result = await runBulk(ids, data.adminKey, 'DELETE', (id) => `/api/evaluations/${id}`);
		entries = entries.filter((e: any) => !result.succeeded.includes(e.id));
		selected.clear();
		message = bulkMessage('Deleted', 'entries', result);
	}

	async function resimulateSelected() {
		const ids = resimulatableEntries.map((entry: any) => entry.id);
		message = `Queueing ${ids.length} entries...`;
		const result = await runBulk(
			ids,
			data.adminKey,
			'POST',
			(id) => `/api/evaluations/${id}/resimulate`
		);
		entries = entries.map((e: any) =>
			result.succeeded.includes(e.id) ? { ...e, status: 'queued', progress: 0 } : e
		);
		selected.clear();
		message = bulkMessage('Queued', 'entries', result);
	}

	function askRemoveSelected() {
		const count = selectedEntries.length;
		prompt.ask({
			title: 'Delete entries',
			message: `Delete ${count} selected ${count === 1 ? 'entry' : 'entries'}? Their results and replays cannot be recovered.`,
			warning: count >= MANY_ENTRIES ? `That is a lot of entries to delete at once.` : '',
			confirmLabel: 'Delete',
			danger: true,
			run: removeSelected
		});
	}

	function askResimulateSelected() {
		const count = resimulatableEntries.length;
		const skipped = selectedEntries.length - count;
		prompt.ask({
			title: 'Re-simulate entries',
			message: `Re-run ${count} selected ${count === 1 ? 'entry' : 'entries'} with the current simulator build? This overwrites their results and replays.`,
			warning:
				skipped > 0
					? `${skipped} selected ${skipped === 1 ? 'entry is' : 'entries are'} already queued or running and will be skipped.`
					: count >= MANY_ENTRIES
						? 'That is a lot of entries to re-simulate at once.'
						: '',
			confirmLabel: 'Re-simulate',
			run: resimulateSelected
		});
	}

	function askCancelEntry(id: string) {
		prompt.ask({
			title: 'Cancel run',
			message: 'Cancel this running evaluation?',
			confirmLabel: 'Cancel run',
			danger: true,
			run: () => cancelEntry(id)
		});
	}

	function askRemoveEntry(id: string) {
		prompt.ask({
			title: 'Delete entry',
			message: 'Delete this entry? Its results and replays cannot be recovered.',
			confirmLabel: 'Delete',
			danger: true,
			run: () => removeEntry(id)
		});
	}

	function askRemovePlayer() {
		const name = profile?.username ?? 'this player';
		const count = profile?.entries ?? entries.length;
		prompt.ask({
			title: 'Delete player',
			message: `Delete "${name}" and all ${count} of their ${count === 1 ? 'entry' : 'entries'}? This cannot be undone.`,
			warning: count >= MANY_ENTRIES ? `That deletes ${count} entries in one go.` : '',
			confirmLabel: 'Delete',
			danger: true,
			run: removePlayer
		});
	}

	async function cancelEntry(id: string) {
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
		<button class="admin-btn-danger" onclick={askRemovePlayer}>Delete player and entries</button>
	</div>

	<div class="stat-grid">
		<div class="stat">
			<div class="label">Rank</div>
			<div>#{profile.overall_rank ?? '—'} of {profile.total_players}</div>
		</div>
		<div class="stat">
			<div class="label">Rating</div>
			<div>{profile.rating ?? '—'}</div>
		</div>
		<div class="stat">
			<div class="label">Levels Played</div>
			<div>{profile.levels_played}/{profile.levels_total}</div>
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
						<th>Weighted</th>
						<th>Level Average</th>
						<th>Entries</th>
						<th>Rank</th>
						<th>XP</th>
					</tr>
				</thead>
				<tbody>
					{#each profile.levels as level}
						<tr>
							<td>{levelName(level.level_number)}</td>
							<td>{rate(level.success_rate)}</td>
							<td>{level.weighted_rate ?? '—'}</td>
							<td>{rate(level.level_average)}</td>
							<td>{level.entries}</td>
							<td>#{level.rank ?? '—'} of {level.players}</td>
							<td>{level.xp}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<h2>Entries</h2>

	<BulkActionBar
		count={selectedEntries.length}
		noun="entry"
		nounPlural="entries"
		onclear={() => selected.clear()}
	>
		<button
			type="button"
			onclick={askResimulateSelected}
			disabled={resimulatableEntries.length === 0}
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
							aria-label="Select all entries"
							checked={allEntriesSelected}
							indeterminate={selectedEntries.length > 0 && !allEntriesSelected}
							onchange={toggleAll}
						/>
					</th>
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
					<tr class:selected={selected.has(entry.id)}>
						<td class="select-cell">
							<input
								type="checkbox"
								aria-label={`Select entry ${entry.id.slice(0, 8)}`}
								checked={selected.has(entry.id)}
								onchange={() => toggleRow(entry.id)}
							/>
						</td>
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
								<a
									class="admin-btn icon-btn"
									href={`/admin/evaluations/${entry.id}`}
									title="View"
									aria-label="View"
								>
									<Icon icon="ph:eye-bold" width="16" />
								</a>
								{#if entry.status === 'queued' || entry.status === 'running'}
									<button
										class="icon-btn"
										title="Cancel run"
										aria-label="Cancel run"
										onclick={() => askCancelEntry(entry.id)}
									>
										<Icon icon="ph:prohibit-bold" width="16" />
									</button>
								{:else}
									<button
										class="icon-btn"
										title="Re-simulate"
										aria-label="Re-simulate"
										onclick={() => resimulateEntry(entry.id)}
									>
										<Icon icon="ph:arrows-clockwise-bold" width="16" />
									</button>
								{/if}
								<button
									class="admin-btn-danger icon-btn"
									title="Delete"
									aria-label="Delete"
									onclick={() => askRemoveEntry(entry.id)}
								>
									<Icon icon="ph:trash-bold" width="16" />
								</button>
							</div>
						</td>
					</tr>
				{:else}
					<tr><td colspan="11">No entries.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
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
	tbody tr.selected td {
		background: color-mix(in srgb, var(--color-brand) 10%, transparent);
	}
</style>
