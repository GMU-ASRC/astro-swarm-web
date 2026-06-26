<script lang="ts">
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

<div class="min-h-screen flex items-center justify-center px-5 py-8 bg-[#f4f4f5]">
	<form
		class="w-full max-w-sm p-10 bg-white border border-[#e4e4e7] border-t-2 border-t-[#27272a]"
		onsubmit={login}
	>
		<div class="flex flex-col items-center gap-1 pb-6 mb-6 border-b border-[#e4e4e7]">
			<span class="text-xl font-bold tracking-[0.1em] text-[#18181b]">ASTROSWARM</span>
			<span class="text-[0.62rem] tracking-[0.24em] uppercase text-[#a1a1aa]">Admin Console</span>
		</div>

		<h1 class="text-lg text-center text-[#18181b] mb-6">Sign in</h1>

		<label for="apiKey" class="block text-xs font-semibold uppercase tracking-wider text-[#71717a] mb-2">
			Master key
		</label>
		<input
			type="password"
			id="apiKey"
			name="apiKey"
			placeholder="Enter API key"
			required
			class="w-full px-3.5 py-2.5 mb-5 bg-white border border-[#d4d4d8] text-[#18181b] placeholder:text-[#a1a1aa] focus:outline-none focus:border-[#18181b]"
		/>

		{#if error}
			<div class="mb-5 px-3 py-2 bg-red-50 border border-red-200 text-red-700 text-sm">
				{error}
			</div>
		{/if}

		<button
			type="submit"
			disabled={loading}
			class="w-full py-2.5 bg-[#27272a] hover:bg-[#18181b] text-white font-semibold disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
		>
			{loading ? 'Authorizing...' : 'Access panel'}
		</button>
	</form>
</div>
