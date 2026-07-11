package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jbrodriguez/mlog"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

type MediaRecord struct {
	ID        string       `db:"id"`
	Metadata  sql.NullString `db:"metadata"`
	CapturedAt time.Time    `db:"captured_at"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		mlog.Info("Warning: Failed to load .env file: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable",
			os.Getenv("POSTGRES_USER"),
			os.Getenv("POSTGRES_PASSWORD"),
			os.Getenv("POSTGRES_DB"))
	}

	if dbURL == "" {
		mlog.Fatal("DATABASE_URL environment variable is not set")
	}

	flag.Parse()

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		mlog.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		mlog.Fatalf("Failed to ping database: %v", err)
	}

	ctx := context.Background()

	// Query all media records
	query := `SELECT id, metadata, captured_at FROM media WHERE deleted_at IS NULL`
	var records []MediaRecord
	err = db.SelectContext(ctx, &records, query)
	if err != nil {
		mlog.Fatalf("Failed to query media records: %v", err)
	}

	mlog.Info("Found %d media records to process", len(records))

	updated := 0
	errors := 0

	for _, record := range records {
		// Parse metadata JSONB
		var metadata map[string]string
		if !record.Metadata.Valid {
			continue
		}
		if err := json.Unmarshal([]byte(record.Metadata.String), &metadata); err != nil {
			mlog.Info("Failed to parse metadata for %s: %v", record.ID, err)
			errors++
			continue
		}

		// Look for DateTimeOriginal in metadata
		dateStr := ""
		for _, key := range []string{"DateTimeOriginal", "DateTimeDigitized", "DateTime", "DateCaptured"} {
			if val, ok := metadata[key]; ok && val != "" {
				dateStr = val
				break
			}
		}

		if dateStr == "" {
			continue
		}

		// Parse the date string
		parsedDate, err := parseDate(dateStr)
		if err != nil {
			mlog.Info("Failed to parse date '%s' for %s: %v", dateStr, record.ID, err)
			errors++
			continue
		}

		// Update captured_at if the date is different from current value
		if !record.CapturedAt.IsZero() && record.CapturedAt == parsedDate {
			continue
		}

		// Update the database
		updateQuery := `UPDATE media SET captured_at = $1 WHERE id = $2`
		_, err = db.ExecContext(ctx, updateQuery, parsedDate, record.ID)
		if err != nil {
			mlog.Info("Failed to update captured_at for %s: %v", record.ID, err)
			errors++
			continue
		}

		updated++
		mlog.Info("Updated %s: %s -> %s", record.ID, record.CapturedAt.Format(time.RFC3339), parsedDate.Format(time.RFC3339))
	}

	mlog.Info("Done! Updated: %d, Errors: %d", updated, errors)
}

func parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	formats := []string{
		"2006:01:02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02",
		"2006/01/02",
		"2006-01-02",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
