package api

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"steadyphoto/internal/domain"
	"steadyphoto/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockJobRepository implements domain.JobRepository for testing.
type MockJobRepository struct {
	jobs []*domain.Job
}

func (m *MockJobRepository) Create(ctx context.Context, job *domain.Job) error {
	m.jobs = append(m.jobs, job)
	return nil
}

func (m *MockJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	for _, job := range m.jobs {
		if job.ID == id {
			return job, nil
		}
	}
	return nil, nil
}

func (m *MockJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errStr string) error {
	for _, job := range m.jobs {
		if job.ID == id {
			job.Status = status
			return nil
		}
	}
	return nil
}

func (m *MockJobRepository) GetPending(ctx context.Context, limit int) ([]*domain.Job, error) {
	var pending []*domain.Job
	for _, job := range m.jobs {
		if job.Status == domain.JobStatusPending {
			pending = append(pending, job)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

// --- Tests ---

func TestDeleteSession_CleansUpChunkFiles(t *testing.T) {
	dir := t.TempDir()
	storageSvc := storage.NewStorageService(dir, filepath.Join(dir, "thumbs"))
	manager := NewUploadSessionManager(storageSvc)

	userID := uuid.New()
	session := manager.CreateSession("test-upload-id", userID, "video.mp4", 1024*1024, 3)

	// Create chunk files on disk (simulating uploaded chunks)
	chunkPaths := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		err := os.WriteFile(chunkPath, []byte(fmt.Sprintf("chunk-%d", i)), 0644)
		assert.NoError(t, err)
		chunkPaths = append(chunkPaths, chunkPath)
	}

	// Verify chunk files exist before deletion
	for _, p := range chunkPaths {
		_, err := os.Stat(p)
		assert.NoError(t, err, "chunk file should exist before DeleteSession: %s", p)
	}

	// Delete the session
	manager.DeleteSession(session.ID)

	// Verify all chunk files are removed
	for _, p := range chunkPaths {
		_, err := os.Stat(p)
		assert.True(t, os.IsNotExist(err), "chunk file should be removed after DeleteSession: %s", p)
	}

	// Verify the assembled temp file is also removed
	_, err := os.Stat(session.tempPath)
	assert.True(t, os.IsNotExist(err), "assembled temp file should be removed")

	// Verify session is gone from memory
	assert.Nil(t, manager.GetSession(session.ID))
}

func TestDeleteSession_CleansUpChunkFiles_WhenOnlyPartialChunksExist(t *testing.T) {
	dir := t.TempDir()
	storageSvc := storage.NewStorageService(dir, filepath.Join(dir, "thumbs"))
	manager := NewUploadSessionManager(storageSvc)

	userID := uuid.New()
	session := manager.CreateSession("test-upload-id", userID, "video.mp4", 1024*1024, 5)

	// Only create 2 out of 5 chunk files (partial upload)
	for i := 0; i < 2; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		err := os.WriteFile(chunkPath, []byte(fmt.Sprintf("chunk-%d", i)), 0644)
		assert.NoError(t, err)
	}

	// Delete the session — should not panic even though chunks 2-4 don't exist
	manager.DeleteSession(session.ID)

	// The 2 existing chunks should be removed
	for i := 0; i < 2; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		_, err := os.Stat(chunkPath)
		assert.True(t, os.IsNotExist(err), "chunk %d should be removed", i)
	}
}

func TestDeleteSession_CleansUpChunkFiles_WhenNoChunksExist(t *testing.T) {
	dir := t.TempDir()
	storageSvc := storage.NewStorageService(dir, filepath.Join(dir, "thumbs"))
	manager := NewUploadSessionManager(storageSvc)

	userID := uuid.New()
	session := manager.CreateSession("test-upload-id", userID, "video.mp4", 1024*1024, 3)

	// Don't create any chunk files — session was just created

	// Delete should not panic
	manager.DeleteSession(session.ID)

	// Verify session is gone
	assert.Nil(t, manager.GetSession(session.ID))
}

func TestDeleteSession_SessionNotFound_NoPanic(t *testing.T) {
	dir := t.TempDir()
	storageSvc := storage.NewStorageService(dir, filepath.Join(dir, "thumbs"))
	manager := NewUploadSessionManager(storageSvc)

	// Delete a non-existent session — should not panic
	manager.DeleteSession(uuid.New().String())
}

func TestCleanupExpiredSessions_RemovesOldSessionsAndChunks(t *testing.T) {
	dir := t.TempDir()
	storageSvc := storage.NewStorageService(dir, filepath.Join(dir, "thumbs"))
	manager := NewUploadSessionManager(storageSvc)

	userID := uuid.New()

	// Create an "old" session (manually set CreatedAt to 25 hours ago)
	session := manager.CreateSession("test-upload-id", userID, "old-video.mp4", 1024*1024, 2)

	// Manually backdate the CreatedAt
	manager.mu.Lock()
	session.CreatedAt = time.Now().Add(-25 * time.Hour)
	manager.mu.Unlock()

	// Create chunk files for the old session
	for i := 0; i < 2; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		err := os.WriteFile(chunkPath, []byte(fmt.Sprintf("chunk-%d", i)), 0644)
		assert.NoError(t, err)
	}

	// Create a "new" session that should NOT be cleaned up
	newSession := manager.CreateSession("test-upload-id", userID, "new-video.mp4", 512*1024, 1)
	// Create a chunk for the new session too
	chunkPath := filepath.Join(dir, ".upload-temp", newSession.ID+fmt.Sprintf("_0.tmp"))
	err := os.WriteFile(chunkPath, []byte("chunk-0"), 0644)
	assert.NoError(t, err)

	// Run cleanup
	count := manager.CleanupExpiredSessions()

	// Only the old session should be cleaned up
	assert.Equal(t, 1, count, "should clean up exactly 1 expired session")

	// Old session's chunks should be gone
	for i := 0; i < 2; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		_, err := os.Stat(chunkPath)
		assert.True(t, os.IsNotExist(err), "old chunk %d should be removed", i)
	}

	// Old session should be gone from memory
	assert.Nil(t, manager.GetSession(session.ID))

	// New session should still exist
	assert.NotNil(t, manager.GetSession(newSession.ID))

	// New session's chunk should still exist
	_, err = os.Stat(chunkPath)
	assert.NoError(t, err, "new session chunk should still exist")
}

func TestCleanupExpiredSessions_NewSessions_NotRemoved(t *testing.T) {
	dir := t.TempDir()
	storageSvc := storage.NewStorageService(dir, filepath.Join(dir, "thumbs"))
	manager := NewUploadSessionManager(storageSvc)

	userID := uuid.New()

	// Create a fresh session
	session := manager.CreateSession("test-upload-id", userID, "recent.mp4", 1024*1024, 2)

	// Create chunk files
	for i := 0; i < 2; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		err := os.WriteFile(chunkPath, []byte(fmt.Sprintf("chunk-%d", i)), 0644)
		assert.NoError(t, err)
	}

	// Run cleanup — nothing should be expired
	count := manager.CleanupExpiredSessions()
	assert.Equal(t, 0, count, "no sessions should be expired yet")

	// Session and chunks should still exist
	assert.NotNil(t, manager.GetSession(session.ID))
	for i := 0; i < 2; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		_, err := os.Stat(chunkPath)
		assert.NoError(t, err, "recent chunk %d should still exist", i)
	}
}

func TestHandleAbort_CleansUpChunkFiles(t *testing.T) {
	dir := t.TempDir()
	storageSvc := storage.NewStorageService(dir, filepath.Join(dir, "thumbs"))
	mediaRepo := new(MockMediaRepository)
	jobRepo := new(MockJobRepository)
	sessionManager := NewUploadSessionManager(storageSvc)
	handler := NewMediaUploadHandlerSingle(mediaRepo, jobRepo, storageSvc, sessionManager)

	userID := uuid.New()
	session := sessionManager.CreateSession("test-upload-id", userID, "abort-test.mp4", 1024*1024, 3)

	// Create chunk files
	chunkPaths := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		err := os.WriteFile(chunkPath, []byte(fmt.Sprintf("chunk-%d", i)), 0644)
		assert.NoError(t, err)
		chunkPaths = append(chunkPaths, chunkPath)
	}

	// Verify chunks exist
	for _, p := range chunkPaths {
		_, err := os.Stat(p)
		assert.NoError(t, err)
	}

	// Create a proper multipart/form-data request body
	multipartBody := &bytes.Buffer{}
	writer := multipart.NewWriter(multipartBody)
	writer.WriteField("uploadId", session.ID)
	writer.Close()

	req := httptest.NewRequest("POST", "/abort", multipartBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(withUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	handler.HandleAbort(rr, req)

	assert.Equal(t, 200, rr.Code)

	// Verify all chunk files are removed
	for _, p := range chunkPaths {
		_, err := os.Stat(p)
		assert.True(t, os.IsNotExist(err), "chunk should be removed after abort: %s", p)
	}

	// Verify session is gone
	assert.Nil(t, sessionManager.GetSession(session.ID))
}

func TestHandleComplete_SuccessPath_CleansUpChunkFiles(t *testing.T) {
	dir := t.TempDir()
	storageSvc := storage.NewStorageService(dir, filepath.Join(dir, "thumbs"))
	mediaRepo := new(MockMediaRepository)
	jobRepo := new(MockJobRepository)
	sessionManager := NewUploadSessionManager(storageSvc)
	handler := NewMediaUploadHandlerSingle(mediaRepo, jobRepo, storageSvc, sessionManager)

	userID := uuid.New()
	session := sessionManager.CreateSession("test-upload-id", userID, "complete-test.mp4", 1024*1024, 2)

	// Mark session as complete (all chunks uploaded)
	sessionManager.AddChunk(session.ID, 0)
	sessionManager.AddChunk(session.ID, 1)

	// Create chunk files
	chunkPaths := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		chunkPath := filepath.Join(dir, ".upload-temp", session.ID+fmt.Sprintf("_%d.tmp", i))
		err := os.WriteFile(chunkPath, []byte(fmt.Sprintf("chunk-%d", i)), 0644)
		assert.NoError(t, err)
		chunkPaths = append(chunkPaths, chunkPath)
	}

	// Mock GetByHash to return nil (no duplicate found) so the upload proceeds
	mediaRepo.On("GetByHash", mock.Anything, mock.Anything).Return((*domain.Media)(nil), nil)
	// Mock Create to return nil (no error) so the media is persisted
	mediaRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// Create a proper multipart/form-data request body
	multipartBody := &bytes.Buffer{}
	writer := multipart.NewWriter(multipartBody)
	writer.WriteField("uploadId", session.ID)
	writer.Close()

	req := httptest.NewRequest("POST", "/complete", multipartBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(withUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	handler.HandleComplete(rr, req)

	assert.Equal(t, 200, rr.Code)

	// Verify all chunk files are removed
	for _, p := range chunkPaths {
		_, err := os.Stat(p)
		assert.True(t, os.IsNotExist(err), "chunk should be removed after complete: %s", p)
	}

	// Verify session is gone
	assert.Nil(t, sessionManager.GetSession(session.ID))

	mediaRepo.AssertExpectations(t)
}

// --- Helpers ---

// withUserID adds a userID to the request context for testing.
// This uses the same context key as the production AuthMiddleware.
func withUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDContextKey, userID)
}
