package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/audit"
)

var publisher *audit.AuditMessenger

// MakeHandler registers all available API endpoints.
func MakeHandler(p *audit.AuditMessenger) *chi.Mux {
	publisher = p
	r := chi.NewRouter()
	r.Get("/{link}", handleGetLink)
	r.Get("/ping", handlePing)
	r.Get("/api/user/urls", handleUserListURL)
	r.Post("/", handleCreateLink)
	r.Post("/api/shorten", handleCreateLink)
	r.Post("/api/shorten/batch", handleBatch)
	r.Delete("/api/user/urls", handleDeleteURLs)
	r.Get("/api/internal/stats", handleStats)

	return r
}
