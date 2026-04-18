package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)    // logs every request
	r.Use(middleware.Recoverer) // catches panics, returns 500

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", h.HealthCheck)

		r.Route("/jobs", func(r chi.Router) {
			r.Post("/", h.CreateJob)
			r.Get("/", h.ListJobs)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetJob)
				r.Delete("/", h.DeleteJob)
				r.Get("/runs", h.ListJobRuns)
			})
		})

	})

	return r
}
