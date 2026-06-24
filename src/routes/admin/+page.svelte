<script lang="ts">
	import { onMount } from 'svelte';
	import { apiUrl } from '$lib/ts/api';

	interface Row {
		id: string;
		primary: string;
		secondary: string;
		deletePath: string;
	}

	let apiKey = $state('');
	let evaluations: Row[] = $state([]);
	let leaderboard: Row[] = $state([]);
	let runs: Row[] = $state([]);
	let message = $state('');
	let loading = $state(false);

	function when(iso: string): string {
		return new Date(iso).toLocaleString();
	}

	async function loadAll() {
		loading = true;
		message = '';
		try {
			const [evRes, lbRes, runRes] = await Promise.all([
				fetch(apiUrl('/api/evaluations')),
				fetch(apiUrl('/api/leaderboard')),
				fetch(apiUrl('/api/runs?page=1&page_size=100'))
			]);

			evaluations = (await evRes.json()).map((e: any) => ({
				id: e.id,
				primary: e.username,
				secondary: `${e.level_id ?? 'farp'} · ${e.status} · ${e.success_rate ?? '—'}% · ${when(e.created_at)}`,
				deletePath: `/api/evaluations/${e.id}`
			}));

			leaderboard = (await lbRes.json()).map((e: any) => ({
				id: e.id,
				primary: e.username,
				secondary: `${e.time_seconds}s · ${when(e.created_at)}`,
				deletePath: `/api/leaderboard/${e.id}`
			}));

			const runData = await runRes.json();
			runs = (runData.items ?? []).map((r: any) => ({
				id: r.id,
				primary: r.title,
				secondary: `${r.author} · ${when(r.created_at)}`,
				deletePath: `/api/runs/${r.id}`
			}));
		} catch (err) {
			message = `Failed to load data: ${err}`;
		} finally {
			loading = false;
		}
	}

	async function remove(row: Row, list: 'evaluations' | 'leaderboard' | 'runs') {
		if (!apiKey) {
			message = 'Enter the admin API key first.';
			return;
		}
		if (!confirm(`Delete "${row.primary}"? This cannot be undone.`)) return;
		try {
			const res = await fetch(apiUrl(row.deletePath), {
				method: 'DELETE',
				headers: { 'X-API-Key': apiKey }
			});
			if (res.status === 204 || res.ok) {
				if (list === 'evaluations') evaluations = evaluations.filter((r) => r.id !== row.id);
				else if (list === 'leaderboard') leaderboard = leaderboard.filter((r) => r.id !== row.id);
				else runs = runs.filter((r) => r.id !== row.id);
				message = `Deleted ${row.primary}.`;
			} else {
				message = `Delete failed (${res.status}). Check the API key.`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}

	onMount(() => {
		apiKey = localStorage.getItem('astroswarm_admin_key') ?? '';
		loadAll();
	});

	$effect(() => {
		if (apiKey) localStorage.setItem('astroswarm_admin_key', apiKey);
	});
</script>

<svelte:head>
	<title>Admin — AstroSwarm</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="relative z-1 min-h-screen pt-20 font-sim">
	<div class="max-w-225 mx-auto px-8 max-sm:px-5 pt-12 pb-6">
		<h1 class="font-game text-[clamp(1.6rem,4vw,2.6rem)] font-bold text-star-white leading-tight mb-3" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
			Admin
		</h1>
		<p class="text-sm text-text-muted mb-4">Manage and delete stored data. Deletes require the admin API key.</p>
		<div class="flex flex-wrap items-center gap-3">
			<input
				type="password"
				placeholder="Admin API key"
				bind:value={apiKey}
				class="px-3 py-2 bg-page-bg/60 border-2 border-sky-500/20 text-star-white text-sm w-64 focus:border-sky-400/50 outline-none"
			/>
			<button
				type="button"
				onclick={loadAll}
				class="px-4 py-2 border-2 border-sky-500/30 text-sky-200 text-sm font-game tracking-wider hover:bg-sky-500/10"
			>
				REFRESH
			</button>
			{#if loading}<span class="text-xs text-text-muted">Loading…</span>{/if}
		</div>
		{#if message}
			<div class="mt-3 text-xs text-sky-200">{message}</div>
		{/if}
	</div>

	<div class="max-w-225 mx-auto px-8 max-sm:px-5 py-6 space-y-10">
		{#snippet section(title: string, rows: Row[], list: 'evaluations' | 'leaderboard' | 'runs')}
			<div>
				<h2 class="font-game text-xl text-star-white mb-3">{title} <span class="text-text-muted text-sm">({rows.length})</span></h2>
				{#if rows.length === 0}
					<div class="text-xs text-text-muted">None.</div>
				{:else}
					<div class="flex flex-col gap-2">
						{#each rows as row}
							<div class="flex items-center justify-between gap-3 p-3 border border-sky-500/15 bg-sky-500/5">
								<div class="min-w-0">
									<div class="text-sm text-star-white truncate">{row.primary}</div>
									<div class="text-[11px] text-text-muted truncate">{row.secondary}</div>
								</div>
								<button
									type="button"
									onclick={() => remove(row, list)}
									class="shrink-0 px-3 py-1.5 border border-red-400/40 text-red-300 text-xs font-game tracking-wider hover:bg-red-500/15"
								>
									DELETE
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/snippet}

		{@render section('Evaluations', evaluations, 'evaluations')}
		{@render section('Leaderboard', leaderboard, 'leaderboard')}
		{@render section('Simulator Runs', runs, 'runs')}
	</div>
</div>
