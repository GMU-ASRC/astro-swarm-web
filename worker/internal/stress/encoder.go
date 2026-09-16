package stress

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

const stderrTailLength = 600

type Encoder struct {
	command *exec.Cmd
	input   io.WriteCloser
	errors  bytes.Buffer
}

func StartEncoder(ffmpegPath, outputPath, codec string, width, height, fps int) (*Encoder, error) {
	resolved, err := exec.LookPath(ffmpegPath)
	if err != nil {
		return nil, fmt.Errorf("ffmpeg was not found at %q (install ffmpeg or pass -ffmpeg): %w", ffmpegPath, err)
	}

	arguments := []string{
		"-y", "-loglevel", "error",
		"-f", "rawvideo", "-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", width, height),
		"-r", strconv.Itoa(fps),
		"-i", "-",
		"-c:v", codec,
		"-pix_fmt", "yuv420p",
	}
	if codec == "libx264" {
		arguments = append(arguments, "-preset", "veryfast", "-crf", "23")
	}
	arguments = append(arguments, "-movflags", "+faststart", outputPath)

	encoder := &Encoder{command: exec.Command(resolved, arguments...)}
	encoder.command.Stderr = &encoder.errors
	encoder.input, err = encoder.command.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := encoder.command.Start(); err != nil {
		return nil, fmt.Errorf("starting ffmpeg: %w", err)
	}
	return encoder, nil
}

func (e *Encoder) WriteFrame(frame *image.RGBA) error {
	if _, err := e.input.Write(frame.Pix); err != nil {
		return fmt.Errorf("sending a frame to ffmpeg: %w%s", err, e.stderrTail())
	}
	return nil
}

func (e *Encoder) Close() error {
	closeErr := e.input.Close()
	if err := e.command.Wait(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w%s", err, e.stderrTail())
	}
	return closeErr
}

func (e *Encoder) stderrTail() string {
	message := strings.TrimSpace(e.errors.String())
	if message == "" {
		return ""
	}
	if len(message) > stderrTailLength {
		message = message[len(message)-stderrTailLength:]
	}
	return "\nffmpeg: " + message
}
