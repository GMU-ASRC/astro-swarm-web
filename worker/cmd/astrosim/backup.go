package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const backupFilePrefix = "astroswarm-backup-"

type exportOptions struct {
	Server string
	Key    string
	Parts  []string
	Target string
}

func runDownload(args []string) error {
	flags := flag.NewFlagSet("astrosim download", flag.ExitOnError)
	server := flags.String("server", DefaultServer, "base url of the AstroSwarm web server")
	key := flags.String("key", defaultAdminKey(), "admin key (API_SECRET_KEY or an admin session token), or set ASTROSWARM_ADMIN_KEY")
	output := flags.String("out", "", "directory the data is extracted into (default astroswarm-data-<time>)")
	withDatabase := flags.Bool("database", false, "also include a pg_dump database dump")
	zipOnly := flags.Bool("zip", false, "keep the downloaded zip instead of extracting it")
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Download every table as JSON lines plus all uploaded files.\n\n")
		fmt.Fprintf(os.Stderr, "  astrosim download -key <admin key> [-out dir] [-database] [-zip]\n\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	parts := []string{"tables", "files"}
	if *withDatabase {
		parts = append(parts, "database")
	}
	folder := *output
	if folder == "" {
		folder = "astroswarm-data-" + time.Now().Format("20060102-150405")
	}
	zipPath := folder + ".zip"

	if err := downloadExport(exportOptions{Server: *server, Key: *key, Parts: parts, Target: zipPath}); err != nil {
		return err
	}
	if *zipOnly {
		fmt.Printf("saved %s\n", zipPath)
		return nil
	}
	if err := extractZip(zipPath, folder); err != nil {
		return err
	}
	if err := os.Remove(zipPath); err != nil {
		return err
	}
	fmt.Printf("extracted to %s\n", folder)
	return nil
}

func runBackup(args []string) error {
	flags := flag.NewFlagSet("astrosim backup", flag.ExitOnError)
	server := flags.String("server", DefaultServer, "base url of the AstroSwarm web server")
	key := flags.String("key", defaultAdminKey(), "admin key (API_SECRET_KEY or an admin session token), or set ASTROSWARM_ADMIN_KEY")
	directory := flags.String("dir", "backups", "directory the backup zip is written to")
	withFiles := flags.Bool("files", false, "also include every uploaded file (replays, configs, runs)")
	withTables := flags.Bool("tables", false, "also include every table as JSON lines")
	keep := flags.Int("keep", 0, "delete older backups in -dir so only this many remain (0 keeps all)")
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Create a database backup that pg_restore can load.\n\n")
		fmt.Fprintf(os.Stderr, "  astrosim backup -key <admin key> [-dir backups] [-files] [-tables] [-keep 7]\n\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	parts := []string{"database"}
	if *withTables {
		parts = append(parts, "tables")
	}
	if *withFiles {
		parts = append(parts, "files")
	}
	if err := os.MkdirAll(*directory, 0o755); err != nil {
		return err
	}
	target := filepath.Join(*directory, backupFilePrefix+time.Now().Format("20060102-150405")+".zip")
	if err := downloadExport(exportOptions{Server: *server, Key: *key, Parts: parts, Target: target}); err != nil {
		return err
	}
	if err := checkDatabaseDump(target); err != nil {
		return err
	}
	fmt.Printf("saved %s\n", target)
	if *keep > 0 {
		return pruneBackups(*directory, *keep)
	}
	return nil
}

func defaultAdminKey() string {
	if value := os.Getenv("ASTROSWARM_ADMIN_KEY"); value != "" {
		return value
	}
	return os.Getenv("API_SECRET_KEY")
}

func downloadExport(options exportOptions) error {
	if options.Key == "" {
		return errors.New("an admin key is required: pass -key or set ASTROSWARM_ADMIN_KEY")
	}
	address := strings.TrimRight(options.Server, "/") + "/api/admin/export?parts=" + url.QueryEscape(strings.Join(options.Parts, ","))
	request, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	request.Header.Set("X-API-Key", options.Key)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 500))
		return fmt.Errorf("server returned %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}

	partial := options.Target + ".part"
	file, err := os.Create(partial)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, &progressReader{source: response.Body, label: filepath.Base(options.Target)})
	closeErr := file.Close()
	fmt.Fprintln(os.Stderr)
	if copyErr != nil || closeErr != nil {
		os.Remove(partial)
		return errors.Join(copyErr, closeErr)
	}
	if written == 0 {
		os.Remove(partial)
		return errors.New("the server sent an empty archive")
	}
	return os.Rename(partial, options.Target)
}

type progressReader struct {
	source   io.Reader
	label    string
	total    int64
	lastShow time.Time
}

func (reader *progressReader) Read(buffer []byte) (int, error) {
	count, err := reader.source.Read(buffer)
	reader.total += int64(count)
	if time.Since(reader.lastShow) > 500*time.Millisecond || err == io.EOF {
		reader.lastShow = time.Now()
		fmt.Fprintf(os.Stderr, "\rdownloading %s: %.1f MB", reader.label, float64(reader.total)/(1024*1024))
	}
	return count, err
}

func extractZip(zipPath string, destination string) error {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer archive.Close()

	root, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	for _, item := range archive.File {
		target := filepath.Join(root, filepath.FromSlash(item.Name))
		if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry %q escapes the destination", item.Name)
		}
		if item.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := extractFile(item, target); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(item *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	source, err := item.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, source)
	return errors.Join(copyErr, file.Close())
}

func checkDatabaseDump(zipPath string) error {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("backup is not a readable zip: %w", err)
	}
	defer archive.Close()
	hasDump := false
	for _, item := range archive.File {
		switch item.Name {
		case "database/pg_dump_error.txt":
			source, openErr := item.Open()
			if openErr != nil {
				return openErr
			}
			message, _ := io.ReadAll(io.LimitReader(source, 1000))
			source.Close()
			return fmt.Errorf("pg_dump failed on the server: %s", strings.TrimSpace(string(message)))
		case "database/database.dump":
			hasDump = item.UncompressedSize64 > 0
		}
	}
	if !hasDump {
		return errors.New("the backup has no database dump; is pg_dump installed on the server?")
	}
	return nil
}

func pruneBackups(directory string, keep int) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	names := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasPrefix(name, backupFilePrefix) && strings.HasSuffix(name, ".zip") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for len(names) > keep {
		if err := os.Remove(filepath.Join(directory, names[0])); err != nil {
			return err
		}
		fmt.Printf("removed old backup %s\n", names[0])
		names = names[1:]
	}
	return nil
}
