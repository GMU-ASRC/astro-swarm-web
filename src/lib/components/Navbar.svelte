<script lang="ts">
	import '$lib/css/navbar.css';
	import { page } from '$app/state';

	const currentPath = $derived(page.url.pathname);
	let menuOpen = $state(false);

	function closeMenu() {
		menuOpen = false;
	}
</script>

<nav
	class="fixed top-0 left-0 right-0 z-[100] flex items-center justify-between px-8 max-sm:px-5 h-16"
	style="background: linear-gradient(to bottom, rgba(2,4,9,0.95) 0%, rgba(2,4,9,0) 100%)"
>
	<a
		href="/"
		class="font-game text-[1.4rem] text-star-white no-underline tracking-[0.05em] z-[101]"
		onclick={closeMenu}
	>
		ASTROSWARM
	</a>

	<ul class="hidden sm:flex items-center gap-8 list-none m-0 p-0">
		{#each [
			{ href: '/', label: 'HOME', active: currentPath === '/' },
			{ href: '/previews', label: 'PREVIEWS', active: currentPath === '/previews' },
			{ href: '/downloads', label: 'DOWNLOADS', active: currentPath === '/downloads' },
			{ href: '/leaderboard', label: 'LEADERBOARD', active: currentPath.startsWith('/leaderboard') },
			{ href: '/simulator', label: 'SIMULATOR', active: currentPath.startsWith('/simulator') }
		] as link}
			<li>
				<a
					href={link.href}
					target={link.external ? '_blank' : undefined}
					rel={link.external ? 'noreferrer' : undefined}
					class="font-game text-base no-underline tracking-[0.08em] transition-colors hover:text-accent-cyan
						{link.active ? 'text-star-white' : 'text-text-muted'}"
				>
					{link.label}
				</a>
			</li>
		{/each}
	</ul>

	<button
		class="navbar-hamburger flex sm:hidden"
		class:open={menuOpen}
		onclick={() => (menuOpen = !menuOpen)}
		aria-label="Toggle menu"
	>
		<span></span>
		<span></span>
		<span></span>
	</button>
</nav>

<ul class="navbar-drawer" class:open={menuOpen}>
	<li><a href="/" class:active={currentPath === '/'} onclick={closeMenu}>HOME</a></li>
	<li><a href="/previews" class:active={currentPath === '/previews'} onclick={closeMenu}>PREVIEWS</a></li>
	<li><a href="/downloads" class:active={currentPath === '/downloads'} onclick={closeMenu}>DOWNLOADS</a></li>
	<li><a href="/leaderboard" class:active={currentPath.startsWith('/leaderboard')} onclick={closeMenu}>LEADERBOARD</a></li>
	<li><a href="/simulator" class:active={currentPath.startsWith('/simulator')} onclick={closeMenu}>SIMULATOR</a></li>
</ul>
