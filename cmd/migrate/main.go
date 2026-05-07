package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	migrationsDirEnv = "MIGRATIONS_DIR"
	defaultMigDir    = "./migrations"
	dbURLKey         = "DATABASE_URL"
	tableName        = "schema_migrations"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/migrate/main.go up")
		os.Exit(1)
	}

	action := os.Args[1]
	switch action {
	case "up":
		runUpMigrations()
	default:
		log.Fatalf("Unknown action: %s. Supported actions: up", action)
	}
}

func runUpMigrations() {
	dbURL := os.Getenv(dbURLKey)
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	migDir := os.Getenv(migrationsDirEnv)
	if migDir == "" {
		migDir = defaultMigDir
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := createMigrationTable(db.DB); err != nil {
		log.Fatalf("Failed to create migration table: %v", err)
	}

	currentVersion, err := getCurrentVersion(db.DB)
	if err != nil {
		log.Fatalf("Failed to get current version: %v", err)
	}

	files, err := os.ReadDir(migDir)
	if err != nil {
		log.Fatalf("Failed to read migrations directory %s: %v", migDir, err)
	}

	var versions []int
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".up.sql") {
			version, err := extractVersion(file.Name())
			if err != nil {
				log.Printf("Skipping %s: invalid version format", file.Name())
				continue
			}
			if version > currentVersion {
				versions = append(versions, version)
			}
		}
	}

	sort.Ints(versions)

	if len(versions) == 0 {
		fmt.Println("No migrations to run.")
		return
	}

	for _, version := range versions {
		filePath := findMigFile(migDir, version)
		if filePath == "" {
			log.Fatalf("Migration file for version %d not found", version)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Failed to read migration file: %v", err)
		}

		fmt.Printf("Applying migration version %d...\n", version)

		tx, err := db.Beginx()
		if err != nil {
			log.Fatalf("Transaction begin failed: %v", err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			log.Fatalf("Migration %d execution failed: %v", version, err)
		}

		if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s (version) VALUES ($1)", tableName), version); err != nil {
			tx.Rollback()
			log.Fatalf("Failed to record migration %d: %v", version, err)
		}

		if err := tx.Commit(); err != nil {
			log.Fatalf("Commit failed for migration %d: %v", version, err)
		}

		fmt.Printf("Successfully applied migration %d.\n", version)
		currentVersion = version
	}

	fmt.Println("All migrations completed successfully.")
}

func findMigFile(dir string, version int) string {
	files, _ := os.ReadDir(dir)
	prefix := fmt.Sprintf("%04d_", version)
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ".up.sql") {
			return filepath.Join(dir, f.Name())
		}
	}
	return ""
}

func extractVersion(filename string) (int, error) {
	re := regexp.MustCompile(`^(\d+)_`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid filename format: %s", filename)
	}
	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	return version, nil
}

func createMigrationTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY);`
	_, err := db.Exec(query)
	return err
}

func getCurrentVersion(db *sql.DB) (int, error) {
	var version *int // Use pointer to detect NULL
	err := db.QueryRow(fmt.Sprintf("SELECT MAX(version) FROM %s", tableName)).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}
	
	// If version is nil (no rows or all NULL), return 0 as base case
	if version == nil {
		return 0, nil
	}
	
	return *version, nil
}
