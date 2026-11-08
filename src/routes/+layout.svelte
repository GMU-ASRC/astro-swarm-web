<script lang="ts">
	import './layout.css';
	import StarField from '$lib/components/StarField.svelte';
	import Navbar from '$lib/components/Navbar.svelte';
	import PageSkeleton from '$lib/components/skeleton/PageSkeleton.svelte';
	import { navigating, page } from '$app/state';
	import { afterNavigate, beforeNavigate } from '$app/navigation';
	import { reportPageLoad } from '$lib/ts/pageLoadStats';

	let { children } = $props();

	const SLOW_ROUTES = new Set(['/levels/[id]', '/leaderboard', '/leaderboard/[id]', '/survive/[id]', '/simulator/[id]', '/downloads']);
	const SKELETON_DELAY_MS = 120;

	const isAdmin = $derived(page.url.pathname.startsWith('/admin'));
	const pendingRoute = $derived(navigating.to?.route.id ?? null);
	const waitingOnSlowRoute = $derived(
		pendingRoute !== null && SLOW_ROUTES.has(pendingRoute) && navigating.to?.url.pathname !== page.url.pathname
	);

	let showSkeleton = $state(false);
	let navigationStartedAt: number | null = null;

	beforeNavigate((navigation) => {
		navigationStartedAt = navigation.willUnload ? null : performance.now();
	});

	afterNavigate((navigation) => {
		const route = navigation.to?.route.id ?? null;
		if (navigation.type === 'enter') {
			reportPageLoad(route, 'first', performance.now());
		} else if (navigationStartedAt !== null) {
			reportPageLoad(route, 'navigation', performance.now() - navigationStartedAt);
		}
		navigationStartedAt = null;
	});

	$effect(() => {
		if (!waitingOnSlowRoute) {
			showSkeleton = false;
			return;
		}
		const timer = setTimeout(() => (showSkeleton = true), SKELETON_DELAY_MS);
		return () => clearTimeout(timer);
	});
</script>

<svelte:head>
	<title>AstroSwarm</title>
	<meta name="description" content="Design your fleet, program its behavior, and send it against the swarm. AstroSwarm is a swarm strategy game built in Godot 4." />
</svelte:head>

<div class="app-root" style="display: contents">
	{#if !isAdmin}
		<StarField />
		<Navbar />
	{/if}

	{#if showSkeleton}
		<PageSkeleton />
	{:else}
		{@render children()}
	{/if}
</div>
