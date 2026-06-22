package api

import (
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

func TestHandler_SearchMedia(t *testing.T) {
	mediaRepo := new(MockMediaRepository)
	faceRepo := new(MockFaceRepository)
	handler := &Handler{
		mediaRepo:   mediaRepo,
		faceRepo:    faceRepo,
		storageRoot: "/tmp",
		thumbRoot:   "/tmp/thumbs",
	}

	r := chi.NewRouter()
	r.Get("/media/search", handler.SearchMedia)

	t.Run("success", func(t *testing.T) {
		tag := "vacation"
		mediaList := []*domain.Media{
			{ID: uuid.New(), Tags: "nature:vacation:summer"},
		}
		mediaRepo.On("SearchByTags", mock.Anything, tag, mock.Anything).Return(mediaList, nil)

		req := httptest.NewRequest("GET", "/media/search?tags="+tag, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp SearchMediaResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(resp.Photos))
		assert.Equal(t, "nature:vacation:summer", resp.Photos[0].Tags)
	})

	t.Run("missing_tags", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/media/search", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
