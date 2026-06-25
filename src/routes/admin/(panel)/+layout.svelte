<script lang="ts">
	import { page } from '$app/stores';

	let { children } = $props();

	const links = [
		{ href: '/admin', label: 'DASHBOARD' },
		{ href: '/admin/evaluations', label: 'EVALUATIONS' },
		{ href: '/admin/leaderboard', label: 'LEADERBOARD' },
		{ href: '/admin/runs', label: 'SIMULATOR RUNS' }
	];
</script>

<svelte:head>
	<title>Admin Panel — AstroSwarm</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="min-h-screen flex flex-col md:flex-row font-sim pt-16">
	<aside class="w-full md:w-64 shrink-0 bg-page-bg border-r border-sky-500/20 p-6 flex flex-col h-full md:h-[calc(100vh-4rem)] sticky top-16">
		<h2 class="font-game text-xl text-star-white mb-8 tracking-widest text-shadow-sky">
			CMS PANEL
		</h2>
		
		<nav class="flex flex-col gap-2 flex-1">
			{#each links as link}
				<a
					href={link.href}
					class="px-4 py-3 border-2 font-game text-sm tracking-wider transition-colors {($page.url.pathname === link.href || ($page.url.pathname.startsWith(link.href) && link.href !== '/admin')) 
						? 'border-sky-400 bg-sky-500/15 text-sky-200'
						: 'border-transparent text-text-muted hover:border-sky-500/20 hover:text-star-white hover:bg-sky-500/5'}"
				>
					{link.label}
				</a>
			{/each}
		</nav>

		<div class="mt-8 pt-6 border-t border-sky-500/20">
			<form method="POST" action="/admin/logout">
				<button
					type="submit"
					class="w-full px-4 py-3 border-2 border-red-500/30 text-red-300 font-game text-sm tracking-wider hover:bg-red-500/15 transition-colors"
				>
					LOGOUT
				</button>
			</form>
		</div>
	</aside>

	<main class="flex-1 min-w-0 p-6 md:p-10">
		{@render children()}
	</main>
</div>

<style>
	.text-shadow-sky {
		text-shadow: 0 0 10px rgba(56,189,248,0.3);
	}
</style>
