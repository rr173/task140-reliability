package httpapi

import (
	"io"
	"net/http"
	"strings"
)

// registerFrontend serves the embedded static frontend. "/" returns
// index.html; other top-level files (app.js, style.css) are served at their
// names. Unknown paths fall back to index.html for client-side routing.
func registerFrontend(mux *http.ServeMux, webFS http.FileSystem) {
	fileServer := http.FileServer(webFS)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		apiPrefixes := []string{
			"/components", "/analyses", "/reports", "/healthz", "/version",
		}
		for _, p := range apiPrefixes {
			if strings.HasPrefix(r.URL.Path, p) {
				notFound(w)
				return
			}
		}
		if r.URL.Path == "/" {
			f, err := webFS.Open("index.html")
			if err != nil {
				writeError(w, http.StatusInternalServerError, "frontend missing")
				return
			}
			defer f.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, f)
			return
		}
		r2 := new(http.Request)
		*r2 = *r
		r2.URL.Path = strings.TrimPrefix(r.URL.Path, "/")
		fileServer.ServeHTTP(w, r2)
	})
}
