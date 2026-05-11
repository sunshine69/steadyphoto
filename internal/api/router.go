package api

import (
	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(mediaRepo domain.MediaRepository, faceRepo domain.FaceRepository, storageRoot string, thumbRoot string) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := &Handler{
		mediaRepo:   mediaRepo,
		faceRepo:    faceRepo,
		storageRoot: storageRoot,
		thumbRoot:   thumbRoot,
	}

	r.Route("/api/v1", func(r chi.Router) {
		// Photo-specific endpoints (backward compatible)
		r.Get("/photos", h.ListPhotos)
		r.Get("/photos/{id}", h.GetPhoto)
		r.Get("/photos/{id}/file", h.ServePhotoFile)
		r.Get("/photos/{id}/thumb", h.ServeThumbnailFile)
		
		// Unified media endpoints (for video support)
		r.Get("/media", h.ListMedia)
		r.Get("/media/{id}", h.GetMedia)
		r.Get("/media/{id}/file", h.ServeMediaFile)
		r.Get("/media/{id}/thumb", h.ServeThumbnailFile)
		r.Get("/media/search", h.SearchMedia)
	})

	return r
}
