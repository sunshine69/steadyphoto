package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/jbrodriguez/mlog"
	"steadyphoto/internal/processor"
)

func init() {
	mlog.Start(mlog.LevelInfo, "")
}

type MediaRecord struct {
	ID            string         `db:"id"`
	Metadata      sql.NullString `db:"metadata"`
	CapturedAt    time.Time      `db:"captured_at"`
	Filename      string         `db:"filename"`
	FileCreatedAt *time.Time     `db:"file_created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
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

	// Query all media records including file_created_at and updated_at
	query := `SELECT id, metadata, captured_at, filename, file_created_at, updated_at FROM media WHERE deleted_at IS NULL`
	var records []MediaRecord
	err = db.SelectContext(ctx, &records, query)
	if err != nil {
		mlog.Fatalf("Failed to query media records: %v", err)
	}

	mlog.Info("Found %d media records to process", len(records))

	updated := 0
	errors := 0
	skipped := 0

	for _, record := range records {
		// Determine the source label for logging
		source := ""

		// Priority 1: Try EXIF metadata (DateTimeOriginal first, then fallback tags)
		dateStr := ""
		var metadataMap map[string]string
		if record.Metadata.Valid {
			if err := json.Unmarshal([]byte(record.Metadata.String), &metadataMap); err != nil {
				mlog.Info("No date found for %s: metadata parse failed, falling through", record.ID)
				// Don't continue — try filename fallback next
			} else {
				// Try EXIF date tags in priority order
				for _, key := range []string{"DateTimeOriginal", "DateTimeDigitized", "DateTime", "DateCaptured"} {
					if val, ok := metadataMap[key]; ok && val != "" {
						dateStr = val
						source = "EXIF:" + key
						mlog.Info("No date found for %s: EXIF source=%s", record.ID, source)
						break
					}
				}
			}
		}

		// Priority 2: If no EXIF date, try filename heuristic
		if dateStr == "" {
			if fnDate, _, err := processor.ExtractDateFromString(record.Filename); err == nil && !fnDate.IsZero() {
				dateStr = fnDate.Format("2006:01:02 15:04:05")
				source = "filename"
				mlog.Info("No date found for %s: filename source=%s", record.ID, source)
			}
		}

		// Priority 3: If no date from EXIF or filename, use file_created_at
		if dateStr == "" && record.FileCreatedAt != nil && !record.FileCreatedAt.IsZero() {
			dateStr = record.FileCreatedAt.Format("2006:01:02 15:04:05")
			source = "file_created_at"
			mlog.Info("No date found for %s: file_created_at source=%s", record.ID, source)
		}

		// Priority 4: Last fallback — use updated_at if file_created_at is null or empty
		if dateStr == "" && !record.UpdatedAt.IsZero() {
			dateStr = record.UpdatedAt.Format("2006:01:02 15:04:05")
			source = "updated_at"
			mlog.Info("No date found for %s: updated_at source=%s", record.ID, source)
		}

		// If no date found at all, skip this record
		if dateStr == "" {
			mlog.Info("No date found for %s (metadata=%v, filename=%q, file_created_at=%v, updated_at=%v)",
				record.ID, record.Metadata.Valid, record.Filename,
				record.FileCreatedAt, record.UpdatedAt)
			skipped++
			continue
		}

		// Parse the date string
		parsedDate, err := parseDate(dateStr)
		if err != nil {
			mlog.Info("Failed to parse date '%s' for %s: %v", dateStr, record.ID, err)
			errors++
			continue
		}

		// Update captured_at only if different from current value
		if record.CapturedAt.IsZero() || record.CapturedAt.Unix() != parsedDate.Unix() {
			// Update the database
			updateQuery := `UPDATE media SET captured_at = $1 WHERE id = $2`
			_, err = db.ExecContext(ctx, updateQuery, parsedDate, record.ID)
			if err != nil {
				mlog.Info("Failed to update captured_at for %s: %v", record.ID, err)
				errors++
				continue
			}

			updated++
			mlog.Info("Updated %s: %s -> %s (source: %s)",
				record.ID,
				record.CapturedAt.Format(time.RFC3339),
				parsedDate.Format(time.RFC3339),
				source)
		} else {
			mlog.Info("Skipping %s: captured_at already %s (source: %s)",
				record.ID,
				record.CapturedAt.Format(time.RFC3339),
				source)
		}
	}

	mlog.Info("\n=== EXIF Update Complete ===")
	mlog.Info("Total records: %d", len(records))
	mlog.Info("Updated: %d", updated)
	mlog.Info("Skipped (no date): %d", skipped)
	mlog.Info("Errors: %d", errors)
}

func parseDate(dateStr string) (time.Time, error) {
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
