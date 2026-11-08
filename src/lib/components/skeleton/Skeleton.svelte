<script lang="ts">
	let {
		width = '100%',
		height = '0.9rem',
		radius = '4px',
		class: extraClass = ''
	}: { width?: string; height?: string; radius?: string; class?: string } = $props();
</script>

<div class="skeleton {extraClass}" style:width style:height style:border-radius={radius} aria-hidden="true"></div>
