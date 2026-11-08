package benchmarks

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConvertSummary struct {
	Converted int
	Kept      int
	Problems  []string
}

type convertJob struct {
	id           int64
	framesKey    string
	framesFormat string
}

func ConvertPendingRuns(ctx context.Context, pool *pgxpool.Pool, files *Files, batchSize int, progress func(done int, summary ConvertSummary)) (ConvertSummary, error) {
	summary := ConvertSummary{}
	lastID := int64(0)
	done := 0
	for {
		jobs, err := pendingConversions(ctx, pool, lastID, batchSize)
		if err != nil || len(jobs) == 0 {
			return summary, err
		}
		lastID = jobs[len(jobs)-1].id
		if err := convertJobs(ctx, pool, files, jobs, &summary); err != nil {
			return summary, err
		}
		done += len(jobs)
		progress(done, summary)
	}
}

func ConvertBenchmarkRuns(ctx context.Context, pool *pgxpool.Pool, files *Files, benchmarkID string) (ConvertSummary, error) {
	summary := ConvertSummary{}
	rows, err := pool.Query(ctx, `SELECT id, frames_key, coalesce(frames_format, 'delta') FROM benchmark_runs
		WHERE benchmark_id = $1 AND binary_key IS NULL AND frames_key IS NOT NULL ORDER BY id`, benchmarkID)
	if err != nil {
		return summary, err
	}
	jobs, err := scanConvertJobs(rows)
	if err != nil || len(jobs) == 0 {
		return summary, err
	}
	err = convertJobs(ctx, pool, files, jobs, &summary)
	return summary, err
}

func pendingConversions(ctx context.Context, pool *pgxpool.Pool, afterID int64, limit int) ([]convertJob, error) {
	rows, err := pool.Query(ctx, `SELECT id, frames_key, coalesce(frames_format, 'delta') FROM benchmark_runs
		WHERE binary_key IS NULL AND frames_key IS NOT NULL AND id > $1 ORDER BY id LIMIT $2`, afterID, limit)
	if err != nil {
		return nil, err
	}
	return scanConvertJobs(rows)
}

func scanConvertJobs(rows pgx.Rows) ([]convertJob, error) {
	defer rows.Close()
	jobs := []convertJob{}
	for rows.Next() {
		var job convertJob
		if err := rows.Scan(&job.id, &job.framesKey, &job.framesFormat); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func convertJobs(ctx context.Context, pool *pgxpool.Pool, files *Files, jobs []convertJob, summary *ConvertSummary) error {
	var lock sync.Mutex
	var firstErr error
	queue := make(chan convertJob)
	var group sync.WaitGroup
	for range runtime.NumCPU() {
		group.Add(1)
		go func() {
			defer group.Done()
			for job := range queue {
				converted, err := files.ConvertToBinary(job.framesKey, job.framesFormat)
				lock.Lock()
				switch {
				case errors.Is(err, ErrNotFlatFrames) || errors.Is(err, ErrLegacyMissing):
					summary.Kept++
					summary.Problems = append(summary.Problems, fmt.Sprintf("run %d stays in the old format: %v", job.id, err))
				case err != nil:
					summary.Kept++
					summary.Problems = append(summary.Problems, fmt.Sprintf("run %d: %v", job.id, err))
				}
				lock.Unlock()
				if err != nil {
					continue
				}
				_, err = pool.Exec(ctx, "UPDATE benchmark_runs SET binary_key = $2, binary_bytes = $3 WHERE id = $1",
					job.id, converted.Key, converted.Bytes)
				lock.Lock()
				if err != nil && firstErr == nil {
					firstErr = err
				} else if err == nil {
					summary.Converted++
				}
				lock.Unlock()
			}
		}()
	}
	for _, job := range jobs {
		queue <- job
	}
	close(queue)
	group.Wait()
	return firstErr
}

type verifyJob struct {
	id           int64
	framesKey    string
	framesFormat string
	binaryKey    string
}

func VerifyConvertedRuns(ctx context.Context, pool *pgxpool.Pool, files *Files, batchSize int, progress func(done int)) ([]string, error) {
	problems := []string{}
	lastID, done := int64(0), 0
	for {
		rows, err := pool.Query(ctx, `SELECT id, frames_key, coalesce(frames_format, 'delta'), binary_key FROM benchmark_runs
			WHERE binary_key IS NOT NULL AND frames_key IS NOT NULL AND id > $1 ORDER BY id LIMIT $2`, lastID, batchSize)
		if err != nil {
			return problems, err
		}
		jobs := []verifyJob{}
		for rows.Next() {
			var job verifyJob
			if err := rows.Scan(&job.id, &job.framesKey, &job.framesFormat, &job.binaryKey); err != nil {
				rows.Close()
				return problems, err
			}
			jobs = append(jobs, job)
		}
		rows.Close()
		if len(jobs) == 0 {
			return problems, rows.Err()
		}
		lastID = jobs[len(jobs)-1].id
		var lock sync.Mutex
		queue := make(chan verifyJob)
		var group sync.WaitGroup
		for range runtime.NumCPU() {
			group.Add(1)
			go func() {
				defer group.Done()
				for job := range queue {
					if err := files.VerifyBinary(job.framesKey, job.framesFormat, job.binaryKey); err != nil {
						lock.Lock()
						problems = append(problems, fmt.Sprintf("run %d: %v", job.id, err))
						lock.Unlock()
					}
				}
			}()
		}
		for _, job := range jobs {
			queue <- job
		}
		close(queue)
		group.Wait()
		done += len(jobs)
		progress(done)
	}
}
