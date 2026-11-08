package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"astroswarm/worker/internal/server/pytime"
)

var exportTables = []string{
	"admin_sessions", "admin_users", "app_settings", "benchmarks", "benchmark_runs", "leaderboard_entries",
	"player_evaluations", "sim_configs", "sim_runs", "simulator_entries", "survive_matches", "training_scores", "workers",
}

var precompressedSuffixes = []string{".zlib", ".zip", ".gz", ".png", ".jpg", ".jpeg", ".webp", ".mp4", ".webm", ".dump"}

func PgDumpAvailable() bool {
	_, err := exec.LookPath("pg_dump")
	return err == nil
}

const flushEveryBytes = 1024 * 1024

type flushingWriter struct {
	writer         io.Writer
	flusher        http.Flusher
	unflushedBytes int
}

func (target *flushingWriter) Write(data []byte) (int, error) {
	written, err := target.writer.Write(data)
	target.unflushedBytes += written
	if target.flusher != nil && target.unflushedBytes >= flushEveryBytes {
		target.flusher.Flush()
		target.unflushedBytes = 0
	}
	return written, err
}

func StreamExport(ctx context.Context, w http.ResponseWriter, pool *pgxpool.Pool, databaseURL string, uploadDir string, includeDatabase bool, includeTables bool, includeFiles bool) error {
	flusher, _ := w.(http.Flusher)
	archive := zip.NewWriter(&flushingWriter{writer: w, flusher: flusher})
	manifest := map[string]any{
		"created_at":    pytime.Isoformat(pytime.Now()),
		"database_dump": includeDatabase && PgDumpAvailable(),
		"tables":        includeTables,
		"files":         includeFiles,
		"restore":       "pg_restore --clean --if-exists --no-owner --dbname <database url> database/database.dump",
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")
	if err := writeEntry(archive, "manifest.json", true, manifestJSON); err != nil {
		return err
	}
	if manifest["database_dump"] == true {
		if err := streamDatabaseDump(ctx, archive, databaseURL); err != nil {
			return err
		}
	}
	if includeTables {
		if err := streamTables(ctx, archive, pool); err != nil {
			return err
		}
	}
	if includeFiles {
		if err := streamFiles(archive, uploadDir); err != nil {
			return err
		}
	}
	return archive.Close()
}

func entryWriter(archive *zip.Writer, name string, compress bool) (io.Writer, error) {
	method := zip.Deflate
	if !compress {
		method = zip.Store
	}
	return archive.CreateHeader(&zip.FileHeader{Name: name, Method: method, Modified: time.Now()})
}

func writeEntry(archive *zip.Writer, name string, compress bool, data []byte) error {
	target, err := entryWriter(archive, name, compress)
	if err != nil {
		return err
	}
	_, err = target.Write(data)
	return err
}

func streamDatabaseDump(ctx context.Context, archive *zip.Writer, databaseURL string) error {
	target, err := entryWriter(archive, "database/database.dump", false)
	if err != nil {
		return err
	}
	var errorOutput strings.Builder
	dump := exec.CommandContext(ctx, "pg_dump", "--format=custom", "--no-owner", "--no-privileges", "--dbname", databaseURL)
	dump.Stdout = target
	dump.Stderr = &errorOutput
	if err := dump.Run(); err != nil {
		return writeEntry(archive, "database/pg_dump_error.txt", true, []byte(errorOutput.String()))
	}
	return nil
}

func streamTables(ctx context.Context, archive *zip.Writer, pool *pgxpool.Pool) error {
	counts := map[string]int{}
	for _, table := range exportTables {
		target, err := entryWriter(archive, "tables/"+table+".jsonl", true)
		if err != nil {
			return err
		}
		rows, err := pool.Query(ctx, fmt.Sprintf("SELECT row_to_json(exported)::text FROM %s exported", table))
		if err != nil {
			counts[table] = 0
			continue
		}
		count := 0
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				rows.Close()
				return err
			}
			if _, err := io.WriteString(target, line+"\n"); err != nil {
				rows.Close()
				return err
			}
			count++
		}
		rows.Close()
		counts[table] = count
	}
	countsJSON, _ := json.MarshalIndent(counts, "", "  ")
	return writeEntry(archive, "tables/row_counts.json", true, countsJSON)
}

func streamFiles(archive *zip.Writer, uploadDir string) error {
	if info, err := os.Stat(uploadDir); err != nil || !info.IsDir() {
		return nil
	}
	paths := []string{}
	filepath.WalkDir(uploadDir, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.Type().IsRegular() {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	for _, path := range paths {
		relative, _ := filepath.Rel(uploadDir, path)
		source, err := os.Open(path)
		if err != nil {
			continue
		}
		target, err := entryWriter(archive, "uploads/"+filepath.ToSlash(relative), !isPrecompressed(path))
		if err == nil {
			_, err = io.Copy(target, source)
		}
		source.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func isPrecompressed(path string) bool {
	lower := strings.ToLower(path)
	for _, suffix := range precompressedSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}
