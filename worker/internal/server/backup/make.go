package backup

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"astroswarm/worker/internal/server/pytime"
)

func Make(ctx context.Context, databaseURL string, uploadDir string, backupRoot string, logger *log.Logger) (string, error) {
	createdAt := pytime.Now()
	folder := filepath.Join(backupRoot, "astroswarm-"+createdAt.Format("20060102-150405"))
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		return "", err
	}
	if err := os.Mkdir(folder, 0o755); err != nil {
		return "", err
	}

	dumpPath := filepath.Join(folder, DatabaseDumpName)
	logger.Printf("[INFO] Dumping database to %s", dumpPath)
	dump := exec.CommandContext(ctx, "pg_dump", "--format=custom", "--no-owner", "--no-privileges", "--file", dumpPath, "--dbname", databaseURL)
	dump.Stdout, dump.Stderr = os.Stdout, os.Stderr
	if err := dump.Run(); err != nil {
		return folder, fmt.Errorf("pg_dump failed: %w", err)
	}
	problems := checkDump(ctx, dumpPath)

	mirror := filepath.Join(backupRoot, UploadsMirrorFolder)
	logger.Printf("[INFO] Syncing %s into %s", uploadDir, mirror)
	sync := syncTree(uploadDir, mirror)
	problems = append(problems, sync.problems...)
	logger.Printf("[INFO] Uploads mirror: %d new or changed file(s), %d already present", sync.copied, sync.unchanged)
	fileCount, totalBytes := sync.copied+sync.unchanged, sync.totalBytes

	dumpInfo, _ := os.Stat(dumpPath)
	manifest := Manifest{
		CreatedAt:        pytime.Isoformat(createdAt),
		DatabaseDump:     DatabaseDumpName,
		UploadsFolder:    filepath.Join("..", UploadsMirrorFolder),
		UploadFileCount:  fileCount,
		UploadTotalBytes: totalBytes,
		Verified:         len(problems) == 0,
		Problems:         firstProblems(problems, 200),
	}
	if dumpInfo != nil {
		manifest.DatabaseDumpBytes = dumpInfo.Size()
	}
	if err := writeManifest(folder, manifest); err != nil {
		return folder, err
	}
	if len(problems) > 0 {
		for _, problem := range firstProblems(problems, 50) {
			logger.Printf("[ERROR] %s", problem)
		}
		return folder, fmt.Errorf("backup at %s is NOT verified (%d problem(s))", folder, len(problems))
	}
	logger.Printf("[INFO] Backup verified at %s: %d file(s), %d byte(s)", folder, fileCount, totalBytes)
	return folder, nil
}

func checkDump(ctx context.Context, dumpPath string) []string {
	output, err := exec.CommandContext(ctx, "pg_restore", "--list", dumpPath).Output()
	if err != nil {
		return []string{"pg_restore could not read the dump: " + err.Error()}
	}
	if !strings.Contains(string(output), " TABLE DATA ") {
		return []string{"the dump has no table data"}
	}
	return nil
}

type syncResult struct {
	copied     int64
	unchanged  int64
	totalBytes int64
	problems   []string
}

func sameFile(source fs.FileInfo, targetPath string) bool {
	target, err := os.Stat(targetPath)
	return err == nil && target.Size() == source.Size() && target.ModTime().Equal(source.ModTime())
}

func syncTree(sourceRoot string, targetRoot string) syncResult {
	result := syncResult{problems: []string{}}
	walkErr := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			result.problems = append(result.problems, fmt.Sprintf("could not read %s: %v", path, err))
			return nil
		}
		relative, _ := filepath.Rel(sourceRoot, path)
		target := filepath.Join(targetRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		sourceInfo, err := entry.Info()
		if err != nil {
			result.problems = append(result.problems, fmt.Sprintf("could not read %s: %v", path, err))
			return nil
		}
		result.totalBytes += sourceInfo.Size()
		if sameFile(sourceInfo, target) {
			result.unchanged++
			return nil
		}
		copied, err := copyFile(path, target)
		if err != nil {
			result.problems = append(result.problems, fmt.Sprintf("could not copy %s: %v", path, err))
			return nil
		}
		if copied != sourceInfo.Size() {
			result.problems = append(result.problems, fmt.Sprintf("size mismatch for %s: %d != %d", path, sourceInfo.Size(), copied))
		}
		result.copied++
		return nil
	})
	if walkErr != nil {
		result.problems = append(result.problems, walkErr.Error())
	}
	return result
}

func copyFile(sourcePath string, targetPath string) (int64, error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	temporaryPath := targetPath + ".partial"
	target, err := os.Create(temporaryPath)
	if err != nil {
		return 0, err
	}
	copied, copyErr := io.Copy(target, source)
	syncErr := target.Sync()
	closeErr := target.Close()
	for _, err := range []error{copyErr, syncErr, closeErr} {
		if err != nil {
			os.Remove(temporaryPath)
			return copied, err
		}
	}
	if info, err := os.Stat(sourcePath); err == nil {
		os.Chtimes(temporaryPath, info.ModTime(), info.ModTime())
	}
	return copied, os.Rename(temporaryPath, targetPath)
}

func firstProblems(problems []string, limit int) []string {
	if len(problems) > limit {
		return problems[:limit]
	}
	return problems
}
