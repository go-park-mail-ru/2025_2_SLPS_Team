package handler

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"project/domain"
	"strings"

	"go.uber.org/zap"
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
			sendJSONResponse(w, domain.NotFound, http.StatusNotFound)
			domain.Info(r.Context(), "File not found", zap.String("url", r.URL.Path))
			return
		}

		relPath, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, prefix))
		if err != nil {
			sendJSONResponse(w, "Bad request", http.StatusBadRequest)
			domain.Info(r.Context(), "File not found", zap.String("url", relPath))
			return
		}

		cleanPath := path.Clean("/" + relPath)
		cleanPath = strings.TrimPrefix(cleanPath, "/")

		if strings.Contains(cleanPath, "..") {
			sendJSONResponse(w, domain.Forbidden, http.StatusForbidden)
			domain.Warn(r.Context(), "Try get access to forbidden file", zap.String("cleanPath", cleanPath))
			return
		}

		fullPath := filepath.Join(absStaticDir, filepath.FromSlash(cleanPath))

		if !strings.HasPrefix(fullPath, absStaticDir) {
			sendJSONResponse(w, domain.Forbidden, http.StatusForbidden)
			domain.Warn(r.Context(), "Try get access to forbidden file", zap.String("cleanPath", cleanPath))
			return
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			sendJSONResponse(w, domain.NotFound, http.StatusNotFound)
			domain.Info(r.Context(), "File not found", zap.String("cleanPath", cleanPath))
			return
		}
		if info.IsDir() {
			sendJSONResponse(w, domain.NotFound, http.StatusNotFound)
			domain.Info(r.Context(), "File not found", zap.String("cleanPath", cleanPath))
			return
		}

		http.ServeFile(w, r, fullPath)
	})
}
