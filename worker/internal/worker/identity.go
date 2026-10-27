package worker

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func loadOrCreateID(fixedID string, configuredPath string, logger *log.Logger) (string, error) {
	if value := strings.TrimSpace(fixedID); value != "" {
		return value, nil
	}
	paths := idFileCandidates(configuredPath)
	for _, path := range paths {
		if value := readID(path); value != "" {
			return value, nil
		}
	}
	value, err := newUUID()
	if err != nil {
		return "", err
	}
	for _, path := range paths {
		savedID, err := saveID(path, value)
		if err == nil {
			logger.Printf("worker id saved to %s", path)
			return savedID, nil
		}
		logger.Printf("could not save the worker id to %s (%v)", path, err)
	}
	logger.Printf("worker id %s could not be saved; set WORKER_ID or a writable WORKER_ID_FILE or this worker will register again after a restart", value)
	return value, nil
}

func idFileCandidates(configuredPath string) []string {
	paths := []string{}
	if configuredPath != "" {
		paths = append(paths, configuredPath)
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		fallbackPath := filepath.Join(configDir, "astroswarm", "worker_id")
		if fallbackPath != configuredPath {
			paths = append(paths, fallbackPath)
		}
	}
	return paths
}

func readID(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func saveID(path string, value string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		if existingID := readID(path); existingID != "" {
			return existingID, nil
		}
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	}
	if err != nil {
		return "", err
	}
	_, writeErr := file.WriteString(value + "\n")
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return "", err
	}
	if readID(path) != value {
		return "", errors.New("the saved worker id could not be read back")
	}
	return value, nil
}

func newUUID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}
