<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiUrl } from '$lib/ts/api';
	import { sessionKey } from '../+layout';

	let loading = $state(false);
	let error = $state('');

	async function login(event: SubmitEvent) {
		event.preventDefault();
		const formData = new FormData(event.currentTarget as HTMLFormElement);
		const username = (formData.get('username') as string)?.trim();
		const password = (formData.get('password') as string) ?? '';

		if (!username || !password) {
			error = 'Username and password are required';
			return;
		}

		loading = true;
		error = '';

		try {
			const res = await fetch(apiUrl('/api/admin/login'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});

			if (res.status === 401) {
				error = 'Invalid username or password';
				return;
			}
			if (!res.ok) {
				error = `Login failed (${res.status})`;
				return;
			}

			const data = await res.json();
			localStorage.setItem(sessionKey, data.token);
			await goto('/admin');
		} catch (err) {
			error = 'Failed to connect to backend API';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Admin Login — AstroSwarm</title>
</svelte:head>

<div class="login-page">
	<form class="login-card" onsubmit={login}>
		<div class="login-brand">
			<span class="brand-title">ASTROSWARM</span>
			<span class="brand-sub">Admin Console</span>
		</div>

		<h1 class="login-heading">Sign in</h1>

		<label for="username" class="field-label">Username</label>
		<input
			type="text"
			id="username"
			name="username"
			autocomplete="username"
			placeholder="Enter username"
			required
			class="field login-field"
		/>

		<label for="password" class="field-label">Password</label>
		<input
			type="password"
			id="password"
			name="password"
			autocomplete="current-password"
			placeholder="Enter password"
			required
			class="field login-field"
		/>

		{#if error}
			<div class="login-error">{error}</div>
		{/if}

		<button type="submit" disabled={loading} class="btn btn-primary login-submit">
			{loading ? 'Authorizing...' : 'Access panel'}
		</button>
	</form>
</div>

<style>
	.login-page {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
		padding: 2rem 1.25rem;
		background: var(--color-ink);
	}

	.login-card {
		width: 100%;
		max-width: 24rem;
		padding: 2.5rem;
		background: var(--color-surface);
		border: 1px solid var(--color-line);
	}

	.login-brand {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.35rem;
		padding-bottom: 1.5rem;
		margin-bottom: 1.75rem;
		border-bottom: 1px solid var(--color-line);
	}

	.brand-title {
		font-family: var(--font-display);
		font-size: 1.15rem;
		letter-spacing: 0.06em;
		color: var(--color-heading);
	}

	.brand-sub {
		font-size: 0.62rem;
		letter-spacing: 0.22em;
		text-transform: uppercase;
		color: var(--color-faint);
	}

	.login-heading {
		margin-bottom: 1.75rem;
		font-size: 1.05rem;
		font-weight: 600;
		text-align: center;
		color: var(--color-heading);
	}

	.login-field {
		margin-bottom: 1.25rem;
	}

	.login-error {
		margin-bottom: 1.25rem;
		padding: 0.6rem 0.85rem;
		background: color-mix(in srgb, var(--color-loss) 10%, transparent);
		border: 1px solid color-mix(in srgb, var(--color-loss) 40%, transparent);
		color: var(--color-loss);
		font-size: 0.85rem;
	}

	.login-submit {
		width: 100%;
	}
</style>
