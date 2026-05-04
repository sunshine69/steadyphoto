package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// MockPhotoRepository implements domain.PhotoRepository for testing
type MockPhotoRepository struct {
	photos []*domain.Photo
}

func (m *MockPhotoRepository) Create(ctx context.Context, photo *domain.Photo) error {
	m.photos = append(m.photos, photo)
	return nil
}

func (m *MockPhotoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Photo, error) {
	for _, p := range m.photos {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *MockPhotoRepository) GetByHash(ctx context.Context, hash string) (*domain.Photo, error) {
	return nil, nil
}

func (m *MockPhotoRepository) Update(ctx context.Context, photo *domain.Photo) error {
	return nil
}

func (m *MockPhotoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockPhotoRepository) DeleteByPhotoID(ctx context.Context, photoID uuid.UUID) error {
	return nil
}

func (m *MockPhotoRepository) List(ctx context.Context, limit, offset int) ([]*domain.Photo, int, error) {
	return m.photos, len(m.photos), nil
}

// MockFaceRepository implements domain.FaceRepository for testing
type MockFaceRepository struct{}

func (m *MockFaceRepository) Create(ctx context.Context, face *domain.Face) error { return nil }
func (m *MockFaceRepository) GetByPhotoID(ctx context.Context, photoID uuid.UUID) ([]*domain.Face, error) {
	return nil, nil
}
func (m *MockFaceRepository) DeleteByPhotoID(ctx context.Context, photoID uuid.UUID) error { return nil }

func TestHandler_ListPhotos(t *testing.T) {
	mockRepo := &MockPhotoRepository{
		photos: []*domain.Photo{
			{ID: uuid.New(), Hash: "test-hash"},
		},
	}
	h := &Handler{
		photoRepo:   mockRepo,
		faceRepo:    &MockFaceRepository{},
		storageRoot: "/tmp",
	}

	r := chi.NewRouter()
	r.Get("/api/v1/photos", h.ListPhotos)

	req, _ := http.NewRequest("GET", "/api/v1/photos", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp ListPhotosResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Photos) != 1 {
		t.Errorf("expected 1 photo, got %d", len(resp.Photos))
	}
}

func TestHandler_GetPhoto(t *testing.T) {
	id := uuid.New()
	mockRepo := &MockPhotoRepository{
		photos: []*domain.Photo{
			{ID: id, Hash: "test-hash"},
		},
	}
	h := &Handler{
		photoRepo:   mockRepo,
		faceRepo:    &MockFaceRepository{},
		storageRoot: "/tmp",
	}

	r := chi.NewRouter()
	r.Get("/api/v1/photos/{id}", h.GetPhoto)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/photos/%s", id.String()), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var photo domain.Photo
	if err := json.NewDecoder(rr.Body).Decode(&photo); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if photo.ID != id {
		t.Errorf("expected ID %s, got %s", id, photo.ID)
	}
}
