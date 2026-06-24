<script lang="ts">
	interface Block {
		type: string;
		params?: Record<string, any>;
		children?: Block[];
	}
	interface Script {
		x?: number;
		y?: number;
		blocks?: Block[];
	}

	let { scripts = [] }: { scripts?: Script[] } = $props();

	function kind(type: string): 'event' | 'condition' | 'config' | 'action' {
		if (type.startsWith('when_')) return 'event';
		if (type.startsWith('if_') || type.startsWith('else')) return 'condition';
		if (type.startsWith('set_')) return 'config';
		return 'action';
	}

	const styles: Record<string, string> = {
		event: 'border-accent-cyan bg-accent-blue/10 text-accent-cyan',
		condition: 'border-purple-300 bg-purple-300/10 text-purple-200',
		config: 'border-amber-300 bg-amber-300/10 text-amber-200',
		action: 'border-star-white bg-white/5 text-star-white'
	};

	function label(block: Block): string {
		let name = block.type.replace(/^when_/, '').replace(/^if_/, '').replace(/^do_/, '').replace(/^set_/, '');
		name = name.replace(/_/g, ' ').toUpperCase();
		const value = block.params?.value;
		if (value !== undefined && value !== null && value !== '') {
			return `${name} ${value}`;
		}
		return name;
	}

	let nonEmpty = $derived(scripts.filter((s) => (s.blocks ?? []).length > 0));
</script>

{#if nonEmpty.length === 0}
	<p class="font-game text-text-muted text-sm">NO ALGORITHM DATA AVAILABLE.</p>
{:else}
	<div class="flex flex-wrap gap-6">
		{#each nonEmpty as script}
			<div class="flex flex-col gap-1.5 min-w-[180px]">
				{#each script.blocks ?? [] as block}
					<div class="px-3 py-2 border-l-4 font-game text-xs {styles[kind(block.type)]}">
						{label(block)}
					</div>
					{#each block.children ?? [] as child}
						<div class="px-3 py-2 border-l-4 ml-5 font-game text-xs {styles[kind(child.type)]}">
							{label(child)}
						</div>
					{/each}
				{/each}
			</div>
		{/each}
	</div>
{/if}
