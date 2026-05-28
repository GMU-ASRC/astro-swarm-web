<script lang="ts">
	import '$lib/css/game-hero.css';
	import Icon from '@iconify/svelte';

	const features = [
		{
			icon: 'ph:robot',
			title: 'DESIGN SPECIES',
			desc: 'Create unique robot species with distinct physical traits, colors, and visual identities.',
		},
		{
			icon: 'ph:brain',
			title: 'PROGRAM BEHAVIOR',
			desc: 'Use a drag-and-drop block editor to define how each species perceives and reacts to its world.',
		},
		{
			icon: 'ph:squares-four',
			title: 'BUILD ARENAS',
			desc: 'Place robots, walls, and obstacles in a resizable arena. Watch emergent swarm patterns unfold.',
		},
		{
			icon: 'ph:video-camera',
			title: 'RECORD & EXPORT',
			desc: 'Record full simulation sessions and export them to H.264 video for sharing.',
		},
		{
			icon: 'ph:globe',
			title: 'SHARE ONLINE',
			desc: 'Upload your configs and replays to the web — let others view, download, and remix your setups.',
		},
		{
			icon: 'ph:shield-chevron',
			title: 'TOWER DEFENSE',
			desc: 'Deploy your swarms in a full 2D pixel-art tower-defense campaign. Coming soon.',
		},
	];
</script>

<section class="hero" id="home">
	<p class="hero-eyebrow">A GODOT 4 PIXEL-ART EXPERIENCE</p>
	<h1 class="hero-title">
		ASTRO<span class="title-accent">SWARM</span>
	</h1>
	<p class="hero-subtitle">
		DESIGN SPECIES. PROGRAM BEHAVIOR.<br />UNLEASH YOUR SWARM.
	</p>
	<div class="hero-actions">
		<a
			href="https://store.steampowered.com"
			target="_blank"
			rel="noreferrer"
			class="btn-primary"
		>
			WISHLIST ON STEAM
		</a>
		<a href="/simulator" class="btn-secondary">TRY SIMULATOR →</a>
	</div>

	<div class="hero-scroll-hint">
		<span>SCROLL</span>
		<Icon icon="ph:caret-down" width="12" />
	</div>
</section>

<div class="divider-line"></div>

<section class="features-section" id="features">
	<div class="section-header">
		<p class="section-eyebrow">WHAT IS ASTROSWARM</p>
		<h2 class="section-title">FEATURES</h2>
	</div>
	<div class="features-grid">
		{#each features as feature}
			<div class="feature-card">
				<div class="feature-icon">
					<Icon icon={feature.icon} width="28" color="var(--color-accent-cyan)" />
				</div>
				<h3 class="feature-title">{feature.title}</h3>
				<p class="feature-desc">{feature.desc}</p>
			</div>
		{/each}
	</div>
</section>
