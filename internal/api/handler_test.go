package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMediaRepository struct {
	mock.Mock
}

func (m *MockMediaRepository) Create(ctx context.Context, media *domain.Media) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *MockMediaRepository) GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*domain.Media, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *MockMediaRepository) GetByHash(ctx context.Context, hash string) (*domain.Media, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *MockMediaRepository) Update(ctx context.Context, media *domain.Media) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *MockMediaRepository) Delete(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMediaRepository) DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error {
	args := m.Called(ctx, mediaID)
	return args.Error(0)
}

func (m *MockMediaRepository) List(ctx context.Context, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	args := m.Called(ctx, limit, offset, userID)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
}

func (m *MockMediaRepository) ListByType(ctx context.Context, mediaType domain.MediaType, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	args := m.Called(ctx, mediaType, limit, offset, userID)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
}

func (m *MockMediaRepository) SearchByTags(ctx context.Context, tags string, userID *uuid.UUID) ([]*domain.Media, error) {
	args := m.Called(ctx, tags, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Media), args.Error(1)
}

func (m *MockMediaRepository) ListTrashed(ctx context.Context, limit, offset int, userID uuid.UUID) ([]*domain.Media, int, error) {
	args := m.Called(ctx, limit, offset, userID)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
}

func (m *MockMediaRepository) GetTrashedMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Media, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *MockMediaRepository) Search(ctx context.Context, query string, scope string, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	args := m.Called(ctx, query, scope, limit, offset, userID)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
}

func (m *MockMediaRepository) RestoreMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMediaRepository) PermanentlyDeleteMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

type MockFaceRepository struct {
	mock.Mock
}

func (m *MockFaceRepository) Create(ctx context.Context, face *domain.Face) error {
	args := m.Called(ctx, face)
	return args.Error(0)
}

func (m *MockFaceRepository) GetByMediaID(ctx context.Context, mediaID uuid.UUID) ([]*domain.Face, error) {
	args := m.Called(ctx, mediaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Face), args.Error(1)
}

func (m *MockFaceRepository) DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error {
	args := m.Called(ctx, mediaID)
	return args.Error(0)
}

func TestHandler_ListMedia(t *testing.T) {
	mediaRepo := new(MockMediaRepository)
	faceRepo := new(MockFaceRepository)
	handler := &Handler{
		mediaRepo:   mediaRepo,
		faceRepo:    faceRepo,
		storageRoot: "/tmp",
		thumbRoot:   "/tmp/thumbs",
	}

	r := chi.NewRouter()
	r.Get("/media", handler.ListMedia)

	t.Run("success_with_pagination", func(t *testing.T) {
		mediaList := []*domain.Media{
			{ID: uuid.New(), MediaType: domain.MediaTypeVideo},
			{ID: uuid.New(), MediaType: domain.MediaTypeVideo},
		}
		// Testing with limit=10 and offset=0
		mediaRepo.On("List", mock.Anything, 10, 0, mock.Anything).Return(mediaList, 2, nil)

		req := httptest.NewRequest("GET", "/media?limit=10&offset=0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp ListMediaResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(resp.Photos))
		assert.Equal(t, 2, resp.Total)
	})
}

func TestHandler_ListPhotos(t *testing.T) {
	mediaRepo := new(MockMediaRepository)
	faceRepo := new(MockFaceRepository)
	handler := &Handler{
		mediaRepo:   mediaRepo,
		faceRepo:    faceRepo,
		storageRoot: "/tmp",
		thumbRoot:   "/tmp/thumbs",
	}

	r := chi.NewRouter()
	r.Get("/photos", handler.ListPhotos)

	t.Run("success_filtering_photos", func(t *testing.T) {
		id2 := uuid.New()
		id3 := uuid.New()
		
		// We expect ListByType to be called for photos
		// Note: The total count returned by ListByType should be the count of PHOTOS
		mediaRepo.On("ListByType", mock.Anything, domain.MediaTypePhoto, 20, 0, mock.Anything).Return([]*domain.Media{
			{ID: id2, MediaType: domain.MediaTypePhoto},
			{ID: id3, MediaType: domain.MediaTypePhoto},
		}, 2, nil)

		req := httptest.NewRequest("GET", "/photos?limit=20&offset=0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp ListMediaResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(resp.Photos))
		assert.Equal(t, 2, resp.Total)
		assert.Equal(t, id2, resp.Photos[0].ID)
		assert.Equal(t, id3, resp.Photos[1].ID)
	})
}

func TestHandler_GetMedia(t *testing.T) {
	mediaRepo := new(MockMediaRepository)
	faceRepo := new(MockFaceRepository)
	handler := &Handler{
		mediaRepo:   mediaRepo,
		faceRepo:    faceRepo,
		storageRoot: "/tmp",
		thumbRoot:   "/tmp/thumbs",
	}

	r := chi.NewRouter()
	r.Get("/media/{id}", handler.GetMedia)

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		media := &domain.Media{ID: id, MediaType: domain.MediaTypeVideo}
		mediaRepo.On("GetByID", mock.Anything, id, mock.Anything).Return(media, nil)

		req := httptest.NewRequest("GET", "/media/"+id.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		
		var resp domain.Media
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, id, resp.ID)
	})

	t.Run("not_found", func(t *testing.T) {
		id := uuid.New()
		mediaRepo.On("GetByID", mock.Anything, id, mock.Anything).Return(nil, context.DeadlineExceeded)

		req := httptest.NewRequest("GET", "/media/"+id.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
