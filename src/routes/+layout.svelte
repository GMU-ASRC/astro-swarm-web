<script lang="ts">
	import './layout.css';
	import StarField from '$lib/components/StarField.svelte';
	import Navbar from '$lib/components/Navbar.svelte';
	import { page } from '$app/state';

	let { children } = $props();

	const isAdmin = $derived(page.url.pathname.startsWith('/admin'));
</script>

<svelte:head>
	<title>AstroSwarm</title>
	<meta name="description" content="Design species, program behavior, unleash your swarm. AstroSwarm — a pixel-art tower-defense and swarm simulator built in Godot 4." />
</svelte:head>

{#if !isAdmin}
	<StarField />
	<Navbar />
{/if}

{@render children()}
