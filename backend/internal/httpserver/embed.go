package httpserver

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

//go:embed all:frontend_dist
var frontendFS embed.FS

func FrontendHandler() http.Handler {
	sub, err := fs.Sub(frontendFS, "frontend_dist")
	if err != nil {
		log.Printf("frontend assets not available: %v", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"frontend not built"}`, http.StatusNotFound)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to serve static file
		if _, err := fs.ReadFile(sub, path); err == nil {
			http.FileServer(http.FS(sub)).ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html
		w.Header().Set("Content-Type", "text/html")
		data, _ := fs.ReadFile(sub, "index.html")
		w.Write(data)
	})
}
