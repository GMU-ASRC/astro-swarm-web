package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	ManifestName        = "manifest.json"
	DatabaseDumpName    = "database.dump"
	UploadsFolder       = "uploads"
	UploadsMirrorFolder = "uploads-mirror"
	maxBackupAge        = 24 * time.Hour
)

type Manifest struct {
	CreatedAt         string   `json:"created_at"`
	DatabaseDump      string   `json:"database_dump"`
	DatabaseDumpBytes int64    `json:"database_dump_bytes"`
	UploadsFolder     string   `json:"uploads_folder"`
	UploadFileCount   int64    `json:"upload_file_count"`
	UploadTotalBytes  int64    `json:"upload_total_bytes"`
	Verified          bool     `json:"verified"`
	Problems          []string `json:"problems"`
}

func readManifest(folder string) (Manifest, bool) {
	data, err := os.ReadFile(filepath.Join(folder, ManifestName))
	if err != nil {
		return Manifest{}, false
	}
	var manifest Manifest
	if json.Unmarshal(data, &manifest) != nil {
		return Manifest{}, false
	}
	return manifest, true
}

func RecentVerified(root string) (string, bool) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", false
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	newestFolder := ""
	var newestTime time.Time
	for _, name := range names {
		folder := filepath.Join(root, name)
		manifest, found := readManifest(folder)
		if !found || !manifest.Verified || manifest.CreatedAt == "" {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
		if err != nil {
			continue
		}
		if newestFolder == "" || createdAt.After(newestTime) {
			newestFolder, newestTime = folder, createdAt
		}
	}
	if newestFolder == "" || time.Since(newestTime) > maxBackupAge {
		return "", false
	}
	return newestFolder, true
}

func writeManifest(folder string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	temporaryPath := filepath.Join(folder, ManifestName+".tmp")
	if err := os.WriteFile(temporaryPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporaryPath, filepath.Join(folder, ManifestName))
}
