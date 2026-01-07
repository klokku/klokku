package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klokku/klokku/internal/config"
)

// Open opens a Postgres database
func Open(cfg config.Database) (*pgxpool.Pool, error) {
	ctx := context.Background()

	// Escape single quotes in password for PostgreSQL connection string
	escapedPassword := strings.ReplaceAll(cfg.Pass, "'", "\\'")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password='%s' dbname=%s sslmode=disable options='-c search_path=%s'", cfg.Host,
		cfg.Port, cfg.User, escapedPassword, cfg.Name, cfg.Schema)
	poolConfig, err := pgxpool.ParseConfig(psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Optional: Configure pool settings for better performance
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// Migrate runs database migrations using golang-migrate against the configured DB.
func Migrate(cfg config.Database) error {
	escapedPassword := url.QueryEscape(cfg.Pass)

	dbUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=%s", cfg.User, escapedPassword, cfg.Host, cfg.Port, cfg.Name, cfg.Schema)

	migrationsPath, err := findMigrationsPath()
	if err != nil {
		return fmt.Errorf("failed to locate migrations directory: %w", err)
	}

	m, err := migrate.New("file://"+migrationsPath, dbUrl)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration up failed: %w", err)
	}

	return nil
}

// findMigrationsPath searches upward from the current working directory for a "migrations" directory
// and returns its absolute path. This makes migrations resolution robust in tests where the working
// directory can be different from the project root.
func findMigrationsPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, "migrations")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			abs, err := filepath.Abs(candidate)
			if err != nil {
				return "", err
			}
			return abs, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("migrations directory not found")
}
