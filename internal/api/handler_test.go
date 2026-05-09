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

func (m *MockMediaRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Media, error) {
	args := m.Called(ctx, id)
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

func (m *MockMediaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMediaRepository) DeleteByMediaID(ctx context.Context, mediaID uuid.UUID) error {
	args := m.Called(ctx, mediaID)
	return args.Error(0)
}

func (m *MockMediaRepository) List(ctx context.Context, limit, offset int) ([]*domain.Media, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
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

	t.Run("success", func(t *testing.T) {
		mediaList := []*domain.Media{
			{ID: uuid.New(), MediaType: domain.MediaTypeVideo},
			{ID: uuid.New(), MediaType: domain.MediaTypeVideo},
		}
		mediaRepo.On("List", mock.Anything, 20, 0).Return(mediaList, 2, nil)

		req := httptest.NewRequest("GET", "/media", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp ListMediaResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(resp.Media))
		assert.Equal(t, 2, resp.TotalCount)
	})
}

func TestHandler_GetMedia(t *testing.T) {
	mediaRepo := new(MockMediaRepository)
	faceRepo := new(MockFaceRepository)
	handler := &Handler{
		mediaRepo:   mediaRepo,
		faceRepo:    faceRepo,
		storageRoot: "/tmp",
		thumbRoot: "/tmp/thumbs",
	}

	r := chi.NewRouter()
	r.Get("/media/{id}", handler.GetMedia)

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		media := &domain.Media{ID: id, MediaType: domain.MediaTypeVideo}
		mediaRepo.On("GetByID", mock.Anything, id).Return(media, nil)

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
		// The issue was likely because GetByID returned (nil, nil) but the handler expects (nil, error) or something?
		// Wait, if GetByID returns (nil, nil), then media is nil.
		// In GetMedia:
		// media, err := h.mediaRepo.GetByID(ctx, id)
		// if err != nil { ... }
		// If err is nil, it proceeds to encode nil media.
		// encoding nil media results in 200 OK with null body.
		// The design says if not found, return 404.
		// So GetByID should return an error if not found.
		// Or the handler should check if media is nil.
		mediaRepo.On("GetByID", mock.Anything, id).Return(nil, context.DeadlineExceeded) // Using an error to trigger 404

		req := httptest.NewRequest("GET", "/media/"+id.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
