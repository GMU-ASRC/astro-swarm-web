<script lang="ts">
	import { apiUrl } from '$lib/ts/api';
	import { whenVisible } from '$lib/ts/visibility';
	import { formatCount, formatWhen } from '$lib/ts/adminFormat';
	import type { AdminStats } from '$lib/ts/adminStats';
	import KpiGrid from '$lib/components/admin/dashboard/KpiGrid.svelte';
	import StatusBar from '$lib/components/admin/dashboard/StatusBar.svelte';
	import DailyChart from '$lib/components/admin/dashboard/DailyChart.svelte';
	import LevelTable from '$lib/components/admin/dashboard/LevelTable.svelte';
	import TimingTable from '$lib/components/admin/dashboard/TimingTable.svelte';
	import StorageSection from '$lib/components/admin/dashboard/StorageSection.svelte';
	import Skeleton from '$lib/components/skeleton/Skeleton.svelte';
	import SkeletonPanel from '$lib/components/skeleton/SkeletonPanel.svelte';

	const REFRESH_MS = 30000;

	let { data } = $props();
	let stats = $state<AdminStats | null>(null);
	let storage = $state<Record<string, any> | null>(null);
	let statsLoading = $state(true);
	let storageLoading = $state(true);
	let refreshing = $state(false);

	const sections = [
		{ href: '/admin/evaluations', label: 'Evaluations', text: 'Manage player level entries' },
		{ href: '/admin/players', label: 'Players', text: 'Profiles and their entries' },
		{ href: '/admin/leaderboard', label: 'Leaderboard', text: 'Success rate standings' },
		{ href: '/admin/runs', label: 'Simulator runs', text: 'Shared sandbox simulations' },
		{ href: '/admin/workers', label: 'Workers', text: 'Benchmark machines' },
		{ href: '/admin/settings', label: 'Settings', text: 'Seeds, levels and versions' }
	];

	$effect(() => {
		let active = true;
		data.statsPromise.then((row) => {
			if (!active) return;
			stats = row;
			statsLoading = false;
		});
		data.storagePromise.then((row) => {
			if (!active) return;
			storage = row;
			storageLoading = false;
		});
		const timer = setInterval(whenVisible(refreshStats), REFRESH_MS);
		return () => {
			active = false;
			clearInterval(timer);
		};
	});

	async function refreshStats() {
		try {
			const res = await fetch(apiUrl('/api/admin/stats'), { headers: { 'X-API-Key': data.adminKey } });
			if (res.ok) stats = await res.json();
		} catch {
			return;
		}
	}

	async function refreshAll() {
		refreshing = true;
		const storageRequest = fetch(apiUrl('/api/admin/storage'), { headers: { 'X-API-Key': data.adminKey } })
			.then((res) => (res.ok ? res.json() : storage))
			.catch(() => storage);
		await refreshStats();
		storage = await storageRequest;
		refreshing = false;
	}
</script>

<div class="dashboard-head">
	<div>
		<h1>Dashboard</h1>
		<p class="meta">
			{#if stats}Updated {formatWhen(stats.generated_at)} · refreshes every 30 seconds{:else}Overview of entries, workers, performance and storage{/if}
		</p>
	</div>
	<button class="admin-btn" onclick={refreshAll} disabled={refreshing}>{refreshing ? 'Refreshing...' : 'Refresh'}</button>
</div>

<nav class="section-links">
	{#each sections as section}
		<a href={section.href} class="section-link">
			<span class="section-label">{section.label}</span>
			<span class="section-text">{section.text}</span>
		</a>
	{/each}
</nav>

{#if statsLoading}
	<div class="kpi-skeleton">
		{#each Array(8) as _}
			<div class="stat"><Skeleton width="45%" height="0.65rem" /><div class="gap"><Skeleton width="60%" height="1.5rem" /></div></div>
		{/each}
	</div>
	<div class="panel"><SkeletonPanel lines={5} /></div>
{:else if !stats}
	<div class="message">Detailed stats are not available from this server version. They need the Go server.</div>
{:else}
	<KpiGrid {stats} totalBytes={storage?.total_bytes ?? null} />

	<div class="two-column">
		<section class="panel">
			<h2>Submissions</h2>
			<DailyChart days={stats.daily} />
		</section>
		<section class="panel">
			<h2>Entries by status</h2>
			<StatusBar counts={stats.entries.by_status} />
			<h2 class="sub-heading">Other content</h2>
			<dl class="content-counts">
				<div><dt>Simulator entries</dt><dd>{formatCount(stats.content.simulator_entries)}</dd></div>
				<div><dt>Survive matches</dt><dd>{formatCount(stats.content.survive_matches)}</dd></div>
				<div><dt>Training scores</dt><dd>{formatCount(stats.content.training_scores)}</dd></div>
				<div><dt>Sandbox runs</dt><dd>{formatCount(stats.content.sim_runs)}</dd></div>
				<div><dt>Legacy leaderboard</dt><dd>{formatCount(stats.content.leaderboard_entries)}</dd></div>
				<div><dt>Admin users</dt><dd>{formatCount(stats.content.admin_users)}</dd></div>
			</dl>
		</section>
	</div>

	<h2>Levels</h2>
	<LevelTable levels={stats.levels} />

	<h2>Page load times</h2>
	<p class="meta">Measured in visitors' browsers since the server started {formatWhen(stats.timings.since)}. Full load is a fresh visit; in-app is moving between pages.</p>
	<TimingTable rows={stats.timings.page_loads} label="Page" empty="No page loads reported yet." slowMs={3000} warnMs={1200} />

	<h2>Server response times</h2>
	<p class="meta">Time the server spends on each route, kept in memory for the last 500 requests per route.</p>
	<TimingTable rows={stats.timings.requests} label="Route" empty="No requests recorded yet." />
{/if}

<h2>Data storage</h2>
{#if storageLoading}
	<div class="panel"><SkeletonPanel lines={6} /></div>
{:else if !storage}
	<div class="message">Storage usage is unavailable.</div>
{:else}
	<StorageSection bind:storage adminKey={data.adminKey} />
{/if}

<style>
	.dashboard-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}

	.section-links {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
		gap: 0.6rem;
		margin: 1.25rem 0 1.5rem;
	}

	.section-link {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		padding: 0.75rem 0.9rem;
		border: 1px solid var(--color-line);
		background: var(--color-surface);
		text-decoration: none;
		transition:
			border-color 0.15s,
			background 0.15s;
	}

	.section-link:hover {
		border-color: var(--color-brand);
		background: var(--color-surface-raised);
	}

	.section-label {
		color: var(--color-heading);
		font-weight: 600;
		font-size: 0.9rem;
	}

	.section-text {
		color: var(--color-faint);
		font-size: 0.75rem;
	}

	.kpi-skeleton {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
		gap: 0.75rem;
		margin-bottom: 1rem;
	}

	.gap {
		margin-top: 0.6rem;
	}

	.two-column {
		display: grid;
		grid-template-columns: minmax(0, 3fr) minmax(0, 2fr);
		gap: 1rem;
		margin-top: 1rem;
	}

	.panel {
		padding: 1.1rem 1.2rem;
		border: 1px solid var(--color-line);
		background: var(--color-surface);
	}

	.panel h2 {
		margin-top: 0;
	}

	.sub-heading {
		margin-top: 1.4rem !important;
	}

	.content-counts {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.5rem 1rem;
		margin: 0;
	}

	.content-counts div {
		display: flex;
		justify-content: space-between;
		gap: 0.5rem;
		padding-bottom: 0.4rem;
		border-bottom: 1px solid var(--color-line);
		font-size: 0.82rem;
	}

	.content-counts dt {
		color: var(--color-dim);
	}

	.content-counts dd {
		margin: 0;
		color: var(--color-heading);
		font-variant-numeric: tabular-nums;
	}

	@media (max-width: 960px) {
		.two-column {
			grid-template-columns: minmax(0, 1fr);
		}
	}
</style>
