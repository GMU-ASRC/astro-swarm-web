<script lang="ts">
	import '$lib/css/admin.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { apiUrl } from '$lib/ts/api';
	import { sessionKey } from '../+layout';

	let { children } = $props();

	async function logout() {
		const token = localStorage.getItem(sessionKey) ?? '';
		try {
			await fetch(apiUrl('/api/admin/logout'), {
				method: 'POST',
				headers: { 'X-API-Key': token }
			});
		} catch {
			// ignore network errors; clear the session locally regardless
		}
		localStorage.removeItem(sessionKey);
		await goto('/admin/login');
	}

	const links = [
		{ href: '/admin', label: 'Dashboard', icon: 'dashboard' },
		{ href: '/admin/evaluations', label: 'Evaluations', icon: 'evaluations' },
		{ href: '/admin/leaderboard', label: 'Leaderboard', icon: 'leaderboard' },
		{ href: '/admin/runs', label: 'Simulator Runs', icon: 'runs' },
		{ href: '/admin/workers', label: 'Workers', icon: 'workers' },
		{ href: '/admin/settings', label: 'Settings', icon: 'settings' },
		{ href: '/admin/account', label: 'Account', icon: 'account' }
	];

	function isActive(href: string): boolean {
		return $page.url.pathname === href || ($page.url.pathname.startsWith(href) && href !== '/admin');
	}
</script>

<svelte:head>
	<title>Admin Panel — AstroSwarm</title>
	<meta name="robots" content="noindex" />
</svelte:head>

{#snippet icon(name: string)}
	<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
		{#if name === 'dashboard'}
			<rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="14" y="14" width="7" height="7" /><rect x="3" y="14" width="7" height="7" />
		{:else if name === 'evaluations'}
			<path d="M9 11l3 3L22 4" /><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
		{:else if name === 'leaderboard'}
			<path d="M4 20V10" /><path d="M12 20V4" /><path d="M20 20v-6" />
		{:else if name === 'runs'}
			<polygon points="5 3 19 12 5 21 5 3" />
		{:else if name === 'workers'}
			<rect x="4" y="4" width="16" height="6" rx="1" /><rect x="4" y="14" width="16" height="6" rx="1" /><line x1="8" y1="7" x2="8" y2="7" /><line x1="8" y1="17" x2="8" y2="17" />
		{:else if name === 'settings'}
			<circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 8 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H2a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 3.6 8a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H8a1.65 1.65 0 0 0 1-1.51V2a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V8a1.65 1.65 0 0 0 1.51 1H22a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
		{:else if name === 'account'}
			<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
		{/if}
	</svg>
{/snippet}

<div class="admin">
	<div class="admin-layout">
		<aside class="admin-sidebar">
			<div class="admin-brand">
				<span class="brand-title">ASTROSWARM</span>
				<span class="brand-sub">Admin Console</span>
			</div>

			<nav>
				{#each links as link}
					<a href={link.href} class:active={isActive(link.href)}>
						{@render icon(link.icon)}
						<span>{link.label}</span>
					</a>
				{/each}
			</nav>

			<div class="sidebar-footer">
				<button type="button" class="sidebar-logout" onclick={logout}>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
						<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><polyline points="16 17 21 12 16 7" /><line x1="21" y1="12" x2="9" y2="12" />
					</svg>
					<span>Logout</span>
				</button>
			</div>
		</aside>

		<main class="admin-content">
			{@render children()}
		</main>
	</div>
</div>
