package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"steadyphoto/internal/domain"
)

func NewRouter(repo domain.PhotoRepository, storageRoot string) *chi.Mux {
	r := chi.NewRouter()

	// Standard middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	handler := NewPhotoHandler(repo, storageRoot)

	// API Version 1
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/photos", func(r chi.Router) {
			r.Get("/", handler.ListPhotos)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handler.GetPhoto)
				r.Get("/file", handler.ServePhotoFile)
			})
		})
	})

	return r
}
