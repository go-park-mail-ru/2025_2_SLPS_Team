package handler

import (
	"context"
	"errors"
	"net/http"
	"os"
	"project/domain"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("FRONTEND_ORIGIN"))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func SecureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; object-src 'none';")

		next.ServeHTTP(w, r)
	})
}

var ForbiddenPathsWithAuth = map[string]bool{
	"/api/auth/login":    true,
	"/api/auth/register": true,
}
var AllowedPathsWithOutAuth = map[string]bool{
	"/api/auth/login":      true,
	"/api/auth/register":   true,
	"/api/auth/isloggedin": true,
}

func (api *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path
		userID, err := api.IsLoggedIn(r)
		isLoggedIn := true
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				isLoggedIn = false
			} else {
				sendJSONSuccess(w, "Server error", http.StatusInternalServerError)
				return
			}
		}
		if isLoggedIn {
			if ForbiddenPathsWithAuth[path] {
				sendJSONSuccess(w, "Forbidden", http.StatusForbidden)
				return
			} else {
				ctx := context.WithValue(r.Context(), domain.UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
				//userID, ok := r.Context().Value(userIDKey).(int)
			}
		} else {
			if !AllowedPathsWithOutAuth[path] {
				sendJSONSuccess(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
func LoggingMiddleware(logger *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := uuid.New().String()

			reqLogger := logger.With(zap.String("requestID", reqID))
			if userID, ok := r.Context().Value(domain.UserIDKey).(int); ok {
				reqLogger = reqLogger.With(zap.Int("selfUserID", userID))
			}
			ctx := context.WithValue(r.Context(), domain.LoggerKey, reqLogger)

			reqLogger.Info("incoming request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
			)
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(ww, r.WithContext(ctx))
			duration := time.Since(start)
			reqLogger.Info("request completed", zap.Duration("duration", duration), zap.Int("status", ww.statusCode))
		})
	}
}
