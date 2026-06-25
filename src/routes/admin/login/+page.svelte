<script lang="ts">
	import '$lib/css/admin.css';
	import { goto } from '$app/navigation';
	import { apiUrl } from '$lib/ts/api';
	import { sessionKey } from '../+layout';

	let loading = $state(false);
	let error = $state('');

	async function login(event: SubmitEvent) {
		event.preventDefault();
		const formData = new FormData(event.currentTarget as HTMLFormElement);
		const apiKey = (formData.get('apiKey') as string)?.trim();

		if (!apiKey) {
			error = 'API key is required';
			return;
		}

		loading = true;
		error = '';

		try {
			const res = await fetch(apiUrl('/api/evaluations/test-auth'), {
				method: 'DELETE',
				headers: { 'X-API-Key': apiKey }
			});

			if (res.status === 401 || res.status === 403) {
				error = 'Invalid API Key';
				return;
			}

			localStorage.setItem(sessionKey, apiKey);
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

<div class="admin">
	<form class="login" onsubmit={login}>
		<h1>Admin Login</h1>
		<label for="apiKey">Master key</label>
		<input type="password" id="apiKey" name="apiKey" placeholder="API key" required />
		{#if error}<div class="message">{error}</div>{/if}
		<button type="submit" disabled={loading}>{loading ? 'Authorizing...' : 'Access panel'}</button>
	</form>
</div>
