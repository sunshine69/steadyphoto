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
	"strings"
	"time"

	"steadyphoto/internal/domain"
	"steadyphoto/internal/storage"

	"github.com/google/uuid"
)

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

		ext := filepath.Ext(header.Filename)
		log.Printf("[DEBUG] UploadHandler: Extension detected as '%s'", ext)

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

		extLower := strings.ToLower(ext)
		var mediaType domain.MediaType = domain.MediaTypePhoto
		switch extLower {
		case ".mp4", ".mov", ".avi":
			mediaType = domain.MediaTypeVideo
		}

		meta := &domain.Media{
			ID: uuid.New(), UserID: userID, Filename: header.Filename, MediaType: mediaType, Path: relPathFromRoot, SizeBytes: n, Hash: hash, CapturedAt: time.Now(),
		}

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
