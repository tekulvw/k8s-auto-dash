package assets

import (
	"io/fs"
	"net/http"
	"strings"
)

// UIHandler serves the SvelteKit SPA. Any request whose path is not a
// real file falls back to index.html so client-side routing works.
func UIHandler() http.Handler {
	root, err := fs.Sub(ui, "ui")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cheap SPA fallback: if the file doesn't exist, serve index.html.
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(root, clean); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// IconsHandler serves /icons/<slug>.png from the embedded icon set.
func IconsHandler() http.Handler {
	root, err := fs.Sub(icons, "icons")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/icons/", http.FileServer(http.FS(root)))
}
