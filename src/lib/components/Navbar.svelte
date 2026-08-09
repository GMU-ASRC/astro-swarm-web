<script lang="ts">
	import '$lib/css/navbar.css';
	import Icon from '@iconify/svelte';
	import { page } from '$app/state';
	import { REPO } from '$lib/ts/github';

	const LAB_URL = 'https://create.autonomousrobotics.club/projects/astroswarm';

	const links = [
		{ href: '/', label: 'Home', match: (path: string) => path === '/' },
		{ href: '/gamemodes', label: 'Game Modes', match: (path: string) => path.startsWith('/gamemodes') || path.startsWith('/levels') || path.startsWith('/survive') },
		{ href: '/leaderboard', label: 'Leaderboard', match: (path: string) => path.startsWith('/leaderboard') },
		{ href: '/previews', label: 'Screenshots', match: (path: string) => path.startsWith('/previews') },
		{ href: '/downloads', label: 'Downloads', match: (path: string) => path.startsWith('/downloads') }
	];

	const currentPath = $derived(page.url.pathname);
	let menuOpen = $state(false);

	function closeMenu() {
		menuOpen = false;
	}
</script>

<nav class="navbar">
	<a href="/" class="navbar-brand" onclick={closeMenu}>ASTROSWARM</a>

	<ul class="navbar-links">
		{#each links as link}
			<li>
				<a href={link.href} class:active={link.match(currentPath)}>{link.label}</a>
			</li>
		{/each}
	</ul>

	<div class="navbar-actions">
		<a
			href={`https://github.com/${REPO}`}
			target="_blank"
			rel="noreferrer"
			class="navbar-icon-link"
			aria-label="AstroSwarm on GitHub"
		>
			<Icon icon="ph:github-logo-fill" width="20" />
		</a>

		<a
			href={LAB_URL}
			target="_blank"
			rel="noreferrer"
			class="navbar-icon-link"
			aria-label="AstroSwarm at the Autonomous Robotics Club"
			title="Research lab project page"
		>
			<Icon icon="ph:flask-fill" width="20" />
		</a>

		<button
			class="navbar-hamburger"
			class:open={menuOpen}
			onclick={() => (menuOpen = !menuOpen)}
			aria-label="Toggle menu"
			aria-expanded={menuOpen}
		>
			<span></span>
			<span></span>
			<span></span>
		</button>
	</div>
</nav>

<ul class="navbar-drawer" class:open={menuOpen}>
	{#each links as link}
		<li>
			<a href={link.href} class:active={link.match(currentPath)} onclick={closeMenu}>{link.label}</a>
		</li>
	{/each}
</ul>
