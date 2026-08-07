<script lang="ts">
	import Icon from '@iconify/svelte';

	interface Props {
		status: number;
		error: { message?: string };
	}

	let { status, error }: Props = $props();

	const presets: Record<number, { title: string; blurb: string; tone: string }> = {
		404: {
			title: 'Signal lost',
			blurb: 'This coordinate is empty space — no swarm node answered the ping.',
			tone: 'var(--color-brand)'
		},
		403: {
			title: 'Access denied',
			blurb: 'Your clearance does not authorize entry to this sector.',
			tone: 'var(--color-warn)'
		},
		500: {
			title: 'Swarm malfunction',
			blurb: 'A node faulted mid-transmission. Engineering has been pinged.',
			tone: 'var(--color-loss)'
		},
		503: {
			title: 'Swarm offline',
			blurb: 'The hive is temporarily unreachable. Try again shortly.',
			tone: 'var(--color-loss)'
		}
	};

	const preset = $derived(
		presets[status] ?? {
			title: 'Transmission failed',
			blurb: 'The swarm encountered an unexpected condition.',
			tone: 'var(--color-brand)'
		}
	);
</script>

<svelte:head>
	<title>Error {status} — AstroSwarm</title>
</svelte:head>

<div class="error-page" style={`--tone: ${preset.tone}`}>
	<div class="error-body">
		<p class="error-code">Error {status}</p>
		<h1 class="error-title">{preset.title}</h1>
		<p class="error-blurb">{preset.blurb}</p>

		{#if error?.message}
			<p class="error-detail">{error.message}</p>
		{/if}

		<div class="error-actions">
			<a href="/" class="btn btn-primary">
				<Icon icon="ph:house-bold" width="16" />
				Return home
			</a>
			<button class="btn btn-ghost" onclick={() => history.back()}>
				<Icon icon="ph:arrow-left-bold" width="16" />
				Go back
			</button>
		</div>
	</div>
</div>

<style>
	.error-page {
		position: relative;
		z-index: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
		padding: 6rem 2rem 4rem;
		text-align: center;
	}

	.error-body {
		max-width: 32rem;
	}

	.error-code {
		font-family: var(--font-display);
		font-size: 0.85rem;
		letter-spacing: 0.2em;
		text-transform: uppercase;
		color: var(--tone);
	}

	.error-title {
		margin-top: 1rem;
		font-family: var(--font-display);
		font-size: clamp(2.1rem, 5.6vw, 3.6rem);
		line-height: 1.15;
	}

	.error-blurb {
		margin-top: 1rem;
		font-size: 0.975rem;
		line-height: 1.65;
		color: var(--color-dim);
	}

	.error-detail {
		margin-top: 1.5rem;
		padding: 0.75rem 1rem;
		background: var(--color-surface);
		border: 1px solid var(--color-line);
		border-radius: 4px;
		font-family: ui-monospace, monospace;
		font-size: 0.78rem;
		color: var(--color-faint);
		overflow-wrap: anywhere;
	}

	.error-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
		justify-content: center;
		margin-top: 2.5rem;
	}
</style>
