package api

import (
	// For MIME sniffing multi-reader
	"crypto/sha256"
	"database/sql" // Added for sql.ErrNoRows check
	"encoding/json"
	"errors" // Added for errors.Is check
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"steadyphoto/internal/domain"
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
		log.Printf("[%s] %v", prefix, err)
		return
	}

	entry := fmt.Sprintf("%s | %s | %s\n",
		time.Now().Format(time.RFC3339),
		runtime.Version(),
		err,
	)

	if _, writeErr := errorLog.WriteString(entry); writeErr != nil {
		log.Printf("[ERROR] Failed to write to backend error log: %v", writeErr)
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
	storageService *storage.StorageService
}

func NewMediaUploadHandler(mediaRepo domain.MediaRepository, albumRepo domain.AlbumRepository, storageService *storage.StorageService) *MediaUploadHandler {
	return &MediaUploadHandler{
		mediaRepo:      mediaRepo,
		albumRepo:      albumRepo,
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

	log.Printf("[DEBUG] UploadHandler: Starting upload for user %s...", userID.String())

	// 1. Parse Multipart Form (32MB memory limit, rest on disk)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Printf("[ERROR] UploadHandler: Failed to parse form: %v", err)
		logBackendError("[PARSE]", err)
		http.Error(w, "Invalid form data. Ensure you are sending multipart/form-data.", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		log.Printf("[WARN] UploadHandler: No files found in 'files' field.")
		http.Error(w, "No files uploaded. Ensure you use -F \"files=@...\"", http.StatusBadRequest)
		return
	}

	// FIX: Initialize as empty arrays so JSON returns [] instead of null
	var uploaded = make([]map[string]interface{}, 0)
	var skippedDuplicates = make([]interface{}, 0)

	for i, header := range files {
		log.Printf("[DEBUG] UploadHandler: Processing file %d/%d: '%s' (Size on disk: %d bytes)", i+1, len(files), header.Filename, header.Size)

		file, err := header.Open()
		if err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to open multipart file %s: %v", header.Filename, err)
			continue
		}

		hasher := sha256.New()
		tempFile, err := os.CreateTemp("", "upload-*.tmp")
		if err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to create temp file for %s: %v", header.Filename, err)
			file.Close()
			continue
		}

		n, err := io.Copy(io.MultiWriter(tempFile, hasher), file)
		file.Close() // Close original multipart reader immediately

		if err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to copy stream for %s (size=%d): %v", header.Filename, n, err)
			os.Remove(tempFile.Name())
			continue
		}

		log.Printf("[DEBUG] UploadHandler: Hash calculated for '%s': %x...", header.Filename[:min(10, len(header.Filename))], hasher.Sum(nil)[:8]) // Log first 4 bytes of hash

		// FIX 1: Seek back to start of temp file before reading it again later!
		if _, err := tempFile.Seek(int64(0), io.SeekStart); err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to seek in temp file for %s: %v", header.Filename, err)
			os.Remove(tempFile.Name())
			continue
		}

		// FIX 2: Validate MIME type using http.DetectContentType() - reads first 512 bytes for sniffing.
		// This prevents malicious uploads where someone disguises an executable as a photo by changing the extension.
		ext := filepath.Ext(header.Filename)
		buf := make([]byte, 512)
		nRead, readErr := tempFile.Read(buf)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			log.Printf("[ERROR] UploadHandler: Failed to read for MIME sniffing %s: %v", header.Filename, readErr)
			os.Remove(tempFile.Name())
			continue
		}

		detectedType := http.DetectContentType(buf[:nRead])
		extLower := strings.ToLower(ext)
		if !isValidMediaType(detectedType, extLower) {
			log.Printf("[ERROR] UploadHandler: MIME type mismatch for '%s' - detected '%s', expected %s",
				header.Filename, detectedType, extLower)
			os.Remove(tempFile.Name())
			continue
		}

		// Seek back to start of temp file after reading for MIME sniffing!
		if _, err := tempFile.Seek(int64(0), io.SeekStart); err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to seek in temp file for %s: %v", header.Filename, err)
			os.Remove(tempFile.Name())
			continue
		}

		hash := fmt.Sprintf("%x", hasher.Sum(nil))

		log.Printf("[DEBUG] UploadHandler: Checking DB for duplicate hash...")
		existingMedia, dbErr := h.mediaRepo.GetByHash(ctx, hash)

		if dbErr != nil {
			// FIX 2: Handle "No Rows" correctly.
			// If the error is specifically sql.ErrNoRows (or contains that text), it means this file doesn't exist yet in DB.
			isNewFile := errors.Is(dbErr, sql.ErrNoRows) || strings.Contains(dbErr.Error(), "no rows")

			if isNewFile {
				log.Printf("[INFO] UploadHandler: No duplicate found for '%s' (Proceeding to save)", header.Filename) // Normal case -> Proceed to upload logic below
				existingMedia = nil
			} else {
				// If it's NOT a "No Rows" error, then something is actually broken with the DB connection or query.
				log.Printf("[ERROR] UploadHandler: Actual DB failure for '%s' (hash=%s): %v", header.Filename, hash[:8]+"...", dbErr)
				os.Remove(tempFile.Name())
				continue // Skip only on real errors like connection loss
			}
		} else {
			log.Printf("[DEBUG] UploadHandler: No error from DB check.")
		}

		if existingMedia != nil && existingMedia.ID != uuid.Nil {
			log.Printf("[INFO] UploadHandler: Duplicate detected for '%s' (Hash matches ID=%s)", header.Filename, existingMedia.ID.String())
			skippedDuplicates = append(skippedDuplicates, map[string]interface{}{
				"filename": header.Filename,
				"id":       existingMedia.ID.String(),
			})
			os.Remove(tempFile.Name())
			continue
		}

	

		newFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

		dateDir := time.Now().Format("2006/01/02")
		relTimePath := filepath.Join(dateDir, newFilename)
		relPathFromRoot := filepath.Join(userID.String(), relTimePath)

		log.Printf("[DEBUG] UploadHandler: Target path (relative to storage root): '%s'", relPathFromRoot)

		// 1. Ensure the directory exists for this user/date
		if err := h.storageService.EnsureDir(userID, relTimePath); err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to ensure directory %s: %v", dateDir, err)
			os.Remove(tempFile.Name())
			continue
		}

		// 2. Get the absolute path for saving (without checking if it exists yet)
		absTargetPath := h.storageService.GetAbsolutePath(relPathFromRoot)

		destFile, err := os.Create(absTargetPath)
		if err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to create dest file %s: %v", absTargetPath, err)
			os.Remove(tempFile.Name())
			continue
		}

		n2, err := io.Copy(destFile, tempFile)
		if err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to save file %s (copied %d/%d bytes): %v", newFilename, n2, n, err)
			destFile.Close()
			os.Remove(absTargetPath)
			os.Remove(tempFile.Name())
			continue
		}

		destFile.Close() // Close file before removing temp.
		log.Printf("[DEBUG] UploadHandler: File saved to disk '%s'", newFilename)
		os.Remove(tempFile.Name())

		extLower = strings.ToLower(ext)
		var mediaType domain.MediaType = domain.MediaTypePhoto
		switch extLower {
		case ".mp4", ".mov", ".avi":
			mediaType = domain.MediaTypeVideo
		}

		clientSource := detectClientSource(r)
		meta := &domain.Media{
			ID: uuid.New(), UserID: userID, Filename: header.Filename, MediaType: mediaType, Path: relPathFromRoot, SizeBytes: n, Hash: hash, CapturedAt: time.Now(), ClientSource: clientSource,
		}

		log.Printf("[INFO] UploadHandler: Detected client source '%s' for '%s'", clientSource, header.Filename)
		log.Printf("[DEBUG] UploadHandler: Inserting record into DB for '%s'...", meta.ID)

		if err := h.mediaRepo.Create(ctx, meta); err != nil {
			log.Printf("[ERROR] UploadHandler: Failed to insert media %s (hash=%s): %v", newFilename, hash[:8]+"...", err)
			continue
		}

		// FIX: Return the actual filesystem relative path instead of an API URL.
		uploaded = append(uploaded, map[string]interface{}{
			"id":          meta.ID.String(),
			"filename":    header.Filename,
			"mediaType":   string(meta.MediaType),
			"path":        relPathFromRoot, // Use the actual relative path stored in DB for filesystem resolution.
			"size":        n,
			"captured_at": time.Now().Format(time.RFC3339),
		})

		log.Printf("[INFO] UploadHandler: Successfully processed '%s' (ID=%s)", header.Filename, meta.ID)
	}

	log.Printf("[DEBUG] UploadHandler: Finished. Uploaded count: %d, Skipped duplicates: %d", len(uploaded), len(skippedDuplicates))

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

// isValidMediaType checks if the detected MIME type matches the file extension.
// This prevents malicious uploads where someone disguises an executable as a photo by changing the extension.
// Uses http.DetectContentType() which reads up to 512 bytes for sniffing.
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

	// If it's a valid media extension, check that the detected MIME type
	// matches what we expect (image/* or video/*)
	switch {
	case strings.HasPrefix(detectedType, "image/"):
		return true // Valid image format with known extension
	case strings.HasPrefix(detectedType, "video/"):
		return true // Valid video format with known extension
	default:
		// MIME type doesn't match the expected category (e.g., detected as text/html but .jpg)
		log.Printf("[WARN] UploadHandler: MIME mismatch for '%s' - detected '%s', allowed extensions include %s",
			extLower, detectedType, extLower)
		return false // Reject suspicious files like exe disguised as jpg
	}
}

