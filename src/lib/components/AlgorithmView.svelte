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

	interface Def {
		label: string;
		category: 'config' | 'condition' | 'logic' | 'variable' | 'action';
		suffix?: string;
		step?: number;
	}

	const CAT: Record<string, { bg: string; dark: string }> = {
		config: { bg: '#46a952', dark: '#35843f' },
		condition: { bg: '#7b4daf', dark: '#5c378a' },
		logic: { bg: '#d98533', dark: '#ae6823' },
		variable: { bg: '#339e99', dark: '#22736f' },
		action: { bg: '#4176d7', dark: '#2e58ae' }
	};

	const DEFS: Record<string, Def> = {
		set_speed: { label: 'Set speed to', category: 'config', suffix: ' m/s', step: 0.05 },
		set_turn: { label: 'Set turn rate to', category: 'config', suffix: '°/s', step: 5 },
		set_view: { label: 'Set vision range to', category: 'config', suffix: ' m', step: 0.1 },
		set_fov: { label: 'Set FOV to', category: 'config', suffix: '°', step: 1 },
		set_size: { label: 'Set size to', category: 'config', suffix: ' px', step: 0.5 },

		when_start: { label: 'On start', category: 'condition' },
		when_always: { label: 'Always', category: 'condition' },
		when_sees: { label: 'When I see anyone', category: 'condition' },
		when_alone: { label: 'When I see nobody', category: 'condition' },
		when_near_wall: { label: 'When I touch a wall', category: 'condition' },
		when_sees_wall: { label: 'When I see a wall', category: 'condition' },
		when_sees_species: { label: 'When I see a', category: 'condition' },
		when_no_sees_species: { label: "When I don't see a", category: 'condition' },
		when_sees_enemy: { label: 'When I see an enemy', category: 'condition' },
		when_sees_ally: { label: 'When I see an ally', category: 'condition' },
		when_sees_object: { label: 'When I see an object', category: 'condition' },
		when_sees_rim: { label: 'When I see the outer rim', category: 'condition' },

		if_sees: { label: 'If I see anyone', category: 'logic' },
		if_sees_species: { label: 'If I see a', category: 'logic' },
		if_within: { label: 'If target within', category: 'logic', suffix: ' m', step: 0.1 },
		if_beyond: { label: 'If target beyond', category: 'logic', suffix: ' m', step: 0.1 },
		if_see: { label: 'If I see', category: 'logic' },
		if_compare: { label: 'If', category: 'logic' },
		else: { label: 'Else', category: 'logic' },

		set_var: { label: 'Set', category: 'variable' },
		set_var_random: { label: 'Set', category: 'variable' },

		do_forward: { label: 'Move forward', category: 'action' },
		do_backward: { label: 'Move backward', category: 'action' },
		do_stop: { label: 'Stop', category: 'action' },
		do_wander: { label: 'Wander randomly', category: 'action' },
		do_random_walk: { label: 'Random walk', category: 'action' },
		do_turn_left: { label: 'Turn left at', category: 'action', suffix: '°/s', step: 5 },
		do_turn_right: { label: 'Turn right at', category: 'action', suffix: '°/s', step: 5 },
		do_turn_left_by: { label: 'Turn left by', category: 'action', suffix: '°', step: 1 },
		do_turn_right_by: { label: 'Turn right by', category: 'action', suffix: '°', step: 1 },
		do_face: { label: 'Face the target', category: 'action' },
		do_flee: { label: 'Flee the target', category: 'action' },
		do_fire: { label: 'Fire', category: 'action' },
		do_throttle: { label: 'Throttle to', category: 'action', suffix: '×', step: 0.05 },
		do_stop_sim: { label: 'Stop simulation', category: 'action' },
		do_pause_sim: { label: 'Pause simulation', category: 'action' }
	};

	function defFor(type: string): Def {
		return DEFS[type] ?? { label: type.replace(/_/g, ' '), category: 'action' };
	}

	function isContainer(type: string): boolean {
		return type.startsWith('when_') || type.startsWith('if_') || type === 'else';
	}

	function valueText(block: Block, def: Def): string {
		const v = block.params?.value;
		if (v === undefined || v === null || v === '') return '';
		if (typeof v === 'number') {
			const decimals = (def.step ?? 1) >= 1 ? 0 : 1;
			return `${v.toFixed(decimals)}${def.suffix ?? ''}`;
		}
		return String(v);
	}

	let nonEmpty = $derived(scripts.filter((s) => (s.blocks ?? []).length > 0));
</script>

{#snippet node(block: Block)}
	{@const def = defFor(block.type)}
	{@const col = CAT[def.category]}
	{@const val = valueText(block, def)}
	<div class="rounded-md" style={`background:${col.bg};border-left:4px solid ${col.dark}`}>
		<div class="flex items-center gap-2 px-3 py-1.5">
			<span class="text-white text-[13px] leading-tight">{def.label}</span>
			{#if val}
				<span class="px-2 py-0.5 rounded text-white text-[12px] tabular-nums" style="background:rgba(0,0,0,0.24)">{val}</span>
			{/if}
		</div>
		{#if isContainer(block.type) && (block.children?.length ?? 0) > 0}
			<div class="ml-3 mr-2 mb-2 rounded p-2 flex flex-col gap-1.5" style="background:rgba(0,0,0,0.18)">
				{#each block.children ?? [] as child}
					{@render node(child)}
				{/each}
			</div>
		{/if}
	</div>
{/snippet}

{#if nonEmpty.length === 0}
	<p class="text-text-muted text-sm">No algorithm data available.</p>
{:else}
	<div class="flex flex-wrap gap-5 items-start">
		{#each nonEmpty as script}
			<div class="flex flex-col gap-2 min-w-[230px]">
				{#each script.blocks ?? [] as block}
					{@render node(block)}
				{/each}
			</div>
		{/each}
	</div>
{/if}
