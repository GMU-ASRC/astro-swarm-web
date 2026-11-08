package simulator

import (
	"context"
	"os"
	"os/exec"
	"time"
)

const thumbnailTimeout = 30 * time.Second

func (api *API) extractThumbnail(ctx context.Context, videoPath string, thumbnailPath string) bool {
	extractContext, cancel := context.WithTimeout(ctx, thumbnailTimeout)
	defer cancel()
	command := exec.CommandContext(extractContext, "ffmpeg", "-y", "-i", videoPath, "-vframes", "1", "-f", "image2", thumbnailPath)
	if output, err := command.CombinedOutput(); err != nil {
		api.deps.Logger.Printf("[WARNING] Could not extract run thumbnail: %v %s", err, lastLine(output))
		os.Remove(thumbnailPath)
		return false
	}
	info, err := os.Stat(thumbnailPath)
	return err == nil && info.Size() > 0
}

func lastLine(output []byte) string {
	end := len(output)
	for end > 0 && (output[end-1] == '\n' || output[end-1] == '\r') {
		end--
	}
	start := end
	for start > 0 && output[start-1] != '\n' {
		start--
	}
	return string(output[start:end])
}
