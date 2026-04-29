package ui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/ahoylog/kvik-tasks/internal/core"
	"github.com/go-chi/chi/v5"
)

//go:embed static
var staticFS embed.FS

// MountRoutes adds UI routes to the given chi router.
func MountRoutes(r chi.Router, projectService *core.ProjectService) {
	h := &UIHandler{projectService: projectService}

	// Static assets
	staticSub, _ := fs.Sub(staticFS, "static")
	fileServer := http.FileServer(http.FS(staticSub))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Pages
	r.Get("/", h.Dashboard)
	r.Get("/p/{slug}", h.Project)
	r.Get("/p/{slug}/board", h.Board)
	r.Get("/p/{slug}/settings", h.Settings)
	r.Get("/p/{slug}/archive", h.ArchivePage)
	r.Get("/p/{slug}/t/new", h.TaskNewPage)
	r.Post("/p/{slug}/tasks/new", h.CreateDetailedTask)
	r.Get("/p/{slug}/t/{id}", h.TaskDetail)

	// HTMX fragment endpoints (accept JSON, return HTML)
	r.Get("/p/{slug}/search", h.SearchTasks)
	r.Post("/p/{slug}/tasks", h.CreateTask)
	r.Post("/p/{slug}/tasks/{id}/complete", h.CompleteTask)
	r.Post("/p/{slug}/tasks/{id}/move", h.MoveTask)
	r.Post("/p/{slug}/tasks/bulk", h.BulkAction)
	r.Delete("/p/{slug}/tasks/{id}", h.DeleteTask)
	r.Patch("/p/{slug}/tasks/{id}/field", h.UpdateTaskField)
	r.Get("/p/{slug}/tasks/{id}/delete", h.DeleteTaskRedirect)
	r.Get("/p/{slug}/tasks/{id}/archive", h.ArchiveTaskRedirect)
	r.Get("/p/{slug}/tasks/{id}/unarchive", h.UnarchiveTaskRedirect)

	// Project configuration
	r.Post("/p/{slug}/settings/config", h.SaveConfig)

	// Template management
	r.Get("/p/{slug}/templates/{tplID}", h.TemplateDetail)
	r.Post("/p/{slug}/templates", h.CreateTemplate)
	r.Patch("/p/{slug}/templates/{tplID}", h.UpdateTemplate)
	r.Delete("/p/{slug}/templates/{tplID}", h.DeleteTemplate)
	r.Get("/p/{slug}/templates/{tplID}/delete", h.DeleteTemplateRedirect)

	// Category management
	r.Get("/p/{slug}/categories", h.ListCategories)
	r.Post("/p/{slug}/categories", h.CreateCategory)
	r.Patch("/p/{slug}/categories/{catID}", h.UpdateCategory)
	r.Delete("/p/{slug}/categories/{catID}", h.DeleteCategory)
}
