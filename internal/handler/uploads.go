package handler

import (
	"net/http"
	"path/filepath"
	"project/internal/service"
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
		fullPath, err := service.Uploads(r.Context(), absStaticDir, prefix, r.URL.Path)
		if err != nil {
			sendJSONError(w, err)
			return
		}

		http.ServeFile(w, r, *fullPath)

	})
}
