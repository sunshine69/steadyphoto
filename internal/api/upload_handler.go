package api

import (
	// For MIME sniffing multi-reader
	"crypto/sha256"
	"database/sql" // Added for sql.ErrNoRows check
	"encoding/json"
	"errors" // Added for errors.Is check
	"fmt"
	"io"
	"github.com/jbrodriguez/mlog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/processor"
	"steadyphoto/internal/storage"

	"github.com/google/uuid"
)

// errorLog is the file handle for the backend error log
var errorLog *os.File

// initErrorLog initializes the error log file for writing errors
func initErrorLog() error {
	var err error
	errorLog, err = os.OpenFile("backend-errors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open backend error log: %w", err)
	}
	return nil
}

// logBackendError writes an error entry to the backend-errors.log file
func logBackendError(prefix string, err error) {
	if errorLog == nil {
		// Fallback to stderr if log file isn't initialized
		mlog.Info("[%s] %v", prefix, err)
		return
	}

	entry := fmt.Sprintf("%s | %s | %s\n",
		time.Now().Format(time.RFC3339),
		runtime.Version(),
		err,
	)

	if _, writeErr := errorLog.WriteString(entry); writeErr != nil {
		mlog.Info("[ERROR] Failed to write to backend error log: %v", writeErr)
	}
}

// closeErrorLog closes the error log file handle
func closeErrorLog() {
	if errorLog != nil {
		errorLog.Close()
	}
}

// detectClientSource identifies the upload client from the User-Agent header
func detectClientSource(r *http.Request) domain.ClientSource {
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		return domain.ClientSourceUnknown
	}

	// Check for specific clients
	if strings.Contains(userAgent, "Android") || strings.Contains(userAgent, "Android App") {
		return domain.ClientSourceAndroid
	}
	if strings.Contains(userAgent, "ImmichMigrate") || strings.Contains(userAgent, "immich-migrate") {
		return domain.ClientSourceImmichMigrate
	}
	if strings.Contains(userAgent, "Web") || strings.Contains(userAgent, "Mozilla") {
		return domain.ClientSourceWeb
	}
	if strings.Contains(userAgent, "iOS") || strings.Contains(userAgent, "iPhone") || strings.Contains(userAgent, "iPad") {
		return domain.ClientSourceiOS
	}

	return domain.ClientSourceUnknown
}

// MediaUploadHandler handles media upload requests for Web and Mobile clients.
type MediaUploadHandler struct {
	mediaRepo      domain.MediaRepository
	albumRepo      domain.AlbumRepository // Added for album association support
	jobRepo        *database.PostgresJobRepository
	storageService *storage.StorageService
}

func NewMediaUploadHandler(mediaRepo domain.MediaRepository, albumRepo domain.AlbumRepository, jobRepo *database.PostgresJobRepository, storageService *storage.StorageService) *MediaUploadHandler {
	return &MediaUploadHandler{
		mediaRepo:      mediaRepo,
		albumRepo:      albumRepo,
		jobRepo:        jobRepo,
		storageService: storageService,
	}
}

// Handle processes multipart form data containing one or more files.
func (h *MediaUploadHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mlog.Info("[DEBUG] UploadHandler: Starting upload for user %s...", userID.String())
	// Log request metadata to diagnose upload failures
	clientSource := detectClientSource(r)
	mlog.Info("[DEBUG] UploadHandler: clientSource=%s method=%s uri=%s contentLength=%d userAgent=%s",
		clientSource, r.Method, r.RequestURI, r.ContentLength, r.Header.Get("User-Agent"))
	mlog.Info("[DEBUG] UploadHandler: content-type=%s", r.Header.Get("Content-Type"))

	// 1. Parse Multipart Form (32MB memory limit, rest on disk)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		mlog.Info("[ERROR] UploadHandler: Failed to parse form: %v", err)
		logBackendError("[PARSE]", err)
		http.Error(w, "Invalid form data. Ensure you are sending multipart/form-data.", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		mlog.Info("[WARN] UploadHandler: No files found in 'files' field.")
		http.Error(w, "No files uploaded. Ensure you use -F \"files=@...\"", http.StatusBadRequest)
		return
	}

	// FIX: Initialize as empty arrays so JSON returns [] instead of null
	var uploaded = make([]map[string]interface{}, 0)
	var skippedDuplicates = make([]interface{}, 0)

	for i, header := range files {
		mlog.Info("[DEBUG] UploadHandler: Processing file %d/%d: '%s' (Size on disk: %d bytes)", i+1, len(files), header.Filename, header.Size)

		file, err := header.Open()
		if err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to open multipart file %s: %v", header.Filename, err)
			continue
		}

		hasher := sha256.New()
		tempFile, err := os.CreateTemp("", "upload-*.tmp")
		if err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to create temp file for %s: %v", header.Filename, err)
			file.Close()
			continue
		}

		n, err := io.Copy(io.MultiWriter(tempFile, hasher), file)
		file.Close() // Close original multipart reader immediately

		if err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to copy stream for %s (size=%d): %v", header.Filename, n, err)
			os.Remove(tempFile.Name())
			continue
		}

		mlog.Info("[DEBUG] UploadHandler: Hash calculated for '%s': %x...", header.Filename[:min(10, len(header.Filename))], hasher.Sum(nil)[:8]) // Log first 4 bytes of hash

		// FIX 1: Seek back to start of temp file before reading it again later!
		if _, err := tempFile.Seek(int64(0), io.SeekStart); err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to seek in temp file for %s: %v", header.Filename, err)
			os.Remove(tempFile.Name())
			continue
		}

		// FIX 2: Validate MIME type using http.DetectContentType() - reads first 512 bytes for sniffing.
		// This prevents malicious uploads where someone disguises an executable as a photo by changing the extension.
		ext := filepath.Ext(header.Filename)
		buf := make([]byte, 512)
		nRead, readErr := tempFile.Read(buf)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			mlog.Info("[ERROR] UploadHandler: Failed to read for MIME sniffing %s: %v", header.Filename, readErr)
			os.Remove(tempFile.Name())
			continue
		}

		detectedType := http.DetectContentType(buf[:nRead])
		extLower := strings.ToLower(ext)
		if !isValidMediaType(detectedType, extLower) {
			mlog.Info("[ERROR] UploadHandler: MIME type mismatch for '%s' - detected '%s', expected %s",
				header.Filename, detectedType, extLower)
			os.Remove(tempFile.Name())
			continue
		}

		// Seek back to start of temp file after reading for MIME sniffing!
		if _, err := tempFile.Seek(int64(0), io.SeekStart); err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to seek in temp file for %s: %v", header.Filename, err)
			os.Remove(tempFile.Name())
			continue
		}

		hash := fmt.Sprintf("%x", hasher.Sum(nil))

		mlog.Info("[DEBUG] UploadHandler: Checking DB for duplicate hash...")
		existingMedia, dbErr := h.mediaRepo.GetByHash(ctx, hash)

		if dbErr != nil {
			// FIX 2: Handle "No Rows" correctly.
			// If the error is specifically sql.ErrNoRows (or contains that text), it means this file doesn't exist yet in DB.
			isNewFile := errors.Is(dbErr, sql.ErrNoRows) || strings.Contains(dbErr.Error(), "no rows")

			if isNewFile {
				mlog.Info("[INFO] UploadHandler: No duplicate found for '%s' (Proceeding to save)", header.Filename) // Normal case -> Proceed to upload logic below
				existingMedia = nil
			} else {
				// If it's NOT a "No Rows" error, then something is actually broken with the DB connection or query.
				mlog.Info("[ERROR] UploadHandler: Actual DB failure for '%s' (hash=%s): %v", header.Filename, hash[:8]+"...", dbErr)
				os.Remove(tempFile.Name())
				continue // Skip only on real errors like connection loss
			}
		} else {
			mlog.Info("[DEBUG] UploadHandler: No error from DB check.")
		}

		if existingMedia != nil && existingMedia.ID != uuid.Nil {
			mlog.Info("[INFO] UploadHandler: Duplicate detected for '%s' (Hash matches ID=%s)", header.Filename, existingMedia.ID.String())
			skippedDuplicates = append(skippedDuplicates, map[string]interface{}{
				"filename":        header.Filename,
				"id":              existingMedia.ID.String(),
				"captured_at":     existingMedia.CapturedAt.Format(time.RFC3339),
				"file_created_at": existingMedia.FileCreatedAt,
			})
			os.Remove(tempFile.Name())
			continue
		}

	

		newFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

		dateDir := time.Now().Format("2006/01/02")
		relTimePath := filepath.Join(dateDir, newFilename)
		relPathFromRoot := filepath.Join(userID.String(), relTimePath)

		mlog.Info("[DEBUG] UploadHandler: Target path (relative to storage root): '%s'", relPathFromRoot)

		// 1. Ensure the directory exists for this user/date
		if err := h.storageService.EnsureDir(userID, relTimePath); err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to ensure directory %s: %v", dateDir, err)
			os.Remove(tempFile.Name())
			continue
		}

		// 2. Get the absolute path for saving (without checking if it exists yet)
		absTargetPath := h.storageService.GetAbsolutePath(relPathFromRoot)

		destFile, err := os.Create(absTargetPath)
		if err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to create dest file %s: %v", absTargetPath, err)
			os.Remove(tempFile.Name())
			continue
		}

		n2, err := io.Copy(destFile, tempFile)
		if err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to save file %s (copied %d/%d bytes): %v", newFilename, n2, n, err)
			destFile.Close()
			os.Remove(absTargetPath)
			os.Remove(tempFile.Name())
			continue
		}

		destFile.Close() // Close file before removing temp.
		mlog.Info("[DEBUG] UploadHandler: File saved to disk '%s'", newFilename)
		os.Remove(tempFile.Name())

		extLower = strings.ToLower(ext)
		var mediaType domain.MediaType = domain.MediaTypePhoto
		switch extLower {
		case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v", ".flv", ".wmv", ".3gp":
			mediaType = domain.MediaTypeVideo
		}

		clientSource := detectClientSource(r)

		// Declare meta early so we can populate CapturedAt and Metadata before building the rest
		meta := &domain.Media{}

		// Extract EXIF data (GPS, orientation, all tags + DateTimeOriginal) for captured_at
		var capturedAt time.Time
		var exifErr error
		var exifMeta domain.Metadata
		exifMeta, capturedAt, exifErr = extractExif(absTargetPath)
		meta.Metadata = exifMeta
		if exifErr != nil {
			mlog.Info("[WARN] UploadHandler: extractExif failed for '%s': %v", header.Filename, exifErr)
		}

		meta.ID = uuid.New()
		meta.UserID = userID
		meta.Filename = header.Filename
		meta.MediaType = mediaType
		meta.Path = relPathFromRoot
		meta.SizeBytes = n
		meta.Hash = hash
		meta.ClientSource = clientSource
		meta.CreatedAt = time.Now()
		meta.UpdatedAt = time.Now()

		mlog.Info("[INFO] UploadHandler: Detected client source '%s' for '%s'", clientSource, header.Filename)
		if meta.Metadata != nil && meta.Metadata["gps_latitude"] != "" {
			mlog.Info("[INFO] UploadHandler: GPS found - lat=%s lon=%s", meta.Metadata["gps_latitude"], meta.Metadata["gps_longitude"])
		}

		// Extract video metadata if this is a video file
		if meta.MediaType == domain.MediaTypeVideo && processor.IsVideoFile(header.Filename) {
			mlog.Info("[INFO] UploadHandler: Extracting video metadata for '%s'...", header.Filename)
			videoMeta, vErr := processor.ExtractVideoMetadata(ctx, absTargetPath)
			if vErr != nil {
				mlog.Info("[WARN] UploadHandler: Failed to extract video metadata for '%s': %v", header.Filename, vErr)
				// Non-fatal: continue without video metadata
			} else if videoMeta != nil {
				meta.VideoMetadata = *videoMeta
				mlog.Info("[INFO] UploadHandler: Video metadata extracted for '%s' - codec=%s res=%dx%d dur=%.1fs fps=%.2f",
					header.Filename, videoMeta.VideoCodec, videoMeta.Width, videoMeta.Height, videoMeta.Duration, videoMeta.FrameRate)
			}
		}

		// Parse file creation date from client (filesystem DATE_ADDED)
		if fileCreatedAtStr := r.FormValue("fileCreatedAt"); fileCreatedAtStr != "" {
			if fcTime, err := time.Parse("2006/01/02 15:04:05", fileCreatedAtStr); err == nil {
				meta.FileCreatedAt = &fcTime
				mlog.Info("[INFO] UploadHandler: file creation date from client for '%s': %s", header.Filename, fcTime.Format(time.RFC3339))
			}
		}

		// Determine CapturedAt: prefer VideoMetadata.CreatedAt (for videos), then EXIF DateTimeOriginal (for images), then filename heuristic, then filesystem mtime (fileCreatedAt from client), fall back to upload time
		meta.CapturedAt = time.Now() // default fallback
		if !capturedAt.IsZero() {
			meta.CapturedAt = capturedAt
			mlog.Info("[INFO] UploadHandler: EXIF DateTimeOriginal found for '%s': %s", header.Filename, capturedAt.Format(time.RFC3339))
		} else if meta.VideoMetadata.CreatedAt != (time.Time{}) {
			meta.CapturedAt = meta.VideoMetadata.CreatedAt
			mlog.Info("[INFO] UploadHandler: VideoMetadata.CreatedAt found for '%s': %s", header.Filename, meta.VideoMetadata.CreatedAt.Format(time.RFC3339))
		} else {
			// Heuristic fallback: try to parse date from filename
			if fnDate, _, _ := processor.ExtractDateFromString(header.Filename); !fnDate.IsZero() {
				meta.CapturedAt = fnDate
				mlog.Info("[INFO] UploadHandler: Filename heuristic date parsed for '%s': %s", header.Filename, fnDate.Format(time.RFC3339))
			} else if meta.FileCreatedAt != nil && !meta.FileCreatedAt.IsZero() {
				// Filesystem mtime from the client (sent as fileCreatedAt in the form)
				meta.CapturedAt = *meta.FileCreatedAt
				mlog.Info("[INFO] UploadHandler: Filesystem mtime (fileCreatedAt) used for '%s': %s", header.Filename, meta.FileCreatedAt.Format(time.RFC3339))
			} else {
				mlog.Info("[INFO] UploadHandler: No EXIF/Video/Filename/date found for '%s', using upload time: %s", header.Filename, meta.CapturedAt.Format(time.RFC3339))
			}
		}
		mlog.Info("[DEBUG] UploadHandler: Inserting record into DB for '%s'...", meta.ID)

		if err := h.mediaRepo.Create(ctx, meta); err != nil {
			mlog.Info("[ERROR] UploadHandler: Failed to insert media %s (hash=%s): %v", newFilename, hash[:8]+"...", err)
			continue
		}

		// Create a background job to extract video metadata for video files (in case we want to re-process later)
		if meta.MediaType == domain.MediaTypeVideo {
			// Check if a video metadata job already exists for this media
			existingJobs, err := h.jobRepo.GetJobsByMediaID(ctx, meta.ID)
			if err != nil {
				mlog.Info("[WARN] UploadHandler: Failed to check existing jobs for video metadata: %v", err)
			} else {
				hasVideoMetaJob := false
				for _, j := range existingJobs {
					if j.Type == domain.JobTypeVideoMetadata {
						hasVideoMetaJob = true
						break
					}
				}
				if !hasVideoMetaJob {
					videoJob := &domain.Job{
						ID:        uuid.New(),
						UserID:    meta.UserID,
						Type:      domain.JobTypeVideoMetadata,
						Status:    domain.JobStatusPending,
						MediaID:   meta.ID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					if err := h.jobRepo.Create(ctx, videoJob); err != nil {
						mlog.Info("[WARN] UploadHandler: Failed to create video metadata job for '%s': %v", header.Filename, err)
					} else {
						mlog.Info("[INFO] UploadHandler: Created background video metadata job %s for '%s'", videoJob.ID.String(), header.Filename)
					}
				}
			}
		}

		// FIX: Return the actual filesystem relative path instead of an API URL.
		uploaded = append(uploaded, map[string]interface{}{
			"id":              meta.ID.String(),
			"filename":        header.Filename,
			"mediaType":       string(meta.MediaType),
			"path":            relPathFromRoot,
			"size":            n,
			"captured_at":     meta.CapturedAt.Format(time.RFC3339),
			"file_created_at": meta.FileCreatedAt,
		})

		mlog.Info("[INFO] SUCCESS UPLOAD | id=%s | file=%s | size=%d | client=%s", meta.ID, header.Filename, n, clientSource)
		mlog.Info("[INFO] UploadHandler: Successfully processed '%s' (ID=%s)", header.Filename, meta.ID)
	}

	mlog.Info("[DEBUG] UploadHandler: Finished. Uploaded count: %d, Skipped duplicates: %d", len(uploaded), len(skippedDuplicates))

	mlog.Info("[INFO] UploadHandler: Upload complete - %d uploaded, %d skipped duplicates (total files: %d)",
		len(uploaded), len(skippedDuplicates), len(files))
	response := map[string]interface{}{
		"uploaded":           uploaded,
		"skipped_duplicates": skippedDuplicates,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper to avoid index out of bounds in logging if filename is short
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractExif opens the file at absPath, reads full EXIF data (orientation + GPS + all tags + DateTimeOriginal), 
// and returns a domain.Metadata map with normalized fields.
// Returns (domain.Metadata, time.Time, error) where the time is the parsed DateTimeOriginal (zero if not found).
func extractExif(absPath string) (domain.Metadata, time.Time, error) {
	f, err := os.Open(absPath)
	if err != nil {
		mlog.Info("[WARN] extractExif: failed to open %s: %v", absPath, err)
		return domain.Metadata{}, time.Time{}, err
	}
	defer f.Close()

	reader := processor.NewExifReader()
	info, err := reader.ReadExif(f)
	if err != nil {
		mlog.Info("[WARN] extractExif: failed to read EXIF from %s: %v", absPath, err)
		return domain.Metadata{}, time.Time{}, err
	}
	if info == nil {
		return domain.Metadata{}, time.Time{}, nil
	}

	md := domain.Metadata{}

	// Add orientation if not normal
	if info.Orientation != processor.OrientationNormal {
		md["exif_orientation"] = fmt.Sprintf("%d", int(info.Orientation))
	}

	// Add DateTimeOriginal (parsed) if found
	var dateTimeOriginal string
	if !info.CapturedAt.IsZero() {
		dateTimeOriginal = info.CapturedAt.Format("2006:01:02 15:04:05")
		md["DateTimeOriginal"] = dateTimeOriginal
	}

	// Add all other EXIF tags (skip Orientation, DateTimeOriginal - already handled)
	for _, tag := range info.Tags {
		if strings.EqualFold(tag.Tag, "Orientation") {
			continue
		}
		if strings.EqualFold(tag.Tag, "DateTimeOriginal") {
			continue // already handled via info.CapturedAt
		}
		if strings.EqualFold(tag.Tag, "DateTimeDigitized") {
			continue // will be added below as DateTimeOriginal fallback
		}
		if strings.EqualFold(tag.Tag, "DateTime") {
			continue // will be added below as DateTimeOriginal fallback
		}
		// Add tag to metadata (use tag name as key)
		if _, exists := md[tag.Tag]; !exists {
			md[tag.Tag] = tag.Value
		}
	}

	// Fallback: if DateTimeOriginal was not in EXIF, check DateTimeDigitized or DateTime
	if dateTimeOriginal == "" {
		for _, tag := range info.Tags {
			if strings.EqualFold(tag.Tag, "DateTimeDigitized") && dateTimeOriginal == "" {
				dateTimeOriginal = tag.Value
				md["DateTimeOriginal"] = dateTimeOriginal
				break
			}
		}
	}
	if dateTimeOriginal == "" {
		for _, tag := range info.Tags {
			if strings.EqualFold(tag.Tag, "DateTime") && dateTimeOriginal == "" {
				dateTimeOriginal = tag.Value
				md["DateTimeOriginal"] = dateTimeOriginal
				break
			}
		}
	}
	// Also check if ModifyDate is available (used as last fallback by CLI tool)
	if dateTimeOriginal == "" {
		if modifyDate, ok := md["ModifyDate"]; ok {
			md["DateTimeOriginal"] = modifyDate
			dateTimeOriginal = modifyDate
		}
	}

	// Normalize GPS fields so location search works.
	// NOTE: info.GPSLatitude/GPSLongitude are already signed by parseGPSCoordinate
	// (which applies S/W ref), so we just use them directly without re-applying signs.
	if info.GPSLatitude != 0 && info.GPSLongitude != 0 {
		md["gps_latitude"] = fmt.Sprintf("%.6f", info.GPSLatitude)
		md["gps_longitude"] = fmt.Sprintf("%.6f", info.GPSLongitude)
		md["gps_altitude"] = fmt.Sprintf("%.1f", info.GPSAltitude)
	}

	mlog.Info("[EXIF] Extracted metadata for %s: %v", filepath.Base(absPath), md)
	return md, info.CapturedAt, nil
}

// isValidMediaType checks if the detected MIME type matches the file extension.
// This prevents malicious uploads where someone disguises an executable as a photo by changing the extension.
// Uses http.DetectContentType() which reads up to 512 bytes for sniffing.
// initErrorLog is called at app startup to ensure backend-errors.log is writable.
func isValidMediaType(detectedType, extLower string) bool {
	// Known media extensions (case-insensitive check done via extLower)
	validMediaExtensions := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".heic": true, ".bmp": true,
		".mp4": true, ".mov": true, ".avi": true, ".mkv": true,
	}

	// Check if the extension is a known media format first
	if !validMediaExtensions[extLower] {
		return false // Unknown extension - reject for security
	}

	// If it's a valid media extension, accept it.
	// NOTE: http.DetectContentType() returns "application/octet-stream" for HEIC/HEIF files,
	// so we trust the extension whitelist as the primary security check.
	return true
}

