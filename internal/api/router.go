package api

import (
	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(photoRepo domain.PhotoRepository, faceRepo domain.FaceRepository, storageRoot string, thumbRoot string) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := &Handler{
		photoRepo:   photoRepo,
		faceRepo:    faceRepo,
		storageRoot: storageRoot,
		thumbRoot:   thumbRoot,
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/photos", h.ListPhotos)
		r.Get("/photos/{id}", h.GetPhoto)
		r.Get("/photos/{id}/file", h.ServePhotoFile)
		r.Get("/photos/{id}/thumb", h.ServeThumbnailFile)
	})

	return r
}
