<script lang="ts">
	import Icon from '@iconify/svelte';

	let {
		title,
		message,
		warning = '',
		confirmLabel = 'Confirm',
		danger = false,
		onconfirm,
		oncancel
	}: {
		title: string;
		message: string;
		warning?: string;
		confirmLabel?: string;
		danger?: boolean;
		onconfirm: () => void;
		oncancel: () => void;
	} = $props();

	let backdrop: HTMLDivElement | undefined = $state();

	$effect(() => {
		backdrop?.focus();
	});
</script>

<div
	class="backdrop"
	role="alertdialog"
	aria-modal="true"
	aria-label={title}
	tabindex="-1"
	bind:this={backdrop}
	onclick={(event) => event.target === event.currentTarget && oncancel()}
	onkeydown={(event) => event.key === 'Escape' && oncancel()}
>
	<div class="panel">
		<h2>
			{#if danger}<Icon icon="ph:warning-bold" width="18" />{/if}
			{title}
		</h2>
		<p>{message}</p>
		{#if warning}
			<p class="warning"><Icon icon="ph:warning-circle-bold" width="16" />{warning}</p>
		{/if}
		<div class="buttons">
			<button type="button" onclick={oncancel}>Cancel</button>
			<button type="button" class={danger ? 'admin-btn-danger' : ''} onclick={onconfirm}>
				{confirmLabel}
			</button>
		</div>
	</div>
</div>

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 60;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
		background: rgba(0, 0, 0, 0.6);
	}

	.panel {
		width: min(30rem, 100%);
		padding: 1.4rem 1.5rem;
		background: var(--color-surface);
		border: 1px solid var(--color-line-strong);
		outline: none;
	}

	.panel h2 {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin: 0 0 0.75rem 0;
		font-size: 1.05rem;
		font-weight: 600;
		color: var(--color-heading);
	}

	.panel p {
		margin: 0 0 0.75rem 0;
		color: var(--color-dim);
	}

	.warning {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.6rem 0.75rem;
		background: color-mix(in srgb, var(--color-warn) 10%, transparent);
		border-left: 2px solid var(--color-warn);
		color: var(--color-warn);
		font-size: 0.85rem;
	}

	.buttons {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		margin-top: 1.25rem;
	}
</style>
