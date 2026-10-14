<script lang="ts">
	import SettingsSection from '$lib/components/admin/settings/SettingsSection.svelte';
	import GameVersionForm from '$lib/components/admin/settings/GameVersionForm.svelte';
	import LevelSettingsTable from '$lib/components/admin/settings/LevelSettingsTable.svelte';
	import BaseSeedForm from '$lib/components/admin/settings/BaseSeedForm.svelte';
	import RecordedRunLimits from '$lib/components/admin/settings/RecordedRunLimits.svelte';
	import LegacyBenchmarkSettings from '$lib/components/admin/settings/LegacyBenchmarkSettings.svelte';
	import { tick } from 'svelte';
	import { putSettings, type AdminSettings } from '$lib/ts/adminSettings';

	let { data } = $props();

	let settings = $state<AdminSettings | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let notice = $state<{ text: string; error: boolean } | null>(null);

	const sections = [
		{ id: 'submissions', label: 'Submissions' },
		{ id: 'levels', label: 'Levels' },
		{ id: 'benchmark', label: 'Benchmark' },
		{ id: 'recorded-runs', label: 'Recorded runs' }
	];

	$effect(() => {
		let active = true;
		loading = true;
		data.settingsPromise.then((row: AdminSettings | null) => {
			if (!active) return;
			settings = row;
			loading = false;
			scrollToHash();
		});
		return () => {
			active = false;
		};
	});

	async function scrollToHash() {
		await tick();
		const target = window.location.hash ? document.getElementById(window.location.hash.slice(1)) : null;
		target?.scrollIntoView({ block: 'center' });
	}

	async function save(body: Record<string, unknown>, successMessage: string): Promise<boolean> {
		if (!settings) return false;
		saving = true;
		notice = null;
		try {
			const updated = await putSettings(data.adminKey, body);
			Object.assign(settings, updated);
			if (typeof body.required_game_version === 'string') {
				settings.required_game_version = body.required_game_version;
			}
			notice = { text: successMessage, error: false };
			return true;
		} catch (err) {
			notice = { text: `Could not save: ${err instanceof Error ? err.message : err}`, error: true };
			return false;
		} finally {
			saving = false;
		}
	}
</script>

<h1>Settings</h1>

{#if loading}
	<p>Loading settings...</p>
{:else if !settings}
	<div class="message message-error">Could not load settings from the server.</div>
{:else}
	<nav class="section-nav" aria-label="Settings sections">
		{#each sections as section (section.id)}
			<a href={`#${section.id}`}>{section.label}</a>
		{/each}
	</nav>

	{#if notice}
		<div class="message notice" class:message-error={notice.error} role="status">
			<span>{notice.text}</span>
			<button type="button" class="icon-btn" aria-label="Dismiss" onclick={() => (notice = null)}>✕</button>
		</div>
	{/if}

	<SettingsSection
		id="submissions"
		title="Submissions"
		description="Which game build may send entries to this server."
	>
		<GameVersionForm currentVersion={settings.required_game_version} {saving} onsave={save} />
	</SettingsSection>

	<SettingsSection
		id="levels"
		title="Levels"
		description="How each level is graded. A level that is off rejects new submissions; existing entries stay visible."
	>
		<LevelSettingsTable {settings} {saving} onsave={save} />
	</SettingsSection>

	<SettingsSection
		id="benchmark"
		title="Benchmark"
		description="Shared by every benchmarked level, so all entries are graded under identical conditions."
	>
		<div class="stat-grid">
			<div class="stat"><div class="label">Trials per entry</div><div>{settings.assault_trials}</div></div>
			<div class="stat"><div class="label">Sweep trials per size</div><div>{settings.assault_sweep_trials}</div></div>
			<div class="stat"><div class="label">Stream sweep</div><div>n ≤ {settings.stream_sweep_max}</div></div>
			<div class="stat"><div class="label">Assault sweep</div><div>n ≤ {settings.assault_sweep_max}</div></div>
			<div class="stat"><div class="label">Clean streak to stop</div><div>{settings.assault_clean_streak}</div></div>
			<div class="stat"><div class="label">Match time cap</div><div>{settings.match_cap_seconds} s</div></div>
		</div>
		<p class="meta">
			These counts are fixed in <code>server/app_settings.py</code>. The base seed below is the only
			shared value that can be changed here.
		</p>
		<BaseSeedForm seed={settings.seed} {saving} onsave={save} />
		<LegacyBenchmarkSettings {settings} {saving} onsave={save} />
	</SettingsSection>

	<SettingsSection
		id="recorded-runs"
		title="Recorded runs"
		description="Levels 6 to 8 are flown by the player, so nothing is simulated. These limits are checked on upload."
	>
		<RecordedRunLimits {settings} />
	</SettingsSection>
{/if}

<style>
	.section-nav {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		margin: -0.75rem 0 1.75rem 0;
	}

	.section-nav a {
		padding: 0.35rem 0.8rem;
		border: 1px solid var(--color-line);
		background: var(--color-surface);
		color: var(--color-dim);
		font-size: 0.8rem;
	}

	.section-nav a:hover {
		border-color: var(--color-line-strong);
		color: var(--color-heading);
	}

	.notice {
		position: sticky;
		top: 1rem;
		z-index: 5;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}
</style>
