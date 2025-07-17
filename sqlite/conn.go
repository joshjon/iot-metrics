package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	connRetryInterval = time.Second
	connMaxRetries    = 5
)

type options struct {
	dir string
}

type OpenOption func(opts *options)

func WithDir(dir string) OpenOption {
	return func(opts *options) {
		opts.dir = dir
	}
}

func Open(ctx context.Context, opts ...OpenOption) (*sql.DB, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	file := "devices.db"
	if o.dir != "" {
		err := os.MkdirAll(o.dir, 0755)
		if err != nil {
			return nil, fmt.Errorf("create sqlite directory: %w", err)
		}
		file = strings.TrimSuffix(o.dir, "/") + "/" + file
	}

	db, err := sql.Open("sqlite", "file:"+file+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	if _, err = db.Exec("PRAGMA foreign_keys = on;"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err = db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if err = waitHealthy(ctx, db); err != nil {
		return nil, err
	}

	return db, nil
}

type pinger interface {
	PingContext(ctx context.Context) error
}

func waitHealthy(ctx context.Context, pinger pinger) error {
	pingFn := func() error {
		pctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return pinger.PingContext(pctx)
	}
	var err error
	for i := 0; i < connMaxRetries; i++ {
		if err = pingFn(); err == nil {
			return nil
		}
		select {
		case <-time.After(connRetryInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("sqlite db unhealthy: %w", err)
}
