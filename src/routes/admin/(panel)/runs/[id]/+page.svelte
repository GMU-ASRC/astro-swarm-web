<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiUrl } from '$lib/ts/api';

	let { data } = $props();
	let entry = $state<Record<string, any> | null>(null);
	let loading = $state(true);
	let message = $state('');

	$effect(() => {
		let active = true;
		loading = true;
		data.entryPromise.then((row) => {
			if (!active) return;
			entry = row;
			loading = false;
		});
		return () => {
			active = false;
		};
	});

	function formatSize(bytes: number): string {
		if (!bytes) return '—';
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	async function remove() {
		if (!confirm('Delete this simulator run? This cannot be undone.')) return;
		try {
			const res = await fetch(apiUrl(`/api/runs/${data.id}`), {
				method: 'DELETE',
				headers: { 'X-API-Key': data.adminKey }
			});
			if (res.ok || res.status === 204) {
				await goto('/admin/runs');
			} else {
				message = `Failed to delete: ${res.status}`;
			}
		} catch (err) {
			message = `Delete failed: ${err}`;
		}
	}
</script>

<p><a href="/admin/runs">← All simulator runs</a></p>

{#if message}<div class="message">{message}</div>{/if}

{#if loading}
	<p>Loading entry...</p>
{:else if !entry}
	<p>Entry not found.</p>
{:else}
	<h1>{entry.title}</h1>
	<p class="meta">by {entry.author} · uploaded {new Date(entry.created_at).toLocaleString()}</p>
	<p class="meta">{entry.id}</p>

	<div class="actions">
		<a class="admin-btn" href={apiUrl(`/api/runs/${entry.id}/export`)}>Download ZIP</a>
		<button class="admin-btn-danger" onclick={remove}>Delete</button>
	</div>

	{#if entry.video_filename}
		<p>
			<video controls style="width:100%;max-width:720px;background:#000">
				<source src={apiUrl(`/api/runs/${entry.id}/video`)} type="video/mp4" />
			</video>
		</p>
	{/if}

	{#if entry.description}
		<p>{entry.description}</p>
	{/if}

	<div class="stat-grid">
		<div class="stat"><div class="label">Duration</div><div>{entry.duration_seconds}s</div></div>
		<div class="stat"><div class="label">Robots</div><div>{entry.robot_count}</div></div>
		<div class="stat"><div class="label">Frames</div><div>{entry.frame_count}</div></div>
		<div class="stat"><div class="label">Arena</div><div>{entry.arena_width}×{entry.arena_height}</div></div>
		<div class="stat"><div class="label">File size</div><div>{formatSize(entry.file_size)}</div></div>
		<div class="stat"><div class="label">Downloads</div><div>{entry.download_count}</div></div>
	</div>

	<h2>Species</h2>
	{#if entry.species && entry.species.length > 0}
		<div class="actions">
			{#each entry.species as species}
				<span class="pill">{typeof species === 'string' ? species : species.name ?? JSON.stringify(species)}</span>
			{/each}
		</div>
	{:else}
		<p>No species data recorded for this run.</p>
	{/if}
{/if}
