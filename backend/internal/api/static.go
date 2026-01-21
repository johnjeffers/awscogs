package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var frontendFS embed.FS

// SPAHandler serves the embedded frontend files with SPA support.
// For paths that don't match a static file, it serves index.html
// to allow client-side routing to handle the request.
type SPAHandler struct {
	fs http.Handler
}

// NewSPAHandler creates a handler for serving the embedded frontend.
func NewSPAHandler() *SPAHandler {
	dist, _ := fs.Sub(frontendFS, "dist")
	return &SPAHandler{
		fs: http.FileServer(http.FS(dist)),
	}
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Try to serve the file directly
	dist, _ := fs.Sub(frontendFS, "dist")

	// Check if the path exists as a file
	if path != "/" {
		cleanPath := strings.TrimPrefix(path, "/")
		if _, err := fs.Stat(dist, cleanPath); err == nil {
			h.fs.ServeHTTP(w, r)
			return
		}
	}

	// For paths that don't exist as files, serve index.html (SPA routing)
	if path == "/" || !strings.Contains(path, ".") {
		indexContent, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "Frontend not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexContent)
		return
	}

	// For other paths (like missing assets), let the file server handle it
	h.fs.ServeHTTP(w, r)
}
