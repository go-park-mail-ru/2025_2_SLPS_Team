package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"project/domain"
	"project/internal/service/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestPostsHandler_PostsPaginate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPostService := mocks.NewMockPostService(ctrl)
	handler := NewPostsHandler(mockPostService)

	t.Run("Success", func(t *testing.T) {
		posts := []domain.Post{
			{ID: 1, AuthorID: 1, Text: "First post"},
			{ID: 2, AuthorID: 1, Text: "Second post"},
		}
		totalPages := 5

		mockPostService.EXPECT().
			PostsPaginate(gomock.Any(), 1, 20).
			Return(posts, totalPages, nil)

		req := httptest.NewRequest("GET", "/posts?page=1&limit=20", nil)
		w := httptest.NewRecorder()

		handler.PostsPaginate(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response PostsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Posts, 2)
		assert.Equal(t, totalPages, response.TotalPages)
	})

	t.Run("Invalid query params", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts?page=invalid", nil)
		w := httptest.NewRecorder()

		handler.PostsPaginate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPostsHandler_GetPost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPostService := mocks.NewMockPostService(ctrl)
	handler := NewPostsHandler(mockPostService)

	t.Run("Success", func(t *testing.T) {
		post := &domain.Post{ID: 1, AuthorID: 1, Text: "Test post"}

		mockPostService.EXPECT().
			GetPost(gomock.Any(), uint(1)).
			Return(post, nil)

		req := newRequestWithVars(t, "/posts/1", map[string]string{"id": "1"})
		w := httptest.NewRecorder()

		handler.GetPost(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.Post
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, post.ID, response.ID)
	})

	t.Run("Invalid post ID", func(t *testing.T) {
		req := newRequestWithVars(t, "/posts/invalid", map[string]string{"id": "invalid"})
		w := httptest.NewRecorder()

		handler.GetPost(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPostsHandler_CreatePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPostService := mocks.NewMockPostService(ctrl)
	handler := NewPostsHandler(mockPostService)

	t.Run("Success", func(t *testing.T) {
		post := &domain.Post{ID: 1, AuthorID: 1, Text: "New post"}

		mockPostService.EXPECT().
			CreatePost(gomock.Any(), 1, "Test text", gomock.Any(), gomock.Any()).
			Return(post, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("text", "Test text")
		writer.Close()

		req := httptest.NewRequest("POST", "/posts", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.CreatePost(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Missing text", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.Close()

		req := httptest.NewRequest("POST", "/posts", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.CreatePost(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPostsHandler_DeletePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPostService := mocks.NewMockPostService(ctrl)
	handler := NewPostsHandler(mockPostService)

	t.Run("Success", func(t *testing.T) {
		mockPostService.EXPECT().
			DeletePost(gomock.Any(), uint(1), 1).
			Return(nil)

		req := newRequestWithVars(t, "/posts/1", map[string]string{"id": "1"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeletePost(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := newRequestWithVars(t, "/posts/1", map[string]string{"id": "1"})
		w := httptest.NewRecorder()

		handler.DeletePost(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// Вспомогательная функция для создания запроса с переменными маршрута
func newRequestWithVars(t *testing.T, url string, vars map[string]string) *http.Request {
	req := httptest.NewRequest("GET", url, nil)
	return mux.SetURLVars(req, vars)
}
