package api

import (
	"net/http"
	"time"

	"github.com/ahoylog/kvik-tasks/internal/api/handlers"
	"github.com/ahoylog/kvik-tasks/internal/api/middleware"
	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/ahoylog/kvik-tasks/internal/ui"
	"github.com/go-chi/chi/v5"
)

// NewRouter creates a chi router with all REST API routes.
func NewRouter(projectService *core.ProjectService) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.CORS)
	r.Use(middleware.RequestID)
	r.Use(middleware.BasicAuth(projectService.UserService()))

	taskHandler := handlers.NewTaskHandler(projectService)
	projectHandler := handlers.NewProjectHandler(projectService)
	systemHandler := handlers.NewSystemHandler()

	// UI routes (dashboard, project pages, static assets)
	ui.MountRoutes(r, projectService)

	r.Route("/api/v1", func(r chi.Router) {
		// System
		r.Get("/health", systemHandler.Health)
		r.Get("/version", systemHandler.Version)

		// Projects
		r.Get("/projects", projectHandler.List)
		r.Get("/projects/{slug}", projectHandler.Get)
		r.Get("/projects/{slug}/tasks", taskHandler.ListByProject)
		r.Get("/projects/{slug}/stats", projectHandler.Stats)

		// Tasks
		r.Post("/projects/{slug}/tasks", taskHandler.Create)
		r.Get("/projects/{slug}/tasks/{id}", taskHandler.Get)
		r.Patch("/projects/{slug}/tasks/{id}", taskHandler.Update)
		r.Delete("/projects/{slug}/tasks/{id}", taskHandler.Delete)
		r.Post("/projects/{slug}/tasks/{id}/complete", taskHandler.Complete)
		r.Post("/projects/{slug}/tasks/{id}/archive", taskHandler.Archive)
		r.Post("/projects/{slug}/tasks/{id}/unarchive", taskHandler.Unarchive)
	})

	return r
}

// NewServer creates an http.Server with the given handler and address.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
