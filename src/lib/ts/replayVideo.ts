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

export const VIDEO_FPS = 30;
const FARP_VIDEO_SCALE = 2;

export interface VideoJob {
	width: number;
	height: number;
	frameCount: number;
	drawFrame: (context: CanvasRenderingContext2D, videoFrame: number) => void;
}

export async function canExportVideo(width: number, height: number): Promise<boolean> {
	if (typeof VideoEncoder === 'undefined') return false;
	return canEncodeVideo('avc', { width, height });
}

export function videoFrameCount(durationSeconds: number): number {
	return Math.max(1, Math.floor(durationSeconds * VIDEO_FPS) + 1);
}

export function evenSize(value: number): number {
	const rounded = Math.max(2, Math.round(value));
	return rounded % 2 === 0 ? rounded : rounded + 1;
}

export async function encodeVideo(job: VideoJob, onProgress: (fraction: number) => void): Promise<Blob> {
	const canvas = document.createElement('canvas');
	canvas.width = job.width;
	canvas.height = job.height;
	const context = canvas.getContext('2d');
	if (!context) throw new Error('Canvas rendering is not available.');

	const target = new BufferTarget();
	const output = new Output({ format: new Mp4OutputFormat({ fastStart: 'in-memory' }), target });
	const videoSource = new CanvasSource(canvas, { codec: 'avc', bitrate: QUALITY_HIGH });
	output.addVideoTrack(videoSource, { frameRate: VIDEO_FPS });
	await output.start();

	const videoFrameDuration = 1 / VIDEO_FPS;
	try {
		for (let videoFrame = 0; videoFrame < job.frameCount; videoFrame++) {
			context.setTransform(1, 0, 0, 1, 0, 0);
			job.drawFrame(context, videoFrame);
			await videoSource.add(videoFrame * videoFrameDuration, videoFrameDuration);
			onProgress((videoFrame + 1) / job.frameCount);
		}
		await output.finalize();
	} catch (error) {
		await output.cancel();
		throw error;
	}

	if (!target.buffer) throw new Error('The video encoder produced no data.');
	return new Blob([target.buffer], { type: 'video/mp4' });
}

export function farpVideoJob(replay: Replay, mode: ReplayMode): VideoJob {
	const replayFps = replay.fps || 1;
	const lastReplayFrame = Math.max(0, replay.frames.length - 1);
	return {
		width: STAGE_WIDTH * FARP_VIDEO_SCALE,
		height: STAGE_HEIGHT * FARP_VIDEO_SCALE,
		frameCount: videoFrameCount(lastReplayFrame / replayFps),
		drawFrame: (context, videoFrame) => {
			const replayPosition = Math.min(lastReplayFrame, (videoFrame / VIDEO_FPS) * replayFps);
			const frameIndex = Math.floor(replayPosition);
			context.setTransform(FARP_VIDEO_SCALE, 0, 0, FARP_VIDEO_SCALE, 0, 0);
			drawStage(context, replay, mode, sampleFrame(replay, frameIndex, replayPosition - frameIndex), {
				frameIndex,
				selectedSlot: null
			});
		}
	};
}

export function farpVideoFileName(replay: Replay): string {
	const parts = ['replay'];
	if (replay.n != null) parts.push(`n${replay.n}`);
	if (replay.trial != null) parts.push(`trial${replay.trial}`);
	parts.push(replay.outcome || 'run');
	return `${parts.join('-')}.mp4`;
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
