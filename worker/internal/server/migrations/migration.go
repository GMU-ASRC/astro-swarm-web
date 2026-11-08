package migrations

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Migration interface {
	Version() int
	Name() string
	ChangesData() bool
	Describe(ctx context.Context, run *Context) []string
	Run(ctx context.Context, run *Context) error
	Verify(ctx context.Context, run *Context) []string
}

type ManualMigration interface {
	Manual() bool
}

func isManual(migration Migration) bool {
	manual, isManualType := migration.(ManualMigration)
	return isManualType && manual.Manual()
}

type Context struct {
	Pool       *pgxpool.Pool
	Connection *pgx.Conn
	Migration  Migration
	Logger     *log.Logger
	UploadDir  string
}

func (run *Context) Execute(ctx context.Context, statement string, arguments ...any) error {
	_, err := run.Connection.Exec(ctx, statement, arguments...)
	return err
}

func (run *Context) ExecuteOutsideTransaction(ctx context.Context, statement string) error {
	_, err := run.Pool.Exec(ctx, statement)
	return err
}

func (run *Context) Scalar(ctx context.Context, statement string, target any, arguments ...any) error {
	return run.Connection.QueryRow(ctx, statement, arguments...).Scan(target)
}

func (run *Context) Log(format string, arguments ...any) {
	prefix := fmt.Sprintf("[%03d %s] ", run.Migration.Version(), run.Migration.Name())
	run.Logger.Printf("[INFO] "+prefix+format, arguments...)
}

func tableExists(ctx context.Context, run *Context, table string) (bool, error) {
	var exists bool
	err := run.Scalar(ctx, "SELECT to_regclass($1) IS NOT NULL", &exists, "public."+table)
	return exists, err
}

func columnExists(ctx context.Context, run *Context, table string, column string) (bool, error) {
	var exists bool
	err := run.Scalar(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)",
		&exists, table, column)
	return exists, err
}
