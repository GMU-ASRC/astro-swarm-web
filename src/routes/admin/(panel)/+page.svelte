<script lang="ts">
</script>

<div class="max-w-4xl">
	<h1 class="font-game text-3xl text-star-white mb-4" style="text-shadow: 0 0 20px rgba(56,189,248,0.4)">
		Welcome, Admin
	</h1>
	<p class="text-text-muted leading-relaxed mb-8">
		Select a category from the sidebar to manage data, moderate content, and delete old evaluations or simulator runs.
	</p>

	<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
		<a href="/admin/evaluations" class="block p-6 border-2 border-sky-500/20 bg-sky-500/5 hover:border-sky-400/50 hover:bg-sky-500/10 transition-colors">
			<h2 class="font-game text-xl text-sky-200 mb-2">Evaluations</h2>
			<p class="text-xs text-text-muted">Manage player level evaluations.</p>
		</a>
		<a href="/admin/leaderboard" class="block p-6 border-2 border-sky-500/20 bg-sky-500/5 hover:border-sky-400/50 hover:bg-sky-500/10 transition-colors">
			<h2 class="font-game text-xl text-sky-200 mb-2">Leaderboard</h2>
			<p class="text-xs text-text-muted">Manage leaderboard entries.</p>
		</a>
		<a href="/admin/runs" class="block p-6 border-2 border-sky-500/20 bg-sky-500/5 hover:border-sky-400/50 hover:bg-sky-500/10 transition-colors">
			<h2 class="font-game text-xl text-sky-200 mb-2">Simulator Runs</h2>
			<p class="text-xs text-text-muted">Manage shared sandbox simulations.</p>
		</a>
	</div>
</div>
