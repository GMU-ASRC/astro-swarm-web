<script lang="ts">
	import Icon from '@iconify/svelte';

	interface LevelStat {
		level_number: number;
		success_rate: number | null;
		xp: number;
		rank: number | null;
		players: number;
	}

	interface EntryRow {
		id: string;
		level_id: string;
		level_number?: number;
		is_attack?: boolean;
		status: string;
		success_rate: number | null;
		xp_awarded: number | null;
		game_version?: string;
		created_at: string;
	}

	interface Profile {
		player_id: string;
		username: string;
		total_xp: number;
		overall_success: number | null;
		best_success: number | null;
		overall_rank: number | null;
		total_players: number;
		entries: number;
		levels: LevelStat[];
		recent_entries: EntryRow[];
	}

	interface PageData {
		profile: Profile;
	}

	let { data }: { data: PageData } = $props();
	let profile = $derived(data.profile);

	function when(iso: string): string {
		const date = new Date(iso);
		const month = String(date.getMonth() + 1).padStart(2, '0');
		const day = String(date.getDate()).padStart(2, '0');
		return `${month}/${day}/${date.getFullYear()}`;
	}

	function levelName(levelNumber: number): string {
		return levelNumber >= 3 ? `Level ${levelNumber} · Evasion` : `Level ${levelNumber} · Defense`;
	}

	function statusClass(status: string): string {
		if (status === 'done') return 'badge-win';
		if (status === 'running' || status === 'queued') return 'badge-brand';
		return 'badge-loss';
	}
</script>

<svelte:head>
	<title>{profile.username} — AstroSwarm</title>
</svelte:head>

<div class="page">
	<div class="shell page-head">
		<a href="/leaderboard" class="back-link">
			<Icon icon="ph:arrow-left-bold" width="14" />
			Leaderboard
		</a>
		<h1 class="page-title name-heading">{profile.username}</h1>
		<p class="page-lede">
			Rank #{profile.overall_rank ?? '—'} of {profile.total_players} commanders ·
			{profile.overall_success != null ? `${profile.overall_success}% average success` : 'no results yet'}
		</p>
	</div>

	<div class="shell profile">
		<div class="stat-grid">
			<div class="stat">
				<div class="stat-value">
					{profile.overall_success != null ? `${profile.overall_success}%` : '—'}
				</div>
				<div class="stat-label">Average success</div>
			</div>
			<div class="stat">
				<div class="stat-value">
					{profile.best_success != null ? `${profile.best_success}%` : '—'}
				</div>
				<div class="stat-label">Best success</div>
			</div>
			<div class="stat">
				<div class="stat-value">#{profile.overall_rank ?? '—'}</div>
				<div class="stat-label">Overall rank</div>
			</div>
			<div class="stat">
				<div class="stat-value">{profile.entries}</div>
				<div class="stat-label">Entries</div>
			</div>
		</div>

		<h2 class="section-title block-heading">Per-level stats</h2>
		{#if profile.levels.length === 0}
			<div class="notice">No level stats yet.</div>
		{:else}
			<div class="level-grid">
				{#each profile.levels as level}
					<div class="card level">
						<div>
							<div class="level-name">{levelName(level.level_number)}</div>
							<div class="level-detail">
								{level.success_rate != null
									? `${level.success_rate}% ${level.level_number >= 3 ? 'evasion' : 'detection'}`
									: 'no data'} · {level.xp} XP
							</div>
						</div>
						<div class="level-rank">
							<div class="level-rank-value">#{level.rank ?? '—'}</div>
							<div class="level-detail">of {level.players}</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}

		<h2 class="section-title block-heading">Entries</h2>
		{#if profile.recent_entries.length === 0}
			<div class="notice">No entries.</div>
		{:else}
			<div class="entry-list">
				{#each profile.recent_entries as entry}
					<a href={`/levels/${entry.id}`} class="card-link entry">
						<span class="entry-level">
							{levelName(entry.level_number ?? 1).split(' · ')[0]}
						</span>
						<span class="entry-result">
							{entry.success_rate != null
								? `${entry.success_rate}% ${entry.is_attack ? 'evasion' : 'detection'}`
								: 'pending'} · {entry.xp_awarded != null ? `${entry.xp_awarded} XP` : 'unclaimed'}
						</span>
						<span class="entry-version">{entry.game_version ?? 'v0.0.4'}</span>
						<span class="badge {statusClass(entry.status)}">{entry.status}</span>
						<span class="entry-date">{when(entry.created_at)}</span>
					</a>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.name-heading {
		margin-top: 1.25rem;
	}

	.profile {
		padding-bottom: 6rem;
	}

	.stat-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
		gap: 0.75rem;
	}

	.block-heading {
		margin-top: 3rem;
		margin-bottom: 1rem;
	}

	.level-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
		gap: 0.75rem;
	}

	.level {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 1.1rem 1.25rem;
	}

	.level-name {
		font-weight: 600;
		color: var(--color-heading);
	}

	.level-detail {
		margin-top: 0.25rem;
		font-size: 0.78rem;
		color: var(--color-faint);
	}

	.level-rank {
		text-align: right;
		flex-shrink: 0;
	}

	.level-rank-value {
		font-size: 1.2rem;
		font-weight: 600;
		color: var(--color-brand-hover);
	}

	.entry-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.entry {
		display: flex;
		align-items: center;
		gap: 1.25rem;
		padding: 0.9rem 1.25rem;
		font-size: 0.85rem;
	}

	.entry-level {
		width: 5rem;
		flex-shrink: 0;
		font-weight: 600;
		color: var(--color-heading);
	}

	.entry-result {
		flex: 1;
		min-width: 0;
		color: var(--color-dim);
	}

	.entry-version,
	.entry-date {
		flex-shrink: 0;
		font-size: 0.75rem;
		color: var(--color-faint);
	}

	@media (max-width: 720px) {
		.entry {
			flex-wrap: wrap;
			gap: 0.6rem 1rem;
		}

		.entry-result {
			flex-basis: 100%;
		}
	}
</style>
