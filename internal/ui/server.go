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
}
