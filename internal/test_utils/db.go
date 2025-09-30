package test_utils

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/sqlite" // Import the SQLite driver
)

// NewInMemoryDB creates a new in-memory SQLite database for testing
// Each database is completely isolated from others
func NewInMemoryDB(t *testing.T) *sql.DB {
	t.Helper()

	// Open an in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// Set up cleanup
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// SetupTestDB creates a new in-memory SQLite database and applies all migrations
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Create a new in-memory database
	db := NewInMemoryDB(t)

	// Enable foreign keys for SQLite
	_, err := db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Apply migrations
	err = ApplyMigrations(t, db)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	return db
}

// ApplyMigrations uses golang-migrate to apply all migrations to the database
func ApplyMigrations(t *testing.T, db *sql.DB) error {
	t.Helper()

	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %v", err)
	}

	// Create a new migrate instance
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("failed to create sqlite driver: %v", err)
	}

	// Path to migrations directory
	migrationsPath := fmt.Sprintf("file://%s", filepath.Join(projectRoot, "migrations"))

	// Create a new migrate instance
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %v", err)
	}

	// Apply all migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}

	return nil
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
