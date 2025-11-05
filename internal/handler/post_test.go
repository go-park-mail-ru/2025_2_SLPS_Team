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
		avatar := "avatar1.png"

		posts := []domain.PostWithShortUser{
			{
				Post: domain.Post{
					ID:       1,
					AuthorID: 1,
					Text:     "First post",
				},
				Author: domain.ShortProfile{
					UserID:     1,
					FullName:   "John Doe",
					AvatarPath: &avatar,
				},
			},
			{
				Post: domain.Post{
					ID:       2,
					AuthorID: 2,
					Text:     "Second post",
				},
				Author: domain.ShortProfile{
					UserID:     2,
					FullName:   "Jane Smith",
					AvatarPath: nil,
				},
			},
		}

		mockPostService.EXPECT().
			PostsPaginate(gomock.Any(), domain.PaginateQueryParams{
				Page:  1,
				Limit: 20,
			}).
			Return(posts, nil)

		req := httptest.NewRequest("GET", "/posts?page=1&limit=20", nil)
		w := httptest.NewRecorder()

		handler.PostsPaginate(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []domain.PostWithShortUser
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)

		assert.Equal(t, posts[0].Post.ID, response[0].Post.ID)
		assert.Equal(t, posts[0].Author.FullName, response[0].Author.FullName)
		assert.Equal(t, posts[1].Post.Text, response[1].Post.Text)
	})

	t.Run("Invalid query params", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts?page=invalid", nil)
		w := httptest.NewRecorder()

		handler.PostsPaginate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service returns error", func(t *testing.T) {
		mockPostService.EXPECT().
			PostsPaginate(gomock.Any(), gomock.Any()).
			Return(nil, domain.ErrDB)

		req := httptest.NewRequest("GET", "/posts?page=1&limit=20", nil)
		w := httptest.NewRecorder()

		handler.PostsPaginate(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
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

	t.Run("Post not found", func(t *testing.T) {
		mockPostService.EXPECT().
			GetPost(gomock.Any(), uint(1)).
			Return(nil, domain.ErrNotFound)

		req := newRequestWithVars(t, "/posts/1", map[string]string{"id": "1"})
		w := httptest.NewRecorder()

		handler.GetPost(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
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

		var response JSONResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Post created successfully", response.Message)
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

	t.Run("Unauthorized", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("text", "Test text")
		writer.Close()

		req := httptest.NewRequest("POST", "/posts", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()

		handler.CreatePost(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestPostsHandler_UpdatePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPostService := mocks.NewMockPostService(ctrl)
	handler := NewPostsHandler(mockPostService)

	t.Run("Success", func(t *testing.T) {
		mockPostService.EXPECT().
			UpdatePost(gomock.Any(), uint(1), 1, "Updated text", gomock.Any(), gomock.Any()).
			Return(nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("text", "Updated text")
		writer.Close()

		req := httptest.NewRequest("PUT", "/posts/1", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "1"})

		w := httptest.NewRecorder()

		handler.UpdatePost(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid post ID", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/posts/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.UpdatePost(w, req)

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

	t.Run("Invalid post ID", func(t *testing.T) {
		req := newRequestWithVars(t, "/posts/invalid", map[string]string{"id": "invalid"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeletePost(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := newRequestWithVars(t, "/posts/1", map[string]string{"id": "1"})
		w := httptest.NewRecorder()

		handler.DeletePost(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestPostsHandler_GetUserPosts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPostService := mocks.NewMockPostService(ctrl)
	handler := NewPostsHandler(mockPostService)

	t.Run("Success", func(t *testing.T) {
		posts := []domain.Post{
			{ID: 1, AuthorID: 1, Text: "User post 1"},
			{ID: 2, AuthorID: 1, Text: "User post 2"},
		}

		mockPostService.EXPECT().
			GetUserPosts(gomock.Any(), uint(1), domain.PaginateQueryParams{
				Page:  1,
				Limit: 20,
			}).
			Return(posts, nil)

		req := newRequestWithVars(t, "/users/1/posts?page=1&limit=20", map[string]string{"userID": "1"})
		w := httptest.NewRecorder()

		handler.GetUserPosts(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []domain.Post
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("Invalid user ID", func(t *testing.T) {
		req := newRequestWithVars(t, "/users/invalid/posts", map[string]string{"userID": "invalid"})
		w := httptest.NewRecorder()

		handler.GetUserPosts(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid query params", func(t *testing.T) {
		req := newRequestWithVars(t, "/users/1/posts?page=invalid", map[string]string{"userID": "1"})
		w := httptest.NewRecorder()

		handler.GetUserPosts(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Вспомогательная функция для создания запроса с переменными маршрута
func newRequestWithVars(t *testing.T, url string, vars map[string]string) *http.Request {
	req := httptest.NewRequest("GET", url, nil)
	return mux.SetURLVars(req, vars)
}
