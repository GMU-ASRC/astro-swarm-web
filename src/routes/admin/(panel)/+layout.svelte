<script lang="ts">
	import '$lib/css/admin.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { sessionKey } from '../+layout';

	let { children } = $props();

	async function logout() {
		localStorage.removeItem(sessionKey);
		await goto('/admin/login');
	}

	const links = [
		{ href: '/admin', label: 'Dashboard' },
		{ href: '/admin/evaluations', label: 'Evaluations' },
		{ href: '/admin/leaderboard', label: 'Leaderboard' },
		{ href: '/admin/runs', label: 'Simulator Runs' },
		{ href: '/admin/settings', label: 'Settings' }
	];

	function isActive(href: string): boolean {
		return $page.url.pathname === href || ($page.url.pathname.startsWith(href) && href !== '/admin');
	}
</script>

<svelte:head>
	<title>Admin Panel — AstroSwarm</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="admin">
	<div class="admin-layout">
		<aside class="admin-sidebar">
			<h2>CMS Panel</h2>
			<nav>
				{#each links as link}
					<a href={link.href} class:active={isActive(link.href)}>{link.label}</a>
				{/each}
			</nav>
			<p><button type="button" class="admin-btn-danger" onclick={logout}>Logout</button></p>
		</aside>

		<main class="admin-content">
			{@render children()}
		</main>
	</div>
</div>
