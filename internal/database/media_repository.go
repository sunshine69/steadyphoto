package database

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jbrodriguez/mlog"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresMediaRepository implements the MediaRepository interface using PostgreSQL
type PostgresMediaRepository struct {
	db *sqlx.DB
}

// NewPostgresMediaRepository creates a new instance of PostgresMediaRepository
func NewPostgresMediaRepository(db *sqlx.DB) *PostgresMediaRepository {
	return &PostgresMediaRepository{db: db}
}

// Helper function to parse date range strings in various formats
func parseDateRange(dateRange string) (start time.Time, end time.Time, err error) {
	if dateRange == "" {
		return time.Time{}, time.Time{}, nil
	}

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
		start, err = parseSingleDate(dateRange)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = start.Add(24*time.Hour - time.Second)
	}

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
			if format == "2006" {
				t = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			}
			if format == "2006/01" {
				t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// ParseBoundingboxQuery parses a bounding box query string in format "bounding_box:south,north,west,east"
func ParseBoundingboxQuery(query string) (south, north, west, east float64, ok bool) {
	const prefix = "bounding_box:"
	if !strings.HasPrefix(query, prefix) {
		return 0, 0, 0, 0, false
	}

	values := strings.TrimPrefix(query, prefix)
	parts := strings.Split(values, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, false
	}

	var err error
	south, err = parseFloat(parts[0])
	if err != nil {
		return 0, 0, 0, 0, false
	}
	north, err = parseFloat(parts[1])
	if err != nil {
		return 0, 0, 0, 0, false
	}
	west, err = parseFloat(parts[2])
	if err != nil {
		return 0, 0, 0, 0, false
	}
	east, err = parseFloat(parts[3])
	if err != nil {
		return 0, 0, 0, 0, false
	}

	return south, north, west, east, true
}

// parseFloat is a helper to parse a float64 string
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// HaversineDistance calculates the distance between two points on Earth using the Haversine formula.
// Returns distance in kilometers.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// isGPSCoordinate checks if a value looks like GPS coordinates (lat,lon format)
func isGPSCoordinate(value string) bool {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return false
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var f float64
		_, err := fmt.Sscanf(part, "%f", &f)
		if err != nil {
			return false
		}
	}
	return true
}

// SearchNearPoint finds media within a radius (in km) of a given point
func (r *PostgresMediaRepository) SearchNearPoint(ctx context.Context, lat, lon, radiusKm float64, userID *uuid.UUID, limit, offset int) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	baseQuery := `
		SELECT id, user_id, path, filename, hash, size_bytes, width, height,
		       captured_at, media_type, metadata, video_metadata, created_at, updated_at, tags
		FROM media
		WHERE deleted_at IS NULL
	`

	countQuery := baseQuery
	listQuery := baseQuery + ` ORDER BY captured_at DESC LIMIT $1 OFFSET $2`

	args := []interface{}{}
	argIdx := 1

	if userID != nil {
		countQuery += fmt.Sprintf(" AND user_id = $%d", argIdx)
		listQuery += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *userID)
		argIdx++
	}

	gpsFilter := `AND metadata->>'gps_latitude' IS NOT NULL
	               AND metadata->>'gps_longitude' IS NOT NULL`

	countQuery += gpsFilter
	listQuery += gpsFilter

	latRange := radiusKm / 111.0
	lonRange := radiusKm / (111.0 * math.Cos(lat*math.Pi/180))

	countQuery += fmt.Sprintf(`
		AND CAST(metadata->>'gps_latitude' AS FLOAT) BETWEEN %f AND %f
		AND CAST(metadata->>'gps_longitude' AS FLOAT) BETWEEN %f AND %f
	`, lat-latRange, lat+latRange, lon-lonRange, lon+lonRange)

	listQuery += fmt.Sprintf(`
		AND CAST(metadata->>'gps_latitude' AS FLOAT) BETWEEN %f AND %f
		AND CAST(metadata->>'gps_longitude' AS FLOAT) BETWEEN %f AND %f
	`, lat-latRange, lat+latRange, lon-lonRange, lon+lonRange)

	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count media near point: %w", err)
	}

	listArgs := append(args, int64(limit), int64(offset))
	err = r.db.SelectContext(ctx, &mediaList, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list media near point: %w", err)
	}

	filteredMedia := make([]*domain.Media, 0)
	for _, media := range mediaList {
		gpsLatStr := media.Metadata["gps_latitude"]
		gpsLonStr := media.Metadata["gps_longitude"]

		if gpsLatStr == "" || gpsLonStr == "" {
			continue
		}

		var mediaLat, mediaLon float64
		_, err := fmt.Sscanf(gpsLatStr, "%f", &mediaLat)
		if err != nil {
			continue
		}
		_, err = fmt.Sscanf(gpsLonStr, "%f", &mediaLon)
		if err != nil {
			continue
		}

		distance := HaversineDistance(lat, lon, mediaLat, mediaLon)
		if distance <= radiusKm {
			filteredMedia = append(filteredMedia, media)
		}
	}

	return filteredMedia, len(filteredMedia), nil
}

// SearchWithinBoundingBox finds media within a geographic bounding box
func (r *PostgresMediaRepository) SearchWithinBoundingBox(ctx context.Context, south, north, west, east float64, userID *uuid.UUID, limit, offset int) ([]*domain.Media, int, error) {
	var mediaList []*domain.Media
	var total int

	args := []interface{}{}

	whereClauses := []string{"deleted_at IS NULL"}

	if userID != nil {
		args = append(args, *userID)
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", len(args)))
	}

	whereClauses = append(whereClauses,
		"metadata->>'gps_latitude' IS NOT NULL",
		"metadata->>'gps_longitude' IS NOT NULL")

	args = append(args, south, north, west, east)
	whereClauses = append(whereClauses, fmt.Sprintf("CAST(metadata->>'gps_latitude' AS FLOAT) BETWEEN $%d AND $%d", len(args)-3, len(args)-2))
	whereClauses = append(whereClauses, fmt.Sprintf("CAST(metadata->>'gps_longitude' AS FLOAT) BETWEEN $%d AND $%d", len(args)-1, len(args)))

	whereSQL := strings.Join(whereClauses, " AND ")

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM media WHERE %s", whereSQL)
	err := r.db.GetContext(ctx, &total, countSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count media in bounding box: %w", err)
	}

	listArgs := make([]interface{}, len(args)+2)
	copy(listArgs, args)
	listArgs[len(args)] = int64(limit)
	listArgs[len(args)+1] = int64(offset)

	listSQL := fmt.Sprintf(`
		SELECT id, user_id, path, filename, hash, size_bytes, width, height,
		       captured_at, media_type, metadata, video_metadata, created_at, updated_at, tags
		FROM media
		WHERE %s
		ORDER BY captured_at DESC LIMIT $%d OFFSET $%d`, whereSQL, len(listArgs)-1, len(listArgs))

	err = r.db.SelectContext(ctx, &mediaList, listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list media in bounding box: %w", err)
	}

	return mediaList, total, nil
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
	mlog.Info("[DEBUG] GetById %v\n", media)
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

// buildExpressionWhereClauses generates SQL WHERE clauses from an ExpressionResult.
// Only applies to scopes: all, name, tags.
// The caller passes argIdx which is the current parameter position in the query.
// Returns: (whereClause string, args []interface{}, newArgIdx int, hasTerms bool)
// newArgIdx is the next available parameter position after appending these args.
func buildExpressionWhereClauses(exprResult *domain.ExpressionResult, scope string, argIdx int) (string, []interface{}, int, bool) {
	if exprResult == nil {
		return "", nil, 0, false
	}

	if len(exprResult.IncludeTerms) == 0 && len(exprResult.ExcludeTerms) == 0 {
		return "", nil, 0, false
	}

	// Only apply expression parsing to text-based scopes
	if scope != "all" && scope != "name" && scope != "tags" {
		return "", nil, 0, false
	}

	var whereClauses []string
	var args []interface{}
	currentArg := argIdx

	// Add include terms (AND clauses)
	for _, term := range exprResult.IncludeTerms {
		switch term.FieldScope {
		case "name":
			whereClauses = append(whereClauses, fmt.Sprintf("filename LIKE $%d", currentArg))
			args = append(args, "%"+term.Value+"%")
			currentArg++
		case "tags":
			whereClauses = append(whereClauses, fmt.Sprintf("tags LIKE $%d", currentArg))
			args = append(args, "%"+term.Value+"%")
			currentArg++
		default: // "all" or "" — searches filename, tags, and metadata (including GPS)
			whereClauses = append(whereClauses, fmt.Sprintf("(filename LIKE $%d OR tags LIKE $%d OR metadata::text LIKE $%d)", currentArg, currentArg+1, currentArg+2))
			args = append(args, "%"+term.Value+"%", "%"+term.Value+"%", "%"+term.Value+"%")
			currentArg += 3
		}
	}

	// Add exclude terms (NOT LIKE clauses)
	for _, term := range exprResult.ExcludeTerms {
		switch term.FieldScope {
		case "name":
			whereClauses = append(whereClauses, fmt.Sprintf("filename NOT LIKE $%d", currentArg))
			args = append(args, "%"+term.Value+"%")
			currentArg++
		case "tags":
			whereClauses = append(whereClauses, fmt.Sprintf("tags NOT LIKE $%d", currentArg))
			args = append(args, "%"+term.Value+"%")
			currentArg++
		default: // "all" or "" — excludes from filename, tags, and metadata (including GPS)
			whereClauses = append(whereClauses, fmt.Sprintf("NOT (filename LIKE $%d OR tags LIKE $%d OR metadata::text LIKE $%d)", currentArg, currentArg+1, currentArg+2))
			args = append(args, "%"+term.Value+"%", "%"+term.Value+"%", "%"+term.Value+"%")
			currentArg += 3
		}
	}

	if len(whereClauses) == 0 {
		return "", nil, 0, false
	}

	return strings.Join(whereClauses, " AND "), args, currentArg, true
}

// buildSearchWhereClause constructs the WHERE clause for the search query.
// If exprResult is provided, it generates expression-based filtering.
// Otherwise, it falls back to simple LIKE matching.
func (r *PostgresMediaRepository) buildSearchWhereClause(query string, scope string, argIdx int, exprResult *domain.ExpressionResult) (string, []interface{}, int, error) {
	// If expression result is provided and scope supports it, use expression-based filtering
	if exprResult != nil {
		clauses, args, newArgIdx, hasTerms := buildExpressionWhereClauses(exprResult, scope, argIdx)
		if hasTerms {
			return clauses, args, newArgIdx, nil
		}
		// If no expression terms matched, fall through to simple query
	}

	// Fall back to simple query
	switch scope {
	case "name":
		return fmt.Sprintf("filename LIKE $%d", argIdx), []interface{}{"%" + query + "%"}, argIdx + 1, nil
	case "tags":
		return fmt.Sprintf("tags LIKE $%d", argIdx), []interface{}{"%" + query + "%"}, argIdx + 1, nil
	case "location":
		if strings.Contains(query, ",") {
			parts := strings.SplitN(query, ",", 2)
			if len(parts) == 2 {
				var centerLat, centerLon float64
				_, err1 := fmt.Sscanf(parts[0], "%f", &centerLat)
				if err1 == nil {
					_, err2 := fmt.Sscanf(parts[1], "%f", &centerLon)
					if err2 == nil {
						return "HAVERSINE", []interface{}{centerLat, centerLon, float64(50)}, argIdx + 1, nil
					}
				}
			}
		}
		return fmt.Sprintf(`(LOWER(metadata->>'gps_latitude') LIKE $%d OR LOWER(metadata->>'gps_longitude') LIKE $%d OR LOWER(metadata->>'gps_altitude') LIKE $%d)`, argIdx, argIdx+1, argIdx+2),
			[]interface{}{"%" + query + "%", "%" + query + "%", "%" + query + "%"},
			argIdx + 3, nil
	case "all":
		return fmt.Sprintf("(filename LIKE $%d OR tags LIKE $%d OR metadata::text LIKE $%d)", argIdx, argIdx+1, argIdx+2),
			[]interface{}{"%" + query + "%", "%" + query + "%", "%" + query + "%"},
			argIdx + 3, nil
	default:
		return fmt.Sprintf("(filename LIKE $%d OR tags LIKE $%d)", argIdx, argIdx+1),
			[]interface{}{"%" + query + "%", "%" + query + "%"},
			argIdx + 2, nil
	}
}

// Search searches for media by query string with optional scope and pagination.
// Supports expression-based search with include (+) and exclude (-) terms.
// For scopes "all", "name", "tags": applies expression filtering.
// For scopes "place", "location": GPS/geography-based search (no expression parsing).
func (r *PostgresMediaRepository) Search(ctx context.Context, query string, scope string, limit int, offset int, userID *uuid.UUID, startDate string, endDate string, exprResult *domain.ExpressionResult) ([]*domain.Media, int, error) {
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

	// Check for bounding box query (place search)
	if scope == "location" && query != "" {
		south, north, west, east, ok := ParseBoundingboxQuery(query)
		if ok {
			return r.SearchWithinBoundingBox(ctx, south, north, west, east, userID, limit, offset)
		}
	}

	// Build the base query
	queryStr := `SELECT COUNT(*) FROM media WHERE deleted_at IS NULL`
	args := []interface{}{}
	argIdx := 1

	if userID != nil {
		queryStr += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *userID)
		argIdx++
	}

	// Add date range filters
	if !startDateParsed.IsZero() && !endDateParsed.IsZero() {
		dateFilter := fmt.Sprintf(`
			(
				(metadata->>'DateTimeOriginal' IS NOT NULL AND metadata->>'DateTimeOriginal' != '')
				AND TO_TIMESTAMP(REPLACE(metadata->>'DateTimeOriginal', ':', '/'), 'YYYY/MM/DD HH24:MI:SS') BETWEEN $%d AND $%d
			)
			OR
			(
				(metadata->>'DateTimeOriginal' IS NULL OR metadata->>'DateTimeOriginal' = '')
				AND captured_at BETWEEN $%d AND $%d
			)
		`, argIdx, argIdx+1, argIdx+2, argIdx+3)
		queryStr += " AND (" + dateFilter + ")"
		args = append(args, startDateParsed, endDateParsed, startDateParsed, endDateParsed)
		argIdx += 4
	}

	// Add search conditions
	if query != "" {
		whereClause, searchArgs, newArgIdx, err := r.buildSearchWhereClause(query, scope, argIdx, exprResult)
		if err != nil {
			return nil, 0, err
		}

		// Handle special Haversine search marker
		if whereClause == "HAVERSINE" {
			centerLat := searchArgs[0].(float64)
			centerLon := searchArgs[1].(float64)
			mediaList, total, err = r.SearchNearPoint(ctx, centerLat, centerLon, 50.0, userID, limit, offset)
			if err != nil {
				return nil, 0, err
			}
			return mediaList, total, nil
		}

		queryStr += " AND " + whereClause
		args = append(args, searchArgs...)
		argIdx = newArgIdx
	}

	mlog.Info("[DEBUG] ===== SEARCH SQL (COUNT) =====")
	mlog.Info("[DEBUG] QUERY: %s", queryStr)
	mlog.Info("[DEBUG] ARGS: %v", args)
	mlog.Info("[DEBUG] SCOPE: %s | QUERY: %s", scope, query)

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
	if !startDateParsed.IsZero() && !endDateParsed.IsZero() {
		listQuery += fmt.Sprintf(`
			AND (
				(
					(metadata->>'DateTimeOriginal' IS NOT NULL AND metadata->>'DateTimeOriginal' != '')
					AND to_timestamp(REPLACE(metadata->>'DateTimeOriginal', ':', '/'), 'YYYY/MM/DD HH24:MI:SS') BETWEEN $%d AND $%d
				)
				OR (
					(metadata->>'DateTimeOriginal' IS NULL OR metadata->>'DateTimeOriginal' = '')
					AND captured_at BETWEEN $%d AND $%d
				)
			)
		`, listArgIdx, listArgIdx+1, listArgIdx+2, listArgIdx+3)
		listArgs = append(listArgs, startDateParsed, endDateParsed, startDateParsed, endDateParsed)
		listArgIdx += 4
	}

	if query != "" {
		whereClause, searchArgs, newArgIdx, err := r.buildSearchWhereClause(query, scope, listArgIdx, exprResult)
		if err != nil {
			return nil, 0, err
		}

		// Handle special Haversine search marker
		if whereClause == "HAVERSINE" {
			centerLat := searchArgs[0].(float64)
			centerLon := searchArgs[1].(float64)
			mediaList, total, err = r.SearchNearPoint(ctx, centerLat, centerLon, 50.0, userID, limit, offset)
			if err != nil {
				return nil, 0, err
			}
			return mediaList, total, nil
		}

		listQuery += " AND " + whereClause
		listArgs = append(listArgs, searchArgs...)
		listArgIdx = newArgIdx
	}

	listArgs = append(listArgs, int64(limit), int64(offset))
	listQuery += fmt.Sprintf(" ORDER BY captured_at DESC LIMIT $%d OFFSET $%d", len(listArgs)-1, len(listArgs))

	mlog.Info("[DEBUG] ===== SEARCH SQL (LIST) =====")
	mlog.Info("[DEBUG] QUERY: %s", listQuery)
	mlog.Info("[DEBUG] ARGS: %v", listArgs)
	mlog.Info("[DEBUG] LIMIT: %d | OFFSET: %d", limit, offset)

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
	mlog.Info("[DEBUG] GetTrashedMedia %v\n", media)
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
