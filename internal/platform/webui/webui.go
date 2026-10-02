// Package webui serves the built single-page application from the API
// binary (ADR-020): same origin as the API, so no CORS and simple cookies.
//
// The Vite build is synced into ./dist by `make web-embed`. In development the
// directory holds only .gitkeep, and a placeholder page explains how to run
// the Vite dev server instead.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// CSP for the SPA. React sets inline styles through the CSSOM (allowed), but
// some UI primitives inject <style> elements, hence 'unsafe-inline' for
// style-src only. Scripts are strictly same-origin. extraConnect/extraImg
// add object-storage origins for direct uploads and previews.
func CSP(extraConnect, extraImg []string) string {
	connect := append([]string{"'self'"}, extraConnect...)
	img := append([]string{"'self'", "data:", "blob:"}, extraImg...)
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src " + strings.Join(img, " "),
		"font-src 'self'",
		"connect-src " + strings.Join(connect, " "),
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}

// Handler serves hashed assets with immutable caching, index.html with
// no-cache, and falls back to index.html for client-side routes. Requests
// under reserved API prefixes are never answered with the SPA.
func Handler(reservedPrefixes ...string) http.Handler {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	index, indexErr := fs.ReadFile(dist, "index.html")
	files := http.FileServerFS(dist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, p := range reservedPrefixes {
			if strings.HasPrefix(r.URL.Path, p) {
				http.NotFound(w, r)
				return
			}
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if indexErr != nil {
			servePlaceholder(w)
			return
		}

		clean := path.Clean("/" + r.URL.Path)
		name := strings.TrimPrefix(clean, "/")
		if name != "" && name != "index.html" {
			if st, err := fs.Stat(dist, name); err == nil && !st.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					// Vite content-hashes everything under assets/.
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "public, max-age=3600")
				}
				files.ServeHTTP(w, r)
				return
			}
			// A missing file with an extension is a real 404, not a route.
			if path.Ext(name) != "" {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}

func servePlaceholder(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>OpsGrid</title>
<p>The web UI is not embedded in this build. In development, open the Vite dev server
(<code>make web-dev</code>, http://localhost:5173). For a production build run <code>make web-embed</code>.</p>`))
}
