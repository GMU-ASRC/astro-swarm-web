<script lang="ts">
	import './layout.css';
	import StarField from '$lib/components/StarField.svelte';
	import Navbar from '$lib/components/Navbar.svelte';

	let { children } = $props();
</script>

<svelte:head>
	<title>AstroSwarm</title>
	<meta name="description" content="Design species, program behavior, unleash your swarm. AstroSwarm — a pixel-art tower-defense and swarm simulator built in Godot 4." />
</svelte:head>

<StarField />
<Navbar />

{@render children()}
