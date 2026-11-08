package benchmarks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	MaxChunkBytes   = 8 * 1024 * 1024
	maxReplayBytes  = 256 * 1024 * 1024
	InlineUploadID  = "inline"
	replayExtension = ".zlib"
	binaryExtension = ".asr"
)

var (
	safeName          = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)
	ErrInvalidName    = errors.New("invalid replay name")
	ErrInvalidPath    = errors.New("invalid replay path")
	ErrChunkTooLarge  = errors.New("chunk too large")
	ErrReplayTooLarge = errors.New("replay too large")
)

type Files struct {
	root string
}

func NewFiles(uploadDir string) *Files {
	return &Files{root: filepath.Join(uploadDir, "replays")}
}

func checkName(name string) error {
	if !safeName.MatchString(name) {
		return ErrInvalidName
	}
	return nil
}

func RelativePath(evaluationID string, uploadID string, key string) (string, error) {
	for _, name := range []string{evaluationID, uploadID, key} {
		if err := checkName(name); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s/%s/%s%s", evaluationID, uploadID, key, replayExtension), nil
}

func BinaryKeyFor(framesKey string) string {
	return strings.TrimSuffix(framesKey, replayExtension) + binaryExtension
}

func fileStem(name string) (string, bool) {
	for _, extension := range []string{replayExtension, binaryExtension} {
		if strings.HasSuffix(name, extension) {
			return strings.TrimSuffix(name, extension), true
		}
	}
	return "", false
}

func (files *Files) absolute(relative string) (string, error) {
	parts := strings.Split(relative, "/")
	if len(parts) != 3 {
		return "", ErrInvalidPath
	}
	stem, known := fileStem(parts[2])
	if !known {
		return "", ErrInvalidPath
	}
	for _, name := range []string{parts[0], parts[1], stem} {
		if err := checkName(name); err != nil {
			return "", err
		}
	}
	return filepath.Join(files.root, parts[0], parts[1], parts[2]), nil
}

func (files *Files) WriteChunk(evaluationID string, uploadID string, key string, offset int64, data []byte) (string, int64, error) {
	if len(data) > MaxChunkBytes {
		return "", 0, ErrChunkTooLarge
	}
	relative, err := RelativePath(evaluationID, uploadID, key)
	if err != nil {
		return "", 0, err
	}
	path, _ := files.absolute(relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", 0, err
	}
	size := int64(0)
	if info, err := os.Stat(path); err == nil {
		size = info.Size()
	}
	if offset > size {
		return relative, size, nil
	}
	if offset+int64(len(data)) > maxReplayBytes {
		return "", 0, ErrReplayTooLarge
	}
	handle, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return "", 0, err
	}
	defer handle.Close()
	if err := handle.Truncate(offset); err != nil {
		return "", 0, err
	}
	if _, err := handle.WriteAt(data, offset); err != nil {
		return "", 0, err
	}
	return relative, offset + int64(len(data)), nil
}

func (files *Files) Read(relative string) ([]byte, bool) {
	path, err := files.absolute(relative)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func (files *Files) Open(relative string) (*os.File, error) {
	path, err := files.absolute(relative)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (files *Files) Exists(relative string) bool {
	path, err := files.absolute(relative)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func (files *Files) Size(relative string) *int64 {
	path, err := files.absolute(relative)
	if err != nil {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	size := info.Size()
	return &size
}

func (files *Files) WriteFile(evaluationID string, uploadID string, key string, data []byte) (string, error) {
	relative, err := RelativePath(evaluationID, uploadID, key)
	if err != nil {
		return "", err
	}
	return relative, files.WriteRelative(relative, data)
}

func (files *Files) WriteRelative(relative string, data []byte) error {
	path, err := files.absolute(relative)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporaryPath := path + ".tmp"
	handle, err := os.Create(temporaryPath)
	if err != nil {
		return err
	}
	_, writeErr := handle.Write(data)
	syncErr := handle.Sync()
	closeErr := handle.Close()
	for _, err := range []error{writeErr, syncErr, closeErr} {
		if err != nil {
			os.Remove(temporaryPath)
			return err
		}
	}
	return os.Rename(temporaryPath, path)
}

func (files *Files) Remove(relative string) error {
	path, err := files.absolute(relative)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (files *Files) UploadIDs(evaluationID string) ([]string, error) {
	if err := checkName(evaluationID); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(files.root, evaluationID))
	if err != nil {
		return nil, nil
	}
	uploadIDs := []string{}
	for _, entry := range entries {
		if safeName.MatchString(entry.Name()) {
			uploadIDs = append(uploadIDs, entry.Name())
		}
	}
	return uploadIDs, nil
}

func (files *Files) RemoveUpload(evaluationID string, uploadID string) {
	if checkName(evaluationID) != nil || checkName(uploadID) != nil {
		return
	}
	folder := filepath.Join(files.root, evaluationID)
	os.RemoveAll(filepath.Join(folder, uploadID))
	os.Remove(folder)
}
