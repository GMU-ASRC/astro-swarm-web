export type ShipVariant = 'blue' | 'red' | 'purple' | 'gold' | 'green';

interface ShipPalette {
	outline: string;
	shadow: string;
	base: string;
	light: string;
	stripe: string;
	highlight: string;
}

const PALETTES: Record<ShipVariant, ShipPalette> = {
	blue: {
		outline: '#16283a',
		shadow: '#23415f',
		base: '#35618f',
		light: '#5b8fc4',
		stripe: '#7fb0e0',
		highlight: '#a8cdef'
	},
	red: {
		outline: '#3a1a1a',
		shadow: '#5f2a2a',
		base: '#8f3f3f',
		light: '#c46868',
		stripe: '#e08f8f',
		highlight: '#f0b3b3'
	},
	purple: {
		outline: '#2a1a3a',
		shadow: '#472a5f',
		base: '#6b3f8f',
		light: '#9868c4',
		stripe: '#b98fe0',
		highlight: '#d0b3ec'
	},
	gold: {
		outline: '#3a3016',
		shadow: '#5f5023',
		base: '#8f7a35',
		light: '#c4ab5b',
		stripe: '#e0c67f',
		highlight: '#efdca8'
	},
	green: {
		outline: '#1a3a21',
		shadow: '#2a5f34',
		base: '#3f8f4f',
		light: '#68c479',
		stripe: '#8fe0a0',
		highlight: '#b3f0c0'
	}
};

const HULL_OUTLINE = 'M 125 16 L 219 172 L 188 219 L 62 219 L 31 172 Z';

export function shipSvgMarkup(variant: ShipVariant, clipId: string, size = '100%'): string {
	const paint = PALETTES[variant];
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 250 250" width="${size}" height="${size}">
	<defs>
		<clipPath id="${clipId}"><path d="${HULL_OUTLINE}" /></clipPath>
	</defs>
	<g clip-path="url(#${clipId})">
		<rect x="0" y="0" width="250" height="250" fill="${paint.base}" />
		<rect x="0" y="0" width="125" height="250" fill="${paint.shadow}" />
		<rect x="125" y="0" width="125" height="250" fill="${paint.light}" />
		<rect x="109" y="16" width="32" height="203" fill="${paint.stripe}" />
		<rect x="117" y="16" width="16" height="12" fill="${paint.highlight}" />
		<rect x="98" y="60" width="54" height="4" fill="${paint.outline}" />
		<rect x="77" y="95" width="96" height="4" fill="${paint.outline}" />
		<rect x="59" y="125" width="132" height="4" fill="${paint.outline}" />
		<rect x="41" y="155" width="168" height="4" fill="${paint.outline}" />
		<line x1="120" y1="19" x2="214" y2="175" stroke="${paint.highlight}" stroke-width="3" stroke-linecap="round" />
		<line x1="130" y1="19" x2="36" y2="175" stroke="${paint.outline}" stroke-width="3" stroke-linecap="round" />
		<line x1="214" y1="169" x2="183" y2="216" stroke="${paint.highlight}" stroke-width="3" stroke-linecap="round" />
		<line x1="36" y1="169" x2="67" y2="216" stroke="${paint.outline}" stroke-width="3" stroke-linecap="round" />
		<path fill="${paint.outline}" d="M 31 188 h 16 v 16 h -16 Z M 203 188 h 16 v 16 h -16 Z M 62 203 h 126 v 16 h -126 Z" />
		<rect x="66" y="210" width="8" height="6" fill="#000000" />
		<rect x="176" y="210" width="8" height="6" fill="#000000" />
		<rect x="76" y="205" width="12" height="2" fill="${paint.highlight}" />
		<rect x="162" y="205" width="12" height="2" fill="${paint.highlight}" />
		<rect x="76" y="207" width="12" height="8" fill="#000000" />
		<rect x="162" y="207" width="12" height="8" fill="#000000" />
	</g>
</svg>`;
}

export function shipSpriteUrl(variant: ShipVariant, pixels: number): string {
	const markup = shipSvgMarkup(variant, `sprite-${variant}`, String(pixels));
	return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(markup)}`;
}

export function shipAccent(variant: ShipVariant): string {
	return PALETTES[variant].stripe;
}
