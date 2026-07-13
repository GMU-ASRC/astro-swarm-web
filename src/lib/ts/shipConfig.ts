export interface ShipConfig {
	speed: number;
	turn: number;
	view: number;
	fov: number;
}

export const DEFENDER_DEFAULTS: ShipConfig = {
	speed: 3.75,
	turn: 183.3,
	view: 7.5,
	fov: 70
};

export const EVADER_CONFIG: ShipConfig = {
	speed: 2.63,
	turn: 183.3,
	view: 7.5,
	fov: 70
};

export const PILOT_EVADER_CONFIG: ShipConfig = {
	speed: 3.75,
	turn: 160.4,
	view: 7.5,
	fov: 70
};

function collect(blocks: any[], config: ShipConfig) {
	for (const block of blocks ?? []) {
		const params = block?.params ?? {};
		switch (block?.type) {
			case 'set_speed':
				config.speed = Number(params.value ?? config.speed);
				break;
			case 'set_turn':
				config.turn = Number(params.value ?? config.turn);
				break;
			case 'set_view':
				config.view = Number(params.value ?? config.view);
				break;
			case 'set_fov':
				config.fov = Math.min(180, Number(params.value ?? config.fov));
				break;
		}
		if (block?.children) collect(block.children, config);
	}
}

export function defenderConfig(scripts: any[]): ShipConfig {
	const config = { ...DEFENDER_DEFAULTS };
	for (const script of scripts ?? []) collect(script?.blocks ?? [], config);
	return config;
}

export function configRows(config: ShipConfig): { label: string; value: string }[] {
	return [
		{ label: 'Speed', value: `${config.speed.toFixed(2)} m/s` },
		{ label: 'Turn rate', value: `${Math.round(config.turn)}°/s` },
		{ label: 'Vision range', value: `${config.view.toFixed(2)} m` },
		{ label: 'FOV', value: `${Math.round(config.fov)}°` }
	];
}
