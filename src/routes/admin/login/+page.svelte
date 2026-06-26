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

<div class="min-h-screen flex items-center justify-center px-5 py-8 bg-[#0a0f28]">
	<form
		class="w-full max-w-sm p-10 bg-[#111a3d] border border-[#213160] border-t-2 border-t-[#5b8cff] rounded-xl"
		onsubmit={login}
	>
		<div class="flex flex-col items-center gap-1 pb-6 mb-6 border-b border-[#213160]">
			<span class="text-xl font-bold tracking-[0.1em] text-white">ASTROSWARM</span>
			<span class="text-[0.62rem] tracking-[0.24em] uppercase text-[#5f76b8]">Admin Console</span>
		</div>

		<h1 class="text-lg text-center text-[#eaf0ff] mb-6">Sign in</h1>

		<label for="apiKey" class="block text-xs font-semibold uppercase tracking-wider text-[#8ea2d6] mb-2">
			Master key
		</label>
		<input
			type="password"
			id="apiKey"
			name="apiKey"
			placeholder="Enter API key"
			required
			class="w-full px-3.5 py-2.5 mb-5 bg-[#0c1230] border border-[#2a3a6b] rounded-lg text-[#eaf0ff] placeholder:text-[#5b6da0] focus:outline-none focus:border-[#5b8cff]"
		/>

		{#if error}
			<div class="mb-5 px-3 py-2 rounded-lg bg-red-500/15 border border-red-500/30 text-red-200 text-sm">
				{error}
			</div>
		{/if}

		<button
			type="submit"
			disabled={loading}
			class="w-full py-2.5 rounded-lg bg-[#3f6bea] hover:bg-[#4f7bf0] text-white font-semibold disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
		>
			{loading ? 'Authorizing...' : 'Access panel'}
		</button>
	</form>
</div>
