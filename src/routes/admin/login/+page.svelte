<script lang="ts">
	import { enhance } from '$app/forms';

	let { form } = $props();
	let loading = $state(false);
</script>

<svelte:head>
	<title>Admin Login — AstroSwarm</title>
</svelte:head>

<div class="login-wrapper">
	<div class="login-card">
		<h1 class="login-title">ASTROSWARM</h1>
		<p class="login-subtitle">SECURE ADMIN UPLINK</p>

		<form method="POST" use:enhance={() => { loading = true; return async ({ update }) => { loading = false; update(); } }} class="login-form">
			<div class="input-group">
				<label for="apiKey" class="login-label">MASTER KEY</label>
				<input
					type="password"
					id="apiKey"
					name="apiKey"
					placeholder="••••••••••••••••"
					required
					class="login-input"
				/>
			</div>

			{#if form?.error}
				<div class="login-error">
					{form.error}
				</div>
			{/if}

			<button
				type="submit"
				disabled={loading}
				class="login-button"
			>
				{loading ? 'AUTHORIZING...' : 'ACCESS PANEL'}
			</button>
		</form>
	</div>
</div>

<style>
	.login-wrapper {
		min-height: calc(100vh - 80px);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		font-family: var(--font-sim);
	}

	.login-card {
		width: 100%;
		max-width: 400px;
		background-color: rgba(10, 21, 38, 0.85);
		backdrop-filter: blur(12px);
		border: 1px solid rgba(56, 189, 248, 0.3);
		border-radius: 12px;
		padding: 2.5rem;
		box-shadow: 0 0 30px rgba(56, 189, 248, 0.1);
	}

	.login-title {
		font-family: var(--font-game);
		color: #ffffff;
		font-size: 2rem;
		text-align: center;
		margin: 0 0 0.5rem 0;
		text-shadow: 0 0 15px rgba(56, 189, 248, 0.5);
		letter-spacing: 0.1em;
	}

	.login-subtitle {
		color: rgba(139, 163, 201, 0.8);
		font-size: 0.85rem;
		text-align: center;
		margin: 0 0 2rem 0;
		letter-spacing: 0.05em;
	}

	.login-form {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.input-group {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.login-label {
		font-family: var(--font-game);
		color: #38bdf8;
		font-size: 0.75rem;
		letter-spacing: 0.1em;
	}

	.login-input {
		width: 100%;
		background-color: rgba(2, 4, 9, 0.6);
		border: 1px solid rgba(56, 189, 248, 0.3);
		color: #ffffff;
		font-family: var(--font-game);
		font-size: 0.875rem;
		padding: 0.875rem 1rem;
		border-radius: 6px;
		outline: none;
		transition: all 0.2s ease;
		box-sizing: border-box;
	}

	.login-input:focus {
		border-color: #38bdf8;
		background-color: rgba(56, 189, 248, 0.05);
		box-shadow: 0 0 0 2px rgba(56, 189, 248, 0.2);
	}

	.login-input::placeholder {
		color: rgba(139, 163, 201, 0.4);
	}

	.login-error {
		background-color: rgba(239, 68, 68, 0.1);
		border: 1px solid rgba(239, 68, 68, 0.3);
		color: #fca5a5;
		padding: 0.75rem;
		border-radius: 6px;
		font-family: var(--font-game);
		font-size: 0.75rem;
		text-align: center;
		letter-spacing: 0.05em;
	}

	.login-button {
		width: 100%;
		background-color: rgba(56, 189, 248, 0.15);
		border: 1px solid #38bdf8;
		color: #e0f2fe;
		font-family: var(--font-game);
		font-size: 0.875rem;
		padding: 1rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.2s ease;
		letter-spacing: 0.1em;
		outline: none;
		margin-top: 0.5rem;
	}

	.login-button:hover:not(:disabled) {
		background-color: rgba(56, 189, 248, 0.25);
		box-shadow: 0 0 15px rgba(56, 189, 248, 0.3);
	}

	.login-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
