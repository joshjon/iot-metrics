package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func Migrate(db *sql.DB, fsys fs.FS) error {
	sd, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("open migrations fs: %w", err)
	}
	defer sd.Close() //nolint:errcheck

	driver, err := sqlite.WithInstance(db, new(sqlite.Config))
	if err != nil {
		return fmt.Errorf("create sqlite driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sd, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}
