package admin

import (
	"context"
	"io/fs"
	"path/filepath"
	"strconv"
	"time"

	"astroswarm/worker/internal/server/pytime"
)

const (
	measureEvery       = 10 * time.Minute
	measureLockKey     = 734021118
	uploadsBytesKey    = "uploads_bytes"
	uploadsFilesKey    = "uploads_files"
	uploadsMeasuredKey = "uploads_measured_at"
)

type uploadsUsage struct {
	bytes      int64
	files      int64
	measuredAt any
}

func (api *API) settingNumber(ctx context.Context, key string) (int64, error) {
	stored, found, err := api.deps.Settings.Get(ctx, key)
	if err != nil || !found || stored == "" {
		return 0, err
	}
	number, err := strconv.ParseInt(stored, 10, 64)
	if err != nil {
		return 0, nil
	}
	return number, nil
}

func (api *API) cachedUploadsUsage(ctx context.Context) (uploadsUsage, error) {
	usage := uploadsUsage{}
	var err error
	if usage.bytes, err = api.settingNumber(ctx, uploadsBytesKey); err != nil {
		return usage, err
	}
	if usage.files, err = api.settingNumber(ctx, uploadsFilesKey); err != nil {
		return usage, err
	}
	measuredAt, found, err := api.deps.Settings.Get(ctx, uploadsMeasuredKey)
	if err != nil {
		return usage, err
	}
	if found {
		usage.measuredAt = measuredAt
	}
	return usage, nil
}

func isStale(measuredAt any) bool {
	text, isText := measuredAt.(string)
	if !isText || text == "" {
		return true
	}
	measured, parsed := pytime.ParseISO(text)
	if !parsed {
		return true
	}
	return pytime.Now().Sub(measured) > measureEvery
}

func (api *API) refreshUploadsIfStale(ctx context.Context, force bool) (bool, error) {
	if !force {
		usage, err := api.cachedUploadsUsage(ctx)
		if err != nil {
			return false, err
		}
		if !isStale(usage.measuredAt) {
			return false, nil
		}
	}
	if !api.measuring.TryLock() {
		return false, nil
	}
	go api.measureUploads()
	return true, nil
}

func walkUploads(uploadDir string) (int64, int64) {
	var totalBytes, fileCount int64
	filepath.WalkDir(uploadDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		totalBytes += info.Size()
		fileCount++
		return nil
	})
	return totalBytes, fileCount
}

func (api *API) measureUploads() {
	defer api.measuring.Unlock()
	if err := api.measureUploadsUnderLock(context.Background()); err != nil {
		api.deps.Logger.Printf("[WARNING] Could not measure uploads: %v", err)
	}
}

func (api *API) measureUploadsUnderLock(ctx context.Context) error {
	connection, err := api.deps.Store.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer connection.Release()
	var locked bool
	if err := connection.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", int64(measureLockKey)).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return nil
	}
	defer connection.Exec(ctx, "SELECT pg_advisory_unlock($1)", int64(measureLockKey))
	totalBytes, fileCount := walkUploads(api.deps.Config.UploadDir)
	appSettings := api.deps.Settings
	if err := appSettings.Set(ctx, uploadsBytesKey, strconv.FormatInt(totalBytes, 10)); err != nil {
		return err
	}
	if err := appSettings.Set(ctx, uploadsFilesKey, strconv.FormatInt(fileCount, 10)); err != nil {
		return err
	}
	return appSettings.Set(ctx, uploadsMeasuredKey, pytime.Isoformat(pytime.Now()))
}
