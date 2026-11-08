import { apiUrl } from '$lib/ts/api';

const MAGIC = 'ASR1';
const INTEGER_KIND = 0;
const FLOAT_KIND = 1;

class ByteCursor {
	private view: DataView;
	private bytes: Uint8Array;
	offset = 0;

	constructor(buffer: ArrayBuffer) {
		this.view = new DataView(buffer);
		this.bytes = new Uint8Array(buffer);
	}

	get remaining(): number {
		return this.bytes.length - this.offset;
	}

	byte(): number {
		if (this.offset >= this.bytes.length) throw new Error('Replay frames end early');
		return this.bytes[this.offset++];
	}

	uvarint(): number {
		let value = 0;
		let scale = 1;
		for (let shift = 0; shift < 70; shift += 7) {
			const next = this.byte();
			value += (next & 0x7f) * scale;
			if (next < 0x80) return value;
			scale *= 128;
		}
		throw new Error('Replay frames have a bad number');
	}

	varint(): number {
		const zigzag = this.uvarint();
		return zigzag % 2 === 0 ? zigzag / 2 : -(zigzag + 1) / 2;
	}

	float64(): number {
		if (this.remaining < 8) throw new Error('Replay frames end early');
		const value = this.view.getFloat64(this.offset, true);
		this.offset += 8;
		return value;
	}
}

async function inflate(compressed: ArrayBuffer): Promise<ArrayBuffer> {
	const stream = new Blob([compressed]).stream().pipeThrough(new DecompressionStream('deflate'));
	return await new Response(stream).arrayBuffer();
}

export async function decodeReplayFrames(compressed: ArrayBuffer): Promise<number[][]> {
	const cursor = new ByteCursor(await inflate(compressed));
	const magic = String.fromCharCode(cursor.byte(), cursor.byte(), cursor.byte(), cursor.byte());
	if (magic !== MAGIC) throw new Error('Not an AstroSwarm replay file');
	const kind = cursor.byte();
	const frameCount = cursor.uvarint();
	const frames: number[][] = new Array(frameCount);
	let previous: number[] = [];
	for (let frameIndex = 0; frameIndex < frameCount; frameIndex++) {
		const length = cursor.uvarint();
		const frame: number[] = new Array(length);
		for (let index = 0; index < length; index++) {
			if (kind === INTEGER_KIND) {
				const value = cursor.varint();
				frame[index] = index < previous.length ? previous[index] + value : value;
			} else if (kind === FLOAT_KIND) {
				frame[index] = cursor.float64();
			} else {
				throw new Error('Unknown replay frame kind');
			}
		}
		frames[frameIndex] = frame;
		previous = frame;
	}
	return frames;
}

function withBinaryFrames(path: string): string {
	return `${path}${path.includes('?') ? '&' : '?'}frames=binary`;
}

export async function fetchReplay<Replay extends { frames?: unknown[]; frames_url?: string }>(path: string): Promise<Replay | null> {
	const res = await fetch(apiUrl(withBinaryFrames(path)));
	if (!res.ok) return null;
	const replay = (await res.json()) as Replay;
	if (replay.frames || !replay.frames_url || typeof DecompressionStream === 'undefined') return replay;
	const framesRes = await fetch(apiUrl(replay.frames_url));
	if (!framesRes.ok) return { ...replay, frames: [] } as Replay;
	return { ...replay, frames: await decodeReplayFrames(await framesRes.arrayBuffer()) } as Replay;
}
