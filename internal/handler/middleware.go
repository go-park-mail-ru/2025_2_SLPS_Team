package handler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"project/config"
	"project/domain"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", config.GetConfig().FrontendOrigin)
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
var SafeMethods = map[string]bool{"GET": true, "HEAD": true, "OPTIONS": true, "TRACE": true}

func (api *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path
		session, err := api.IsLoggedIn(r)
		isLoggedIn := true
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				isLoggedIn = false
			} else {
				sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
				domain.FromContext(r.Context()).Error("Fail to get IsLoggedIn", zap.Error(err))
				return
			}
		}
		if isLoggedIn {
			if ForbiddenPathsWithAuth[path] {
				sendJSONResponse(w, domain.Forbidden, http.StatusForbidden)
				domain.FromContext(r.Context()).Warn("Try get access to forbidden path")
				return
			} else {
				ctx := context.WithValue(r.Context(), domain.UserIDKey, session.UserID)
				newLogger := domain.FromContext(ctx).With(zap.Int("selfUserID", session.UserID))
				ctx = context.WithValue(ctx, domain.LoggerKey, newLogger)
				domain.FromContext(ctx).Info("User logged in, add userID to context")

				if !SafeMethods[r.Method] && !config.GetConfig().Debug {
					domain.FromContext(r.Context()).Info("in header", zap.String("scrf", r.Header.Get("X-CSRF-Token")))
					domain.FromContext(r.Context()).Info("in session", zap.String("scrf", session.CSRFToken))
					if r.Header.Get("X-CSRF-Token") != session.CSRFToken {
						sendJSONResponse(w, domain.Forbidden, http.StatusForbidden)
						domain.FromContext(r.Context()).Warn("Try do somthing without CSRF token")
						return
					}
				}

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		} else {
			if !AllowedPathsWithOutAuth[path] {
				sendJSONResponse(w, domain.Forbidden, http.StatusForbidden)
				domain.FromContext(r.Context()).Warn("Try get access to forbidden path")
				return
			}
		}
		next.ServeHTTP(w, r)
		return
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
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
	}
	return hj.Hijack()
}
func LoggingMiddleware(logger *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := uuid.New().String()

			reqLogger := logger.With(zap.String("requestID", reqID))
			//if userID, ok := r.Context().Value(domain.UserIDKey).(int); ok {
			//    reqLogger = reqLogger.With(zap.Int("selfUserID", userID))
			//}
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
