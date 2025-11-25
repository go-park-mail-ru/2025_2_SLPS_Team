package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"project/domain"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestRequestConfig конфигурация для создания тестового запроса
type TestRequestConfig struct {
	Method      string
	URL         string
	Vars        map[string]string
	UserID      int32
	Body        interface{}
	ContentType string
	AddAuth     bool // Добавлять ли авторизацию в контекст
}

// NewTestRequest создает тестовый HTTP запрос с контекстом и переменными маршрута
func NewTestRequest(t *testing.T, config TestRequestConfig) *http.Request {
	var req *http.Request

	if config.Body != nil {
		jsonBody, err := json.Marshal(config.Body)
		require.NoError(t, err)
		req = httptest.NewRequest(config.Method, config.URL, bytes.NewBuffer(jsonBody))
	} else {
		req = httptest.NewRequest(config.Method, config.URL, nil)
	}

	// Добавляем переменные маршрута
	if config.Vars != nil {
		req = mux.SetURLVars(req, config.Vars)
	}

	// Создаем базовый контекст с логгером
	ctx := req.Context()
	logger := zap.NewNop()
	ctx = context.WithValue(ctx, domain.LoggerKey, logger)

	// Добавляем userID в контекст если требуется авторизация
	if config.AddAuth && config.UserID != 0 {
		ctx = context.WithValue(ctx, "userID", config.UserID)
		// Также добавляем в доменный контекст
		ctx = context.WithValue(ctx, domain.UserIDKey, config.UserID)
	}

	req = req.WithContext(ctx)

	// Устанавливаем Content-Type
	if config.ContentType != "" {
		req.Header.Set("Content-Type", config.ContentType)
	} else if config.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}
