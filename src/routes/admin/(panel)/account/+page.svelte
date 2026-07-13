<script lang="ts">
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();
	const headers = $derived({ 'X-API-Key': data.adminKey, 'Content-Type': 'application/json' });

	let me = $state<any>(null);
	let users = $state<any[]>([]);
	let message = $state('');
	let error = $state('');

	let currentPassword = $state('');
	let newPassword = $state('');
	let newUsername = $state('');
	let newUserPassword = $state('');

	async function refresh() {
		try {
			const meRes = await fetch(apiUrl('/api/admin/me'), { headers: { 'X-API-Key': data.adminKey } });
			me = meRes.ok ? await meRes.json() : null;
			const usersRes = await fetch(apiUrl('/api/admin/users'), { headers: { 'X-API-Key': data.adminKey } });
			users = usersRes.ok ? await usersRes.json() : [];
		} catch (err) {
			error = `Failed to load account: ${err}`;
		}
	}

	$effect(() => {
		refresh();
	});

	function when(iso: string | null): string {
		return iso ? new Date(iso).toLocaleString() : '—';
	}

	async function changePassword() {
		message = '';
		error = '';
		if (newPassword.length < 8) {
			error = 'New password must be at least 8 characters.';
			return;
		}
		try {
			const res = await fetch(apiUrl('/api/admin/password'), {
				method: 'POST',
				headers,
				body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
			});
			if (res.ok) {
				message = 'Password updated.';
				currentPassword = '';
				newPassword = '';
			} else {
				const body = await res.json().catch(() => ({}));
				error = body.error || `Failed to change password (${res.status}).`;
			}
		} catch (err) {
			error = `Change failed: ${err}`;
		}
	}

	async function addUser() {
		message = '';
		error = '';
		if (!newUsername.trim() || newUserPassword.length < 8) {
			error = 'Username is required and password must be at least 8 characters.';
			return;
		}
		try {
			const res = await fetch(apiUrl('/api/admin/users'), {
				method: 'POST',
				headers,
				body: JSON.stringify({ username: newUsername.trim(), password: newUserPassword })
			});
			if (res.ok) {
				message = `Added admin user "${newUsername.trim()}".`;
				newUsername = '';
				newUserPassword = '';
				await refresh();
			} else {
				const body = await res.json().catch(() => ({}));
				error = body.error || `Failed to add user (${res.status}).`;
			}
		} catch (err) {
			error = `Add failed: ${err}`;
		}
	}

	async function removeUser(id: string, username: string) {
		if (!confirm(`Remove admin user "${username}"?`)) return;
		message = '';
		error = '';
		try {
			const res = await fetch(apiUrl(`/api/admin/users/${id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok) {
				message = `Removed "${username}".`;
				await refresh();
			} else {
				const body = await res.json().catch(() => ({}));
				error = body.error || `Failed to remove user (${res.status}).`;
			}
		} catch (err) {
			error = `Remove failed: ${err}`;
		}
	}
</script>

<h1>Account</h1>
<p class="meta">Signed in as <strong>{me?.username ?? '…'}</strong>{me?.master ? ' (master key)' : ''}.</p>

{#if message}<div class="message">{message}</div>{/if}
{#if error}<div class="message" style="color:#b91c1c">{error}</div>{/if}

<h2>Change password</h2>
{#if me?.master}
	<p class="meta">You are signed in with the master key, which has no password to change. Sign in as a user account to manage its password.</p>
{:else}
	<div class="stat-grid">
		<div class="stat">
			<div class="label">Current password</div>
			<input type="password" autocomplete="current-password" bind:value={currentPassword} style="width:100%" />
		</div>
		<div class="stat">
			<div class="label">New password</div>
			<input type="password" autocomplete="new-password" bind:value={newPassword} style="width:100%" />
		</div>
	</div>
	<div class="actions">
		<button onclick={changePassword} disabled={!currentPassword || !newPassword}>Update password</button>
	</div>
{/if}

<h2>Admin users</h2>
<div class="admin-table-wrap">
	<table>
		<thead>
			<tr><th>Username</th><th>Created</th><th>Last login</th><th></th></tr>
		</thead>
		<tbody>
			{#each users as user}
				<tr>
					<td>{user.username}</td>
					<td>{when(user.created_at)}</td>
					<td>{when(user.last_login)}</td>
					<td>
						<button class="admin-btn-danger" onclick={() => removeUser(user.id, user.username)}>Remove</button>
					</td>
				</tr>
			{:else}
				<tr><td colspan="4">No admin users.</td></tr>
			{/each}
		</tbody>
	</table>
</div>

<h2>Add admin user</h2>
<div class="stat-grid">
	<div class="stat">
		<div class="label">Username</div>
		<input type="text" autocomplete="off" bind:value={newUsername} style="width:100%" />
	</div>
	<div class="stat">
		<div class="label">Password (min 8 chars)</div>
		<input type="password" autocomplete="new-password" bind:value={newUserPassword} style="width:100%" />
	</div>
</div>
<div class="actions">
	<button onclick={addUser} disabled={!newUsername.trim() || newUserPassword.length < 8}>Add user</button>
</div>
