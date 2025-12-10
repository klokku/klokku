package test_utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/klokku/klokku/internal/config"
	"github.com/klokku/klokku/internal/database"
	log "github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func preparePostgresContainer() (*postgres.PostgresContainer, error) {
	ctx := context.Background()

	dbName := "klokku"
	dbUser := "test_klokku"
	dbPassword := "test_klokku"

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %v", err)
	}

	pgContainer, err := postgres.Run(
		ctx, "postgres:18.1-alpine",
		postgres.WithInitScripts(filepath.Join(projectRoot, "dev", "init.sql")),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return nil, err
	}
	return pgContainer, nil
}

// TestWithDB set up a Postgre instance and applies all migrations
func TestWithDB() (*postgres.PostgresContainer, func() *pgx.Conn) {
	ctx := context.Background()

	container, err := preparePostgresContainer()
	if err != nil {
		log.Printf("Failed to start postgres container: %v", err)
		os.Exit(1)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432/tcp")

	log.Infof("Postgres container started at %s:%d", host, port.Int())

	cfg := config.Database{
		Host:   host,
		Port:   port.Int(),
		User:   "test_klokku",
		Pass:   "test_klokku",
		Name:   "klokku",
		Schema: "klokku",
	}

	// Apply migrations
	err = database.Migrate(cfg)
	if err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	err = container.Snapshot(ctx, postgres.WithSnapshotName("postgres-test-snapshot"))
	if err != nil {
		log.Fatalf("Failed to snapshot postgres container: %v", err)
		os.Exit(1)
	}

	return container, func() *pgx.Conn {
		db, err := database.Open(cfg)
		if err != nil {
			log.Fatalf("Failed to open database connection: %v", err)
			os.Exit(1)
		}
		return db
	}
}

// findProjectRoot attempts to locate the project root directory
// It looks for .git directory or go.mod file
func findProjectRoot() (string, error) {
	// Start from the current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree
	for {
		// Check for signs of project root
		if fileExists(filepath.Join(dir, ".git")) || fileExists(filepath.Join(dir, "go.mod")) {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding project root
			return "", fmt.Errorf("could not find project root")
		}
		dir = parent
	}
}

// fileExists checks if a file or directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
