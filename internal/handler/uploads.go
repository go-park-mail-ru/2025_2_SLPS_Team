package handler

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func UploadsHandler(staticDir string, prefix string) http.Handler {

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	absStaticDir, err := filepath.Abs(staticDir)
	if err != nil {
		panic("invalid staticDir: " + err.Error())
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, prefix) {
			sendJSONError(w, "Not found", http.StatusNotFound)
			return
		}

		relPath, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, prefix))
		if err != nil {
			sendJSONError(w, "Bad request", http.StatusBadRequest)
			return
		}

		cleanPath := path.Clean("/" + relPath)
		cleanPath = strings.TrimPrefix(cleanPath, "/")

		if strings.Contains(cleanPath, "..") {
			sendJSONError(w, "Forbidden", http.StatusForbidden)
			return
		}

		fullPath := filepath.Join(absStaticDir, filepath.FromSlash(cleanPath))

		if !strings.HasPrefix(fullPath, absStaticDir) {
			sendJSONError(w, "Forbidden", http.StatusForbidden)
			return
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			sendJSONError(w, "Not found", http.StatusNotFound)
			return
		}
		if info.IsDir() {
			sendJSONError(w, "Not found", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, fullPath)
	})
}
