import {
	BufferTarget,
	CanvasSource,
	Mp4OutputFormat,
	Output,
	QUALITY_HIGH,
	canEncodeVideo
} from 'mediabunny';
import type { Replay } from '$lib/ts/evaluation';
import {
	STAGE_HEIGHT,
	STAGE_WIDTH,
	drawStage,
	sampleFrame,
	type ReplayMode
} from '$lib/ts/replayRenderer';

const VIDEO_SCALE = 2;
const VIDEO_FPS = 30;
const VIDEO_WIDTH = STAGE_WIDTH * VIDEO_SCALE;
const VIDEO_HEIGHT = STAGE_HEIGHT * VIDEO_SCALE;

export async function canExportVideo(): Promise<boolean> {
	if (typeof VideoEncoder === 'undefined') return false;
	return canEncodeVideo('avc', { width: VIDEO_WIDTH, height: VIDEO_HEIGHT });
}

export function videoFileName(replay: Replay): string {
	const parts = ['replay'];
	if (replay.n != null) parts.push(`n${replay.n}`);
	if (replay.trial != null) parts.push(`trial${replay.trial}`);
	parts.push(replay.outcome || 'run');
	return `${parts.join('-')}.mp4`;
}

export async function renderReplayVideo(
	replay: Replay,
	mode: ReplayMode,
	onProgress: (fraction: number) => void
): Promise<Blob> {
	const canvas = document.createElement('canvas');
	canvas.width = VIDEO_WIDTH;
	canvas.height = VIDEO_HEIGHT;
	const context = canvas.getContext('2d');
	if (!context) throw new Error('Canvas rendering is not available.');

	const target = new BufferTarget();
	const output = new Output({ format: new Mp4OutputFormat({ fastStart: 'in-memory' }), target });
	const videoSource = new CanvasSource(canvas, { codec: 'avc', bitrate: QUALITY_HIGH });
	output.addVideoTrack(videoSource, { frameRate: VIDEO_FPS });
	await output.start();

	const replayFps = replay.fps || 1;
	const lastReplayFrame = Math.max(0, replay.frames.length - 1);
	const durationSeconds = lastReplayFrame / replayFps;
	const videoFrameCount = Math.max(1, Math.floor(durationSeconds * VIDEO_FPS) + 1);
	const videoFrameDuration = 1 / VIDEO_FPS;

	try {
		for (let videoFrame = 0; videoFrame < videoFrameCount; videoFrame++) {
			const replayPosition = Math.min(lastReplayFrame, (videoFrame / VIDEO_FPS) * replayFps);
			const frameIndex = Math.floor(replayPosition);
			const blend = replayPosition - frameIndex;
			context.setTransform(VIDEO_SCALE, 0, 0, VIDEO_SCALE, 0, 0);
			drawStage(context, replay, mode, sampleFrame(replay, frameIndex, blend), {
				frameIndex,
				selectedSlot: null
			});
			await videoSource.add(videoFrame * videoFrameDuration, videoFrameDuration);
			onProgress((videoFrame + 1) / videoFrameCount);
		}
		await output.finalize();
	} catch (error) {
		await output.cancel();
		throw error;
	}

	if (!target.buffer) throw new Error('The video encoder produced no data.');
	return new Blob([target.buffer], { type: 'video/mp4' });
}

export function saveBlob(blob: Blob, fileName: string) {
	const url = URL.createObjectURL(blob);
	const link = document.createElement('a');
	link.href = url;
	link.download = fileName;
	document.body.appendChild(link);
	link.click();
	link.remove();
	setTimeout(() => URL.revokeObjectURL(url), 1000);
}
