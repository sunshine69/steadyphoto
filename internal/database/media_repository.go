package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Helper function to parse date range strings in various formats
func parseDateRange(dateRange string) (start time.Time, end time.Time, err error) {
	if dateRange == "" {
		return time.Time{}, time.Time{}, nil
	}

	// First check if this is a range (contains "-" but not as part of date)
	if strings.Contains(dateRange, " - ") {
		parts := strings.SplitN(dateRange, " - ", 2)
		start, err = parseSingleDate(parts[0])
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end, err = parseSingleDate(parts[1])
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	} else {
		// Single date - search for that day only
		start, err = parseSingleDate(dateRange)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = start.Add(24*time.Hour - time.Second)
	}

	// Swap if start > end
	if start.After(end) {
		start, end = end, start
	}

	return start, end, nil
}

// parseSingleDate parses a date string in various formats
func parseSingleDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, nil
	}

	// Try different formats
	formats := []string{
		"01/02/2006",
		"2006/01/02",
		"01/02/2006 15:04:05",
		"2006/01/02 15:04:05",
		"01-02-2006",
		"2006-01-02",
		"01.02.2006",
		"2006.01.02",
		"2006",
		"2006/01",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// For year-only format, set to Jan 1 of that year
			if format == "2006" {
				t = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			}
			// For year/month format, set to first day of that month
			if format == "2006/01" {
				t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

type PostgresMediaRepository struct {
	db *sqlx.DB
}

func NewPostgresMediaRepository(db *sqlx.DB) *PostgresMediaRepository {
	return &PostgresMediaRepository{db: db}
}

// GetDB returns the underlying database connection for direct queries
func (r *PostgresMediaRepository) GetDB() *sqlx.DB {
	return r.db
}

func (r *PostgresMediaRepository) Create(ctx context.Context, media *domain.Media) error {
	query := `
		INSERT INTO media (id, user_id, path, filename, hash, size_bytes, width, height, captured_at, media_type, metadata, video_metadata, created_at, updated_at, tags)
		VALUES (:id, :user_id, :path, :filename, :hash, :size_bytes, :width, :height, :captured_at, :media_type, :metadata, :video_metadata, :created_at, :updated_at, :tags)
	`
	_, err := r.db.NamedExecContext(ctx, query, media)
	return err
}

func (r *PostgresMediaRepository) GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*domain.Media, error) {
	var media domain.Media
	query := `SELECT * FROM media WHERE deleted_at IS NULL AND id = $1`
	args := []interface{}{id}

	if userID != nil {
		query += ` AND user_id = $2`
		args = append(args, *userID)
	}

	err := r.db.GetContext(ctx, &media, query, args...)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] GetById %v\n", media)
	return &media, nil
}

func (r *PostgresMediaRepository) GetByHash(ctx context.Context, hash string) (*domain.Media, error) {
	var media domain.Media
	query := `SELECT * FROM media WHERE deleted_at IS NULL AND hash = $1`
	err := r.db.GetContext(ctx, &media, query, hash)
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (r *PostgresMediaRepository) Update(ctx context.Context, media *domain.Media) error {
	query := `
		UPDATE media
		SET path = :path, filename = :filename, size_bytes = :size_bytes, width = :width, height = :height, media_type = :media_type, metadata = :metadata, video_metadata = :video_metadata, updated_at = :updated_at, tags = :tags
		WHERE id = :id AND user_id = :user_id AND deleted_at IS NULL
	`
	result, err := r.db.NamedExecContext(ctx, query, media)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no record updated (id not found or ownership mismatch)")
	}

	return nil
}

func (r *PostgresMediaRepository) Delete(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error {
	query := `UPDATE media SET deleted_at = NOW() WHERE id = $1`
	args := []interface{}{id}

	if userID != nil {
		query += ` AND user_id = $2`
		args = append(args, *userID)
	} else {
		return fmt.Errorf("user ID is required for soft delete")
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("media not found or already deleted (id: %s)", id)
	}

	return nil
}

func (r *PostgresMediaRepository) DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error {
	query := `DELETE FROM faces WHERE media_id = $1`
	_, err := r.db.ExecContext(ctx, query, mediaID)
	return err
}

func (r *PostgresMediaRepository) List(ctx context.Context, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	countQuery := `SELECT COUNT(*) FROM media WHERE deleted_at IS NULL`
	argsCount := []interface{}{}

	if userID != nil {
		countQuery += ` AND user_id = $1`
		argsCount = append(argsCount, *userID)
	}

	err := r.db.GetContext(ctx, &total, countQuery, argsCount...)
	if err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT * FROM media WHERE deleted_at IS NULL`
	argsList := []interface{}{}

	if userID != nil {
		listQuery += ` AND user_id = $1`
		argsList = append(argsList, *userID)
	}

	listQuery += fmt.Sprintf(` ORDER BY captured_at DESC LIMIT $%d OFFSET $%d`, len(argsList)+1, len(argsList)+2)
	argsList = append(argsList, int64(limit), int64(offset))

	err = r.db.SelectContext(ctx, &mediaList, listQuery, argsList...)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, total, nil
}

func (r *PostgresMediaRepository) ListByType(ctx context.Context, mediaType domain.MediaType, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	countQuery := `SELECT COUNT(*) FROM media WHERE deleted_at IS NULL AND media_type = $1`
	argsCount := []interface{}{mediaType}

	if userID != nil {
		countQuery += ` AND user_id = $2`
		argsCount = append(argsCount, *userID)
	}

	err := r.db.GetContext(ctx, &total, countQuery, argsCount...)
	if err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT * FROM media WHERE deleted_at IS NULL AND media_type = $1`
	argsList := []interface{}{mediaType}

	if userID != nil {
		listQuery += ` AND user_id = $2`
		argsList = append(argsList, *userID)
	}

	listQuery += fmt.Sprintf(` ORDER BY captured_at DESC LIMIT $%d OFFSET $%d`, len(argsList)+1, len(argsList)+2)
	argsList = append(argsList, int64(limit), int64(offset))

	err = r.db.SelectContext(ctx, &mediaList, listQuery, argsList...)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, total, nil
}

// Search searches for media by query string with optional scope and pagination
// ListAll returns all non-deleted media across all users (admin/worker use only)
func (r *PostgresMediaRepository) ListAll(ctx context.Context, limit int, offset int) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	countQuery := `SELECT COUNT(*) FROM media WHERE deleted_at IS NULL`
	err := r.db.GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT * FROM media WHERE deleted_at IS NULL ORDER BY captured_at DESC LIMIT $1 OFFSET $2`
	err = r.db.SelectContext(ctx, &mediaList, listQuery, int64(limit), int64(offset))
	if err != nil {
		return nil, 0, err
	}

	return mediaList, total, nil
}

func (r *PostgresMediaRepository) Search(ctx context.Context, query string, scope string, limit int, offset int, userID *uuid.UUID, startDate string, endDate string) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	// Parse date range
	var startDateParsed, endDateParsed time.Time
	if startDate != "" || endDate != "" {
		var err error
		if startDate != "" && endDate != "" {
			startDateParsed, endDateParsed, err = parseDateRange(startDate + " - " + endDate)
		} else if startDate != "" {
			startDateParsed, err = parseSingleDate(startDate)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to parse start date: %w", err)
			}
			endDateParsed = startDateParsed.Add(24*time.Hour - time.Second)
		} else {
			endDateParsed, err = parseSingleDate(endDate)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to parse end date: %w", err)
			}
			startDateParsed = time.Time{}
		}
		if err != nil && (startDateParsed.IsZero() && endDateParsed.IsZero()) {
			return nil, 0, fmt.Errorf("failed to parse date range: %w", err)
		}
	}

	// Start with base query
	queryStr := `SELECT COUNT(*) FROM media WHERE deleted_at IS NULL`
	args := []interface{}{}
	argIdx := 1

	if userID != nil {
		queryStr += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *userID)
		argIdx++
	}

	// Add date range filters
	if !startDateParsed.IsZero() {
		queryStr += fmt.Sprintf(" AND captured_at >= $%d", argIdx)
		args = append(args, startDateParsed)
		argIdx++
	}
	if !endDateParsed.IsZero() {
		queryStr += fmt.Sprintf(" AND captured_at <= $%d", argIdx)
		args = append(args, endDateParsed)
		argIdx++
	}

	// Add search conditions based on scope
	if query != "" {
		switch scope {
		case "name":
			queryStr += fmt.Sprintf(" AND LOWER(filename) LIKE $%d", argIdx)
			args = append(args, "%"+query+"%")
			argIdx++
		case "tags":
			queryStr += fmt.Sprintf(" AND tags LIKE $%d", argIdx)
			args = append(args, "%"+query+"%")
			argIdx++
		case "location":
			// Search GPS coordinates stored in metadata JSONB column
			queryStr += fmt.Sprintf(` AND (LOWER(metadata->>'gps_latitude') LIKE $%d OR LOWER(metadata->>'gps_longitude') LIKE $%d OR LOWER(metadata->>'gps_altitude') LIKE $%d)`, argIdx, argIdx+1, argIdx+2)
			args = append(args, "%"+query+"%", "%"+query+"%", "%"+query+"%")
			argIdx += 3
		case "all":
			queryStr += fmt.Sprintf(" AND (LOWER(filename) LIKE $%d OR tags LIKE $%d)", argIdx, argIdx+1)
			args = append(args, "%"+query+"%", "%"+query+"%")
			argIdx += 2
		default:
			// Default to searching both name and tags
			queryStr += fmt.Sprintf(" AND (LOWER(filename) LIKE $%d OR tags LIKE $%d)", argIdx, argIdx+1)
			args = append(args, "%"+query+"%", "%"+query+"%")
			argIdx += 2
		}
	}

	// Execute count query
	err := r.db.GetContext(ctx, &total, queryStr, args...)
	if err != nil {
		return nil, 0, err
	}

	// Build the list query
	listQuery := `SELECT * FROM media WHERE deleted_at IS NULL`
	listArgs := []interface{}{}
	listArgIdx := 1

	if userID != nil {
		listQuery += fmt.Sprintf(" AND user_id = $%d", listArgIdx)
		listArgs = append(listArgs, *userID)
		listArgIdx++
	}

	// Add date range filters
	if !startDateParsed.IsZero() {
		listQuery += fmt.Sprintf(" AND captured_at >= $%d", listArgIdx)
		listArgs = append(listArgs, startDateParsed)
		listArgIdx++
	}
	if !endDateParsed.IsZero() {
		listQuery += fmt.Sprintf(" AND captured_at <= $%d", listArgIdx)
		listArgs = append(listArgs, endDateParsed)
		listArgIdx++
	}

	if query != "" {
		switch scope {
		case "name":
			listQuery += fmt.Sprintf(" AND LOWER(filename) LIKE $%d", listArgIdx)
			listArgs = append(listArgs, "%"+query+"%")
			listArgIdx++
		case "tags":
			listQuery += fmt.Sprintf(" AND tags LIKE $%d", listArgIdx)
			listArgs = append(listArgs, "%"+query+"%")
			listArgIdx++
		case "location":
			// Search GPS coordinates stored in metadata JSONB column
			listQuery += fmt.Sprintf(` AND (LOWER(metadata->>'gps_latitude') LIKE $%d OR LOWER(metadata->>'gps_longitude') LIKE $%d OR LOWER(metadata->>'gps_altitude') LIKE $%d)`, listArgIdx, listArgIdx+1, listArgIdx+2)
			listArgs = append(listArgs, "%"+query+"%", "%"+query+"%", "%"+query+"%")
			listArgIdx += 3
		case "all":
			listQuery += fmt.Sprintf(" AND (LOWER(filename) LIKE $%d OR tags LIKE $%d)", listArgIdx, listArgIdx+1)
			listArgs = append(listArgs, "%"+query+"%", "%"+query+"%")
			listArgIdx += 2
		default:
			listQuery += fmt.Sprintf(" AND (LOWER(filename) LIKE $%d OR tags LIKE $%d)", listArgIdx, listArgIdx+1)
			listArgs = append(listArgs, "%"+query+"%", "%"+query+"%")
			listArgIdx += 2
		}
	}

	// Append limit/offset, then use their final positions in the query
	listArgs = append(listArgs, int64(limit), int64(offset))
	listQuery += fmt.Sprintf(" ORDER BY captured_at DESC LIMIT $%d OFFSET $%d", len(listArgs)-1, len(listArgs))

	// Execute list query
	err = r.db.SelectContext(ctx, &mediaList, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}

	return mediaList, total, nil
}

func (r *PostgresMediaRepository) SearchByTags(ctx context.Context, tags string, userID *uuid.UUID) ([]*domain.Media, error) {
	var mediaList []*domain.Media
	query := `SELECT * FROM media WHERE deleted_at IS NULL AND tags LIKE $1`
	args := []interface{}{"%" + tags + "%"}

	if userID != nil {
		query += ` AND user_id = $2`
		args = append(args, *userID)
	}

	err := r.db.SelectContext(ctx, &mediaList, query, args...)
	if err != nil {
		return nil, err
	}
	return mediaList, nil
}

// ListTrashed returns all soft-deleted (trashed) media for a user.
func (r *PostgresMediaRepository) ListTrashed(ctx context.Context, limit int, offset int, userID uuid.UUID) ([]*domain.Media, int, error) {
	var trashedList []*domain.Media
	var total int

	countQuery := `SELECT COUNT(*) FROM media WHERE deleted_at IS NOT NULL AND user_id = $1`
	err := r.db.GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT * FROM media WHERE deleted_at IS NOT NULL AND user_id = $1 ORDER BY deleted_at DESC LIMIT $2 OFFSET $3`
	err = r.db.SelectContext(ctx, &trashedList, listQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return trashedList, total, nil
}

// RestoreMedia restores a soft-deleted media item back to the active library.
func (r *PostgresMediaRepository) RestoreMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE media SET deleted_at = NULL WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("media not found in trash or ownership mismatch (id: %s)", id)
	}

	return nil
}

// GetTrashedMedia retrieves a media item that has been soft-deleted (in trash).
func (r *PostgresMediaRepository) GetTrashedMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Media, error) {
	var media domain.Media
	query := `SELECT * FROM media WHERE deleted_at IS NOT NULL AND id = $1`
	args := []interface{}{id}

	if userID != uuid.Nil {
		query += ` AND user_id = $2`
		args = append(args, userID)
	}

	err := r.db.GetContext(ctx, &media, query, args...)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] GetTrashedMedia %v\n", media)
	return &media, nil
}

// PermanentlyDeleteMedia permanently removes a media item from the database and storage.
func (r *PostgresMediaRepository) PermanentlyDeleteMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	faceQuery := `DELETE FROM faces WHERE media_id = $1`
	if _, err := tx.ExecContext(ctx, faceQuery, id); err != nil {
		return fmt.Errorf("failed to delete associated faces: %w", err)
	}

	deleteMediaQuery := `DELETE FROM media WHERE id = $1 AND user_id = $2`
	result, err := tx.ExecContext(ctx, deleteMediaQuery, id, userID)
	if err != nil {
		return fmt.Errorf("failed to permanently delete media: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("media not found in trash or ownership mismatch (id: %s)", id)
	}

	return tx.Commit()
}