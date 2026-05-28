import { readdirSync } from 'fs';
import { join } from 'path';

export const prerender = true;

export function load() {
	const dir = join(process.cwd(), 'static', 'previews');
	let images: string[] = [];

	try {
		images = readdirSync(dir)
			.filter((f) => /\.(png|jpg|jpeg|webp|gif)$/i.test(f))
			.sort();
	} catch {
		images = [];
	}

	return { images };
}
