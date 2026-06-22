package api

import (
	"crypto/sha256"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"steadyphoto/internal/domain"
	"steadyphoto/internal/storage"

	"github.com/google/uuid"
)

// UploadSession represents an in-progress resumable upload session.
type UploadSession struct {
	ID           string         `json:"uploadId"`
	Filename     string         `json:"fileName"`
	FileSize     int64          `json:"fileSize"`
	UploadedChunks []int        `json:"uploadedChunks"`
	TotalChunks  int            `json:"totalChunks"`
	CreatedAt    time.Time      `json:"createdAt"`
	UserID       uuid.UUID      `json:"-"` // Not serialized, used internally for ownership check
	mu           sync.Mutex     // Protects UploadedChunks
	tempPath     string         // Path to the temp file being assembled
	IsComplete   bool           `json:"isComplete,omitempty"` // Whether all chunks have been uploaded
}

// UploadSessionManager manages resumable upload sessions.
type UploadSessionManager struct {
	sessions       map[string]*UploadSession
	mu             sync.RWMutex
	storageService *storage.StorageService
	sessionFile    string // Path to JSON file for session persistence
}

func NewUploadSessionManager(storageService *storage.StorageService) *UploadSessionManager {
	manager := &UploadSessionManager{
		sessions: make(map[string]*UploadSession),
		storageService: storageService,
		// Store sessions in a subdirectory of the upload temp dir for persistence across restarts
		sessionFile: filepath.Join(storageService.GetUploadTempDir(), ".sessions.json"),
	}

	// Load existing sessions from disk on startup (for recovery after server restart)
	manager.loadSessions()

	return manager
}

// loadSessions loads active sessions from the JSON file on disk.
func (m *UploadSessionManager) loadSessions() {
	data, err := os.ReadFile(m.sessionFile)
	if err != nil {
		log.Printf("[INFO] UploadSessionManager: No existing session file found (%v). Starting fresh.", err)
		return
	}

	var sessions map[string]*UploadSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		log.Printf("[ERROR] UploadSessionManager: Failed to parse session file: %v. Discarding corrupt data.", err)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for id, session := range sessions {
		// Verify the temp directory still exists and has all chunk files
		tempDir := m.storageService.GetUploadTempDir()
		allChunksExist := true
		for i := 0; i < session.TotalChunks && session.TotalChunks > 1; i++ {
			chunkPath := filepath.Join(tempDir, id+fmt.Sprintf("_%d.tmp", i))
			if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
				allChunksExist = false
				log.Printf("[WARN] UploadSessionManager: Missing chunk %d for session %s. Marking as incomplete.", i, id)
				break
			}
		}

		m.sessions[id] = session
		if !allChunksExist {
			session.IsComplete = false // Force incomplete if chunks are missing
		}
		log.Printf("[INFO] UploadSessionManager: Recovered session %s (%d/%d chunks uploaded)", id, len(session.UploadedChunks), session.TotalChunks)
	}

	log.Printf("[INFO] UploadSessionManager: Loaded %d sessions from disk", len(m.sessions))
}

// saveSessions persists all active sessions to the JSON file on disk.
func (m *UploadSessionManager) saveSessions() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.Marshal(m.sessions)
	if err != nil {
		log.Printf("[ERROR] UploadSessionManager: Failed to serialize sessions for persistence: %v", err)
		return
	}

	tempFile := m.sessionFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		log.Printf("[ERROR] UploadSessionManager: Failed to write session file: %v", err)
		return
	}

	// Atomically rename temp file to actual file (prevents corruption on crash during write)
	if err := os.Rename(tempFile, m.sessionFile); err != nil {
		log.Printf("[ERROR] UploadSessionManager: Failed to rename session file: %v", err)
		os.Remove(tempFile) // Clean up if rename fails
	}
}

// SessionWithStatus wraps a session with additional status information for the client.
type SessionWithStatus struct {
	ID           string         `json:"uploadId"`
	Filename     string         `json:"fileName"`
	FileSize     int64          `json:"fileSize"`
	UploadedChunks []int        `json:"uploadedChunks"`
	TotalChunks  int            `json:"totalChunks"`
	CreatedAt    time.Time      `json:"createdAt"`
	IsComplete   bool           `json:"isComplete"`
}

// GetSession retrieves a session by ID. Returns nil if not found.
func (m *UploadSessionManager) GetSession(id string) *UploadSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

// CreateSession creates a new upload session and returns it.
func (m *UploadSessionManager) CreateSession(userID uuid.UUID, filename string, fileSize int64, totalChunks int) *UploadSession {
	session := &UploadSession{
		ID:           uuid.New().String(),
		Filename:     filename,
		FileSize:     fileSize,
		TotalChunks:  totalChunks,
		CreatedAt:    time.Now(),
		UserID:       userID,
		UploadedChunks: make([]int, 0),
	}

	// Create temp file for assembling chunks
	session.tempPath = filepath.Join(m.storageService.GetUploadTempDir(), session.ID+".tmp")

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	return session
}

// AddChunk records a chunk as uploaded and returns the updated list of uploaded chunks.
func (m *UploadSessionManager) AddChunk(sessionID string, chunkIndex int) []int {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.sessions[sessionID]
	if session == nil {
		return nil
	}

	// Check if already recorded to avoid duplicates
	for _, idx := range session.UploadedChunks {
		if idx == chunkIndex {
			return session.UploadedChunks
		}
	}

	session.UploadedChunks = append(session.UploadedChunks, chunkIndex)
	return session.UploadedChunks
}

// IsComplete checks if all chunks have been uploaded for this session.
func (m *UploadSessionManager) IsComplete(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session := m.sessions[sessionID]
	if session == nil || len(session.UploadedChunks) != session.TotalChunks {
		return false
	}

	// Verify all chunks are present (0 to TotalChunks-1)
	for i := 0; i < session.TotalChunks; i++ {
		found := false
		for _, idx := range session.UploadedChunks {
			if idx == i {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// MarkComplete marks a session as complete and saves it to disk for persistence.
func (m *UploadSessionManager) MarkComplete(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.sessions[sessionID]
	if session == nil || len(session.UploadedChunks) != session.TotalChunks {
		return false
	}

	session.IsComplete = true
	m.saveSessions() // Persist the complete status to disk
	return true
}

// GetUploadTempDir returns the temporary upload directory path.
func (m *UploadSessionManager) GetUploadTempDir() string {
	return m.storageService.GetUploadTempDir()
}

// DeleteSession removes a session after completion or abort.
func (m *UploadSessionManager) DeleteSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.sessions[sessionID]
	if session != nil && session.tempPath != "" {
		os.Remove(session.tempPath) // Clean up temp file
	}
	delete(m.sessions, sessionID)
}

// CleanupExpiredSessions removes sessions older than 24 hours.
func (m *UploadSessionManager) CleanupExpiredSessions() int {
	now := time.Now().Add(-24 * time.Hour)
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, session := range m.sessions {
		if session.CreatedAt.Before(now) {
			os.Remove(session.tempPath) // Clean up temp file
			delete(m.sessions, id)
			count++
		}
	}
	return count
}

// MediaUploadHandlerSingle handles single-file upload requests for mobile clients.
type MediaUploadHandlerSingle struct {
	mediaRepo      domain.MediaRepository
	storageService *storage.StorageService
	sessionManager *UploadSessionManager
}

func NewMediaUploadHandlerSingle(mediaRepo domain.MediaRepository, storageService *storage.StorageService, sessionManager *UploadSessionManager) *MediaUploadHandlerSingle {
	return &MediaUploadHandlerSingle{
		mediaRepo:      mediaRepo,
		storageService: storageService,
		sessionManager: sessionManager,
	}
}

// HandleSingleFileUpload processes a single file upload with progress tracking.
func (h *MediaUploadHandlerSingle) HandleSingleFileUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Add a database context timeout to prevent DB queries from hanging indefinitely.
	// The HTTP server already has a 5-minute write timeout, but individual DB operations
	// should not block for the entire duration if they're stuck (e.g., lock contention).
	dbCtx, dbCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dbCancel()

	userID, ok := GetUserIDFromContext(dbCtx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	log.Printf("[DEBUG] UploadHandlerSingle: Starting single file upload for user %s...", userID.String())

	err := r.ParseMultipartForm(1 << 30) // 1GB memory limit - allows large single-file uploads without chunking
	if err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to parse form: %v", err)
		http.Error(w, "Invalid form data.", http.StatusBadRequest)
		return
	}

	// Get file from the 'file' field (not 'files')
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded. Ensure you use -F \"file=@...\"", http.StatusBadRequest)
		return
	}

	header := files[0]
	fileName := r.FormValue("fileName")
	_ = fileName // mimeType is not used currently - it could be added later for validation
	fileSizeStr := r.FormValue("fileSize")
	var fileSize int64 = header.Size
	if fileSizeStr != "" {
		fmt.Sscanf(fileSizeStr, "%d", &fileSize)
	}

	log.Printf("[DEBUG] UploadHandlerSingle: Processing file '%s' (size=%d bytes)", fileName, fileSize)

	file, err := header.Open()
	if err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to open multipart file %s: %v", fileName, err)
		http.Error(w, "Failed to read uploaded file.", http.StatusBadRequest)
		return
	}

	hasher := sha256.New()
	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to create temp file for %s: %v", fileName, err)
		file.Close()
		http.Error(w, "Failed to process file.", http.StatusInternalServerError)
		return
	}

	n, err := io.Copy(io.MultiWriter(tempFile, hasher), file)
	file.Close()

	if err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to copy stream for %s (size=%d): %v", fileName, n, err)
		os.Remove(tempFile.Name())
		http.Error(w, "Failed to process uploaded file.", http.StatusInternalServerError)
		return
	}

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to seek in temp file for %s: %v", fileName, err)
		os.Remove(tempFile.Name())
		http.Error(w, "Failed to process uploaded file.", http.StatusInternalServerError)
		return
	}

	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Check for duplicate by hash
	existingMedia, dbErr := h.mediaRepo.GetByHash(dbCtx, hash)
	if dbErr != nil && !errors.Is(dbErr, sql.ErrNoRows) && !strings.Contains(dbErr.Error(), "no rows") {
		log.Printf("[ERROR] UploadHandlerSingle: DB error checking duplicate for %s (hash=%s): %v", fileName, hash[:8]+"...", dbErr)
		os.Remove(tempFile.Name())
		http.Error(w, "Database error.", http.StatusInternalServerError)
		return
	}

	if existingMedia != nil && existingMedia.ID != uuid.Nil {
		log.Printf("[INFO] UploadHandlerSingle: Duplicate detected for '%s' (Hash matches ID=%s)", fileName, existingMedia.ID.String())
		response := map[string]interface{}{
			"uploaded": []interface{}{},
			"skipped_duplicates": []map[string]string{{
				"filename": fileName,
				"id":       existingMedia.ID.String(),
			}},
			"errors": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		os.Remove(tempFile.Name())
		return
	}

	ext := filepath.Ext(fileName)
	newFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	dateDir := time.Now().Format("2006/01/02")
	relTimePath := filepath.Join(dateDir, newFilename)
	relPathFromRoot := filepath.Join(userID.String(), relTimePath)

	if err := h.storageService.EnsureDir(userID, relTimePath); err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to ensure directory %s: %v", dateDir, err)
		os.Remove(tempFile.Name())
		http.Error(w, "Failed to prepare storage.", http.StatusInternalServerError)
		return
	}

	absTargetPath := h.storageService.GetAbsolutePath(relPathFromRoot)

	destFile, err := os.Create(absTargetPath)
	if err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to create dest file %s: %v", absTargetPath, err)
		os.Remove(tempFile.Name())
		http.Error(w, "Failed to save uploaded file.", http.StatusInternalServerError)
		return
	}

	n2, err := io.Copy(destFile, tempFile)
	if err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to save file %s (copied %d/%d bytes): %v", newFilename, n2, n, err)
		destFile.Close()
		os.Remove(absTargetPath)
		os.Remove(tempFile.Name())
		http.Error(w, "Failed to save uploaded file.", http.StatusInternalServerError)
		return
	}

	destFile.Close()
	os.Remove(tempFile.Name())

	extLower := strings.ToLower(ext)
	var mediaType domain.MediaType = domain.MediaTypePhoto
	switch extLower {
	case ".mp4", ".mov", ".avi":
		mediaType = domain.MediaTypeVideo
	}

	meta := &domain.Media{
		ID:         uuid.New(),
		UserID:     userID,
		Filename:   fileName,
		MediaType:  mediaType,
		Path:       relPathFromRoot,
		SizeBytes:  n,
		Hash:       hash,
		CapturedAt: time.Now(),
	}

	if err := h.mediaRepo.Create(dbCtx, meta); err != nil {
		log.Printf("[ERROR] UploadHandlerSingle: Failed to insert media %s (hash=%s): %v", newFilename, hash[:8]+"...", err)
		http.Error(w, "Failed to save media record.", http.StatusInternalServerError)
		return
	}

	uploadedMedia := map[string]interface{}{
		"id":          meta.ID.String(),
		"filename":    fileName,
		"mediaType":   string(meta.MediaType),
		"path":        relPathFromRoot,
		"size":        n,
		"captured_at": time.Now().Format(time.RFC3339),
	}

	response := map[string]interface{}{
		"uploaded":           []interface{}{uploadedMedia},
		"skipped_duplicates": []interface{}{},
		"errors":             []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("[INFO] UploadHandlerSingle: Successfully uploaded '%s' (ID=%s)", fileName, meta.ID)
}

// HandleChunkUpload handles chunk uploads for resumable file transfers.
func (h *MediaUploadHandlerSingle) HandleChunkUpload(w http.ResponseWriter, r *http.Request, sessionManager *UploadSessionManager) {
	ctx := r.Context()
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := r.ParseMultipartForm(64 << 20) // 64MB memory limit for chunk uploads
	if err != nil {
		log.Printf("[ERROR] UploadHandlerChunk: Failed to parse form: %v", err)
		http.Error(w, "Invalid form data.", http.StatusBadRequest)
		return
	}

	chunks := r.MultipartForm.File["chunk"]
	if len(chunks) == 0 {
		http.Error(w, "No chunk uploaded. Ensure you use -F \"chunk=@...\"", http.StatusBadRequest)
		return
	}

	header := chunks[0]
	fileName := r.FormValue("fileName")
	uploadIDStr := r.FormValue("uploadId")
	chunkIndexStr := r.FormValue("chunkIndex")
	totalChunksStr := r.FormValue("totalChunks")

	if uploadIDStr == "" || chunkIndexStr == "" {
		http.Error(w, "Missing required fields: uploadId and chunkIndex", http.StatusBadRequest)
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		log.Printf("[ERROR] UploadHandlerChunk: Invalid chunkIndex '%s': %v", chunkIndexStr, err)
		http.Error(w, "Invalid chunk index.", http.StatusBadRequest)
		return
	}

	totalChunks := 1 // Default to single-chunk if not specified
	if totalChunksStr != "" {
		fmt.Sscanf(totalChunksStr, "%d", &totalChunks)
	}

	session := sessionManager.GetSession(uploadIDStr)
	if session == nil {
		log.Printf("[WARN] UploadHandlerChunk: Session %s not found for user %s. Creating new one.", uploadIDStr, userID.String())
		fileSize := header.Size
		fmt.Sscanf(r.FormValue("fileSize"), "%d", &fileSize)
		session = sessionManager.CreateSession(userID, fileName, fileSize, totalChunks)
	}

	if session.UserID != userID {
		log.Printf("[WARN] UploadHandlerChunk: Session %s belongs to different user. Creating new one.", uploadIDStr)
		fileSize := header.Size
		fmt.Sscanf(r.FormValue("fileSize"), "%d", &fileSize)
		session = sessionManager.CreateSession(userID, fileName, fileSize, totalChunks)
	}

	log.Printf("[DEBUG] UploadHandlerChunk: Processing chunk %d/%d for session %s (file='%s', size=%d)", chunkIndex+1, totalChunks, uploadIDStr, header.Filename, header.Size)

	chunkFile, err := header.Open()
	if err != nil {
		log.Printf("[ERROR] UploadHandlerChunk: Failed to open chunk file: %v", err)
		http.Error(w, "Failed to read uploaded chunk.", http.StatusBadRequest)
		return
	}

	tempPath := filepath.Join(sessionManager.GetUploadTempDir(), uploadIDStr+fmt.Sprintf("_%d.tmp", chunkIndex))

	chunkData, err := io.ReadAll(chunkFile)
	chunkFile.Close()

	if err != nil {
		log.Printf("[ERROR] UploadHandlerChunk: Failed to read chunk %d: %v", chunkIndex, err)
		http.Error(w, "Failed to process uploaded chunk.", http.StatusInternalServerError)
		return
	}

	err = os.WriteFile(tempPath, chunkData, 0644)
	if err != nil {
		log.Printf("[ERROR] UploadHandlerChunk: Failed to write chunk %d to temp file: %v", chunkIndex, err)
		http.Error(w, "Failed to save uploaded chunk.", http.StatusInternalServerError)
		return
	}

	sessionManager.AddChunk(uploadIDStr, chunkIndex)
	// Persist the updated session to disk for recovery after server restart
	sessionManager.saveSessions()

	bytesReceived := int64(len(chunkData))

	isComplete := sessionManager.IsComplete(uploadIDStr)

	response := map[string]interface{}{
		"success":        true,
		"uploadId":       uploadIDStr,
		"chunkIndex":     chunkIndex + 1, // 1-indexed for the client
		"totalChunks":    totalChunks,
		"bytesReceived":  bytesReceived,
		"isComplete":     isComplete,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("[INFO] UploadHandlerChunk: Chunk %d/%d uploaded for session %s (totalChunks=%d)", chunkIndex+1, totalChunks, uploadIDStr, totalChunks)
}

// HandleStatus returns the status of a resumable upload session.
func (h *MediaUploadHandlerSingle) HandleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	uploadID := r.URL.Query().Get("uploadId")
	if uploadID == "" {
		http.Error(w, "Missing required query parameter: uploadId", http.StatusBadRequest)
		return
	}

	session := h.sessionManager.GetSession(uploadID)
	if session == nil {
		log.Printf("[WARN] UploadHandlerStatus: Session %s not found for user %s.", uploadID, userID.String())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uploadId":       uploadID,
			"fileName":       "",
			"fileSize":       0,
			"uploadedChunks": []int{},
			"totalChunks":    0,
			"isComplete":     false,
			"createdAt":      nil,
		})
		return
	}

	if session.UserID != userID {
		log.Printf("[WARN] UploadHandlerStatus: Session %s belongs to different user.", uploadID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uploadId":       uploadID,
			"fileName":       "",
			"fileSize":       0,
			"uploadedChunks": []int{},
			"totalChunks":    0,
			"isComplete":     false,
			"createdAt":      nil,
		})
		return
	}

	response := map[string]interface{}{
		"uploadId":       session.ID,
		"fileName":       session.Filename,
		"fileSize":       session.FileSize,
		"uploadedChunks": session.UploadedChunks,
		"totalChunks":    session.TotalChunks,
		"isComplete":     h.sessionManager.IsComplete(session.ID),
		"createdAt":      session.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("[INFO] UploadHandlerStatus: Status returned for session %s (uploadedChunks=%d/%d)", uploadID, len(session.UploadedChunks), session.TotalChunks)
}

// HandleComplete assembles all uploaded chunks into the final file and creates the media record.
func (h *MediaUploadHandlerSingle) HandleComplete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Add a database context timeout to prevent DB queries from hanging indefinitely.
	dbCtx, dbCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dbCancel()

	userID, ok := GetUserIDFromContext(dbCtx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := r.ParseMultipartForm(1 << 20) // Small limit for complete request
	if err != nil {
		log.Printf("[ERROR] UploadHandlerComplete: Failed to parse form: %v", err)
		http.Error(w, "Invalid form data.", http.StatusBadRequest)
		return
	}

	uploadID := r.FormValue("uploadId")
	if uploadID == "" {
		http.Error(w, "Missing required field: uploadId", http.StatusBadRequest)
		return
	}

	session := h.sessionManager.GetSession(uploadID)
	if session == nil || !h.sessionManager.IsComplete(session.ID) {
		log.Printf("[WARN] UploadHandlerComplete: Session %s not complete or not found.", uploadID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Upload session is incomplete or does not exist. Upload all chunks first.",
		})
		return
	}

	if session.UserID != userID {
		log.Printf("[WARN] UploadHandlerComplete: Session %s belongs to different user.", uploadID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session does not belong to you.",
		})
		return
	}

	log.Printf("[INFO] UploadHandlerComplete: Assembling chunks for session %s (totalChunks=%d)", uploadID, session.TotalChunks)

	// Read and combine all chunk temp files in order
	tempDir := h.sessionManager.GetUploadTempDir()
	var combinedWriter io.Writer
	hasher := sha256.New()

	// Create a temporary file to hold the assembled content
	assembledFile, err := os.CreateTemp("", "assembled-*.tmp")
	if err != nil {
		log.Printf("[ERROR] UploadHandlerComplete: Failed to create temp file for assembly: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to assemble chunks.",
		})
		return
	}

	combinedWriter = io.MultiWriter(assembledFile, hasher)

	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := filepath.Join(tempDir, uploadID+fmt.Sprintf("_%d.tmp", i))

		log.Printf("[DEBUG] UploadHandlerComplete: Reading chunk %d from %s", i, chunkPath)

		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			log.Printf("[ERROR] UploadHandlerComplete: Failed to read chunk %d for session %s: %v", i, uploadID, err)
			os.Remove(assembledFile.Name()) // Clean up temp file on error
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Failed to read chunk %d: %v", i+1, err),
			})
			return
		}

		n, err := combinedWriter.Write(chunkData)
		if err != nil {
			log.Printf("[ERROR] UploadHandlerComplete: Failed to write chunk %d to assembled file: %v", i, err)
			os.Remove(assembledFile.Name()) // Clean up temp file on error
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Failed to assemble chunk %d: %v", i+1, err),
			})
			return
		}

		log.Printf("[DEBUG] UploadHandlerComplete: Wrote %d bytes for chunk %d (total assembled so far)", n, i)
	}

	assembledFile.Close() // Flush all data before reading back

	if _, err := assembledFile.Seek(0, io.SeekStart); err != nil {
		log.Printf("[ERROR] UploadHandlerComplete: Failed to seek in assembled file for %s: %v", uploadID, err)
		os.Remove(assembledFile.Name())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to assemble chunks.",
		})
		return
	}

	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	assembledFileSize, _ := assembledFile.Stat()

	log.Printf("[DEBUG] UploadHandlerComplete: Hash calculated for '%s': %x...", session.Filename[:min(10, len(session.Filename))], hasher.Sum(nil)[:8])

	// Check for duplicate by hash
	existingMedia, dbErr := h.mediaRepo.GetByHash(dbCtx, hash)
	if dbErr != nil && !errors.Is(dbErr, sql.ErrNoRows) && !strings.Contains(dbErr.Error(), "no rows") {
		log.Printf("[ERROR] UploadHandlerComplete: DB error checking duplicate for %s (hash=%s): %v", session.Filename, hash[:8]+"...", dbErr)
		os.Remove(assembledFile.Name()) // Clean up temp file on error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Database error.",
		})
		return
	}

	if existingMedia != nil && existingMedia.ID != uuid.Nil {
		log.Printf("[INFO] UploadHandlerComplete: Duplicate detected for '%s' (Hash matches ID=%s)", session.Filename, existingMedia.ID.String())
		response := map[string]interface{}{
			"uploaded":           []interface{}{},
			"skipped_duplicates": []map[string]string{{
				"filename": session.Filename,
				"id":       existingMedia.ID.String(),
			}},
			"errors":             []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		os.Remove(assembledFile.Name()) // Clean up temp file on error
		return
	}

	ext := filepath.Ext(session.Filename)
	newFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	dateDir := time.Now().Format("2006/01/02")
	relTimePath := filepath.Join(dateDir, newFilename)
	relPathFromRoot := filepath.Join(userID.String(), relTimePath)

	if err := h.storageService.EnsureDir(userID, relTimePath); err != nil {
		log.Printf("[ERROR] UploadHandlerComplete: Failed to ensure directory %s: %v", dateDir, err)
		os.Remove(assembledFile.Name()) // Clean up temp file on error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to prepare storage.",
		})
		return
	}

	absTargetPath := h.storageService.GetAbsolutePath(relPathFromRoot)

	destFile, err := os.Create(absTargetPath)
	if err != nil {
		log.Printf("[ERROR] UploadHandlerComplete: Failed to create dest file %s: %v", absTargetPath, err)
		os.Remove(assembledFile.Name()) // Clean up temp file on error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to save uploaded file.",
		})
		return
	}

	n2, err := io.Copy(destFile, assembledFile)
	if err != nil {
		log.Printf("[ERROR] UploadHandlerComplete: Failed to save file %s (copied %d/%d bytes): %v", newFilename, n2, assembledFileSize.Size(), err)
		destFile.Close()
		os.Remove(absTargetPath) // Clean up dest file on error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to save uploaded file.",
		})
		return
	}

	destFile.Close()
	os.Remove(assembledFile.Name()) // Clean up temp file on success

	extLower := strings.ToLower(ext)
	var mediaType domain.MediaType = domain.MediaTypePhoto
	switch extLower {
	case ".mp4", ".mov", ".avi":
		mediaType = domain.MediaTypeVideo
	}

	meta := &domain.Media{
		ID:         uuid.New(),
		UserID:     userID,
		Filename:   session.Filename,
		MediaType:  mediaType,
		Path:       relPathFromRoot,
		SizeBytes:  assembledFileSize.Size(),
		Hash:       hash,
		CapturedAt: time.Now(),
	}

	if err := h.mediaRepo.Create(dbCtx, meta); err != nil {
		log.Printf("[ERROR] UploadHandlerComplete: Failed to insert media %s (hash=%s): %v", newFilename, hash[:8]+"...", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to save media record.",
		})
		return
	}

	uploadedMedia := map[string]interface{}{
		"id":          meta.ID.String(),
		"filename":    session.Filename,
		"mediaType":   string(meta.MediaType),
		"path":        relPathFromRoot,
		"size":        assembledFileSize.Size(),
		"captured_at": time.Now().Format(time.RFC3339),
	}

	response := map[string]interface{}{
		"uploaded":           []interface{}{uploadedMedia},
		"skipped_duplicates": []interface{}{},
		"errors":             []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Clean up all chunk temp files and session from memory
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := filepath.Join(tempDir, uploadID+fmt.Sprintf("_%d.tmp", i))
		os.Remove(chunkPath) // Remove each chunk file after successful assembly
	}
	h.sessionManager.DeleteSession(uploadID)

	log.Printf("[INFO] UploadHandlerComplete: Successfully assembled '%s' (ID=%s)", session.Filename, meta.ID)
}

// HandleAbort aborts an in-progress resumable upload session and cleans up temp files.
func (h *MediaUploadHandlerSingle) HandleAbort(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := r.ParseMultipartForm(1 << 20) // Small limit for abort request
	if err != nil {
		log.Printf("[ERROR] UploadHandlerAbort: Failed to parse form: %v", err)
		http.Error(w, "Invalid form data.", http.StatusBadRequest)
		return
	}

	uploadID := r.FormValue("uploadId")
	if uploadID == "" {
		http.Error(w, "Missing required field: uploadId", http.StatusBadRequest)
		return
	}

	session := h.sessionManager.GetSession(uploadID)
	if session == nil {
		log.Printf("[WARN] UploadHandlerAbort: Session %s not found.", uploadID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Session not found or already completed.",
		})
		return
	}

	if session.UserID != userID {
		log.Printf("[WARN] UploadHandlerAbort: Session %s belongs to different user.", uploadID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Session does not belong to you.",
		})
		return
	}

	h.sessionManager.DeleteSession(uploadID)
	// Persist session deletion to disk for recovery after server restart
	h.sessionManager.saveSessions()

	log.Printf("[INFO] UploadHandlerAbort: Aborted upload session %s for user %s", uploadID, userID.String())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Upload session aborted and cleaned up.",
	})
}

// HandleDelete deletes a media item from the server.
func (h *MediaUploadHandlerSingle) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Add a database context timeout to prevent DB queries from hanging indefinitely.
	dbCtx, dbCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dbCancel()

	userID, ok := GetUserIDFromContext(dbCtx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := r.ParseMultipartForm(1 << 20) // Small limit for delete request
	if err != nil {
		log.Printf("[ERROR] HandleDelete: Failed to parse form: %v", err)
		http.Error(w, "Invalid form data.", http.StatusBadRequest)
		return
	}

	mediaIDStr := r.FormValue("mediaId")
	if mediaIDStr == "" {
		http.Error(w, "Missing required field: mediaId", http.StatusBadRequest)
		return
	}

	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		log.Printf("[ERROR] HandleDelete: Invalid UUID format for mediaId '%s': %v", mediaIDStr, err)
		http.Error(w, "Invalid media ID.", http.StatusBadRequest)
		return
	}

	// Get the media item to find its path before deleting from DB
	media, err := h.mediaRepo.GetByID(dbCtx, mediaID, &userID)
	if err != nil {
		log.Printf("[ERROR] HandleDelete: Failed to get media %s for user %s: %v", mediaIDStr, userID.String(), err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to get media: %v", err),
		})
		return
	}

	// Delete the file from storage
	if err := h.storageService.DeleteFile(media.Path); err != nil {
		log.Printf("[ERROR] HandleDelete: Failed to delete file from storage for media ID %s: %v", mediaIDStr, err)
		// Continue with DB deletion even if file delete fails
	}

	// Delete thumbnail separately (it's optional and may not exist)
	// Strip user ID from path since thumbnails are stored without it
	cleanPath := strings.TrimPrefix(media.Path, "storage/")
	parts := strings.SplitN(cleanPath, string(filepath.Separator), 2)
	if len(parts) >= 2 {
		cleanPath = parts[1]
	}
	ext := filepath.Ext(cleanPath)
	if thumbRelPath := h.storageService.GetThumbnailRelativePath(string(media.MediaType), cleanPath, ext); thumbRelPath != "" {
		h.storageService.DeleteFileSilently(thumbRelPath) // Ignore error for thumbnails
	}

	// Permanently delete from database
	err = h.mediaRepo.PermanentlyDeleteMedia(dbCtx, mediaID, userID)
	if err != nil {
		log.Printf("[ERROR] HandleDelete: Failed to permanently delete media %s from DB: %v", mediaIDStr, err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to delete media record: %v", err),
		})
		return
	}

	log.Printf("[INFO] HandleDelete: Successfully deleted media ID %s for user %s", mediaIDStr, userID.String())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Media deleted successfully.",
	})
}
