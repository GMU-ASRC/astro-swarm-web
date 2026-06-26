<script lang="ts">
	import '$lib/css/simulator.css';
	import Icon from '@iconify/svelte';
	import { invalidateAll } from '$app/navigation';
	import { apiUrl } from '$lib/ts/api';

	let dragOver = $state(false);
	let selectedFile = $state<File | null>(null);
	let title = $state('');
	let description = $state('');
	let author = $state('');
	let submitting = $state(false);
	let uploadError = $state('');
	let uploadSuccess = $state(false);

	let fileInput = $state<HTMLInputElement>();

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		dragOver = false;
		const file = event.dataTransfer?.files[0];
		if (file) selectedFile = file;
	}

	function handleFileChange(event: Event) {
		const input = event.target as HTMLInputElement;
		selectedFile = input.files?.[0] ?? null;
		uploadError = '';
		uploadSuccess = false;
	}

	function getEndpoint(file: File): string | null {
		if (file.name.endsWith('.run')) return apiUrl('/api/runs');
		if (file.name.endsWith('.cfg')) return apiUrl('/api/configs');
		return null;
	}

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedFile || !title.trim()) return;

		const endpoint = getEndpoint(selectedFile);
		if (!endpoint) {
			uploadError = 'Unsupported file type. Only .cfg and .run files are accepted.';
			return;
		}

		submitting = true;
		uploadError = '';
		uploadSuccess = false;

		const formData = new FormData();
		formData.append('file', selectedFile);
		formData.append('title', title.trim());
		if (description.trim()) formData.append('description', description.trim());
		if (author.trim()) formData.append('author', author.trim());

		try {
			const res = await fetch(endpoint, { method: 'POST', body: formData });

			if (!res.ok) {
				const data = await res.json().catch(() => ({}));
				uploadError = data.error ?? `Upload failed (${res.status})`;
			} else {
				uploadSuccess = true;
				selectedFile = null;
				title = '';
				description = '';
				author = '';
				if (fileInput) fileInput.value = '';
				await invalidateAll();
			}
		} catch {
			uploadError = 'Could not reach the server. Check your connection and try again.';
		} finally {
			submitting = false;
		}
	}
</script>

<div class="bg-panel-bg/80 border border-accent-blue/25 p-8">
	<h2 class="font-sim text-lg font-semibold text-star-white mb-1">Share Your Config</h2>
	<p class="font-sim text-sm text-text-muted mb-7 leading-relaxed">
		Upload a <code class="text-accent-cyan">.cfg</code> or <code class="text-accent-cyan">.run</code>
		file exported from AstroSwarm to share it with the community.
	</p>

	{#if uploadSuccess}
		<div class="flex flex-col items-center gap-3 py-10 text-center">
			<Icon icon="ph:check-circle" width="40" color="var(--color-accent-cyan)" />
			<p class="font-sim text-sm text-star-white font-semibold m-0">Uploaded successfully!</p>
			<p class="font-sim text-xs text-text-muted m-0">Your file is now visible in the gallery.</p>
			<button
				onclick={() => (uploadSuccess = false)}
				class="font-sim text-xs text-accent-cyan mt-3 border-none bg-transparent cursor-pointer hover:text-star-white transition-colors"
			>
				Upload another
			</button>
		</div>
	{:else}
		<form onsubmit={handleSubmit}>
			<button
				type="button"
				class="upload-dropzone"
				class:drag-over={dragOver}
				ondragover={(e) => { e.preventDefault(); dragOver = true; }}
				ondragleave={() => (dragOver = false)}
				ondrop={handleDrop}
				onclick={() => fileInput?.click()}
			>
				<div class="flex justify-center mb-3">
					{#if selectedFile}
						<Icon icon="ph:check-circle" width="32" color="var(--color-accent-cyan)" />
					{:else}
						<Icon icon="ph:folder-open" width="32" color="var(--color-text-muted)" />
					{/if}
				</div>
				<p class="font-sim text-sm text-text-muted mb-1">
					{selectedFile ? selectedFile.name : 'Drop your file here or click to browse'}
				</p>
				<p class="font-sim text-xs text-text-muted/50">
					{selectedFile ? `${(selectedFile.size / 1024).toFixed(1)} KB` : 'Supports .cfg and .run files'}
				</p>
				<input bind:this={fileInput} type="file" accept=".cfg,.run" onchange={handleFileChange} />
			</button>

			<div class="mb-4">
				<label for="cfg-title" class="font-sim text-sm text-text-muted block mb-1">Title *</label>
				<input
					id="cfg-title"
					type="text"
					class="w-full bg-space-deep/80 border border-accent-blue/25 text-text-primary font-sim text-sm px-3 py-2 outline-none focus:border-accent-cyan focus:ring-2 focus:ring-accent-cyan/10 rounded-none"
					placeholder="e.g. Predator-prey swarm"
					bind:value={title}
					maxlength={80}
				/>
			</div>

			<div class="mb-4">
				<label for="cfg-author" class="font-sim text-sm text-text-muted block mb-1">Your name</label>
				<input
					id="cfg-author"
					type="text"
					class="w-full bg-space-deep/80 border border-accent-blue/25 text-text-primary font-sim text-sm px-3 py-2 outline-none focus:border-accent-cyan focus:ring-2 focus:ring-accent-cyan/10 rounded-none"
					placeholder="Anonymous"
					bind:value={author}
					maxlength={60}
				/>
			</div>

			<div class="mb-5">
				<label for="cfg-desc" class="font-sim text-sm text-text-muted block mb-1">Description</label>
				<textarea
					id="cfg-desc"
					class="w-full bg-space-deep/80 border border-accent-blue/25 text-text-primary font-sim text-sm px-3 py-2 outline-none focus:border-accent-cyan focus:ring-2 focus:ring-accent-cyan/10 resize-y min-h-[80px] rounded-none"
					placeholder="Describe what makes this config interesting…"
					bind:value={description}
					maxlength={400}
				></textarea>
			</div>

			{#if uploadError}
				<p class="font-sim text-xs text-red-400 mb-4 leading-snug">{uploadError}</p>
			{/if}

			<button
				type="submit"
				class="w-full font-sim text-sm font-semibold py-3 px-7 bg-accent-blue text-white border-none cursor-pointer transition-[background,box-shadow] duration-200 hover:bg-accent-cyan hover:text-space-deep hover:shadow-[0_0_20px_rgba(56,189,248,0.4)] disabled:opacity-40 disabled:cursor-not-allowed"
				disabled={!selectedFile || !title.trim() || submitting}
			>
				{submitting ? 'Uploading…' : 'Upload'}
			</button>
		</form>
	{/if}
</div>
