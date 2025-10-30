package database

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/klokku/klokku/internal/config"
	_ "modernc.org/sqlite"
)

// Open opens a SQLite database using the configured path and sets sane pooling for SQLite.
func Open(cfg config.Application) (*sql.DB, error) {
	db, err := sql.Open("sqlite", cfg.Database.SqlitePath)
	if err != nil {
		return nil, err
	}
	// SQLite should keep connections low; WAL is configured via DSN.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	return db, nil
}

// Migrate runs database migrations using golang-migrate against the configured SQLite file.
func Migrate(cfg config.Application) error {
	// golang-migrate sqlite URL does not support DSN query params; strip them if present.
	filePath := stripQuery(cfg.Database.SqlitePath)
	m, err := migrate.New("file://migrations", "sqlite://"+filePath)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func stripQuery(path string) string {
	if i := strings.Index(path, "?"); i >= 0 {
		return path[:i]
	}
	return path
}
