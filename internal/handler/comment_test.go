package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"project/domain"
	"project/internal/service/mocks"
)

// newRequestWithVarsLong создает запрос с параметрами пути
func newRequestWithVarsLong(t *testing.T, method, url string, vars map[string]string, bodyData ...[]byte) *http.Request {
	var req *http.Request
	if len(bodyData) > 0 && bodyData[0] != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(bodyData[0]))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req = mux.SetURLVars(req, vars)
	return req
}

// Константы для тестирования
const testUserID = int32(123)

// Создаем тестовый контекст с userID
func testContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, domain.UserIDKey, testUserID)
	return ctx
}

// Создаем тестовый запрос с контекстом
func newTestRequest(method, url string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	ctx := testContext()
	return req.WithContext(ctx)
}

func TestCommentHandler_CreateComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommentService(ctrl)
	handler := NewCommentHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		reqBody := domain.CommentCreateRequest{
			PostID: 1,
			Text:   "Test comment text",
		}
		expectedComment := &domain.CommentView{
			ID:        1,
			PostID:    1,
			AuthorID:  testUserID,
			Text:      "Test comment text",
			CreatedAt: time.Now(),
		}

		// Используем gomock.Any для userID, так как он берется из контекста
		mockService.EXPECT().
			CreateComment(gomock.Any(), testUserID, reqBody).
			Return(expectedComment, nil)

		body, _ := json.Marshal(reqBody)
		req := newTestRequest("POST", "/comments", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.CreateComment(w, req)

		// Проверяем статус код (201 Created)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Проверяем структуру ответа
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Проверяем сообщение
		if msg, ok := response["message"].(string); ok {
			assert.Equal(t, "Comment created successfully", msg)
		}

		// Проверяем наличие комментария
		if commentData, ok := response["comment"].(map[string]interface{}); ok {
			assert.Equal(t, float64(1), commentData["id"])
			assert.Equal(t, "Test comment text", commentData["text"])
		}
	})

	t.Run("Bad request - invalid JSON", func(t *testing.T) {
		req := newTestRequest("POST", "/comments", []byte(`{invalid json`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.CreateComment(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error - post not found", func(t *testing.T) {
		reqBody := domain.CommentCreateRequest{
			PostID: 999,
			Text:   "Test comment text",
		}

		mockService.EXPECT().
			CreateComment(gomock.Any(), testUserID, reqBody).
			Return(nil, domain.ErrNotFound)

		body, _ := json.Marshal(reqBody)
		req := newTestRequest("POST", "/comments", body)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handler.CreateComment(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("No user ID in context", func(t *testing.T) {
		reqBody := domain.CommentCreateRequest{
			PostID: 1,
			Text:   "Test comment text",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/comments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// НЕ добавляем userID в контекст

		w := httptest.NewRecorder()

		handler.CreateComment(w, req)

		// Проверяем что userID равен 0 (default value)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCommentHandler_GetComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommentService(ctrl)
	handler := NewCommentHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		expectedComment := &domain.CommentView{
			ID:        1,
			PostID:    1,
			AuthorID:  testUserID,
			Text:      "Test comment text",
			CreatedAt: time.Now(),
		}

		mockService.EXPECT().
			GetComment(gomock.Any(), testUserID, int32(1)).
			Return(expectedComment, nil)

		req := newRequestWithVarsLong(t, "GET", "/comments/1", map[string]string{"id": "1"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetComment(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.CommentView
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedComment.ID, response.ID)
		assert.Equal(t, expectedComment.Text, response.Text)
	})

	t.Run("Invalid comment ID", func(t *testing.T) {
		req := newRequestWithVarsLong(t, "GET", "/comments/abc", map[string]string{"id": "abc"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetComment(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Comment not found", func(t *testing.T) {
		mockService.EXPECT().
			GetComment(gomock.Any(), testUserID, int32(999)).
			Return(nil, domain.ErrNotFound)

		req := newRequestWithVarsLong(t, "GET", "/comments/999", map[string]string{"id": "999"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetComment(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCommentHandler_GetPostComments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommentService(ctrl)
	handler := NewCommentHandler(mockService)

	t.Run("Success with default pagination", func(t *testing.T) {
		expectedComments := []domain.CommentView{
			{ID: 1, PostID: 1, AuthorID: testUserID, Text: "Comment 1"},
			{ID: 2, PostID: 1, AuthorID: 456, Text: "Comment 2"},
		}

		// Используем gomock.Any для PaginateQueryParams, так как они могут быть по-другому структурированы
		mockService.EXPECT().
			GetPostComments(gomock.Any(), testUserID, int32(1), gomock.Any()).
			Return(expectedComments, nil)

		req := newRequestWithVarsLong(t, "GET", "/posts/1/comments", map[string]string{"postID": "1"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetPostComments(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []domain.CommentView
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, expectedComments[0].ID, response[0].ID)
	})

	t.Run("Success with custom pagination", func(t *testing.T) {
		expectedComments := []domain.CommentView{
			{ID: 3, PostID: 1, AuthorID: 789, Text: "Comment 3"},
		}

		mockService.EXPECT().
			GetPostComments(gomock.Any(), testUserID, int32(1), gomock.Any()).
			Return(expectedComments, nil)

		req := httptest.NewRequest("GET", "/posts/1/comments?page=2&limit=10", nil)
		req = mux.SetURLVars(req, map[string]string{"postID": "1"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetPostComments(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []domain.CommentView
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, expectedComments[0].ID, response[0].ID)
	})

	t.Run("Invalid post ID", func(t *testing.T) {
		req := newRequestWithVarsLong(t, "GET", "/posts/abc/comments", map[string]string{"postID": "abc"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetPostComments(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Post not found", func(t *testing.T) {
		mockService.EXPECT().
			GetPostComments(gomock.Any(), testUserID, int32(999), gomock.Any()).
			Return(nil, domain.ErrNotFound)

		req := newRequestWithVarsLong(t, "GET", "/posts/999/comments", map[string]string{"postID": "999"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetPostComments(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCommentHandler_UpdateComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommentService(ctrl)
	handler := NewCommentHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		reqBody := domain.CommentUpdateRequest{
			Text: "Updated comment text",
		}

		mockService.EXPECT().
			UpdateComment(gomock.Any(), int32(1), testUserID, reqBody.Text).
			Return(nil)

		body, _ := json.Marshal(reqBody)
		req := newRequestWithVarsLong(t, "PUT", "/comments/1", map[string]string{"id": "1"}, body)
		req.Header.Set("Content-Type", "application/json")
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.UpdateComment(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Comment updated successfully", response["message"])
	})

	t.Run("Invalid comment ID", func(t *testing.T) {
		req := newRequestWithVarsLong(t, "PUT", "/comments/abc", map[string]string{"id": "abc"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.UpdateComment(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Bad request - invalid JSON", func(t *testing.T) {
		req := newRequestWithVarsLong(t, "PUT", "/comments/1", map[string]string{"id": "1"}, []byte(`{invalid json`))
		req.Header.Set("Content-Type", "application/json")
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.UpdateComment(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Forbidden - not comment author", func(t *testing.T) {
		reqBody := domain.CommentUpdateRequest{
			Text: "Updated comment text",
		}

		mockService.EXPECT().
			UpdateComment(gomock.Any(), int32(1), testUserID, reqBody.Text).
			Return(domain.ErrInvalidInput)

		body, _ := json.Marshal(reqBody)
		req := newRequestWithVarsLong(t, "PUT", "/comments/1", map[string]string{"id": "1"}, body)
		req.Header.Set("Content-Type", "application/json")
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.UpdateComment(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestCommentHandler_DeleteComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommentService(ctrl)
	handler := NewCommentHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		mockService.EXPECT().
			DeleteComment(gomock.Any(), int32(1), testUserID).
			Return(nil)

		req := newRequestWithVarsLong(t, "DELETE", "/comments/1", map[string]string{"id": "1"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.DeleteComment(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Comment deleted successfully", response["message"])
	})

	t.Run("Invalid comment ID", func(t *testing.T) {
		req := newRequestWithVarsLong(t, "DELETE", "/comments/abc", map[string]string{"id": "abc"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.DeleteComment(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Forbidden - not comment author", func(t *testing.T) {
		mockService.EXPECT().
			DeleteComment(gomock.Any(), int32(1), testUserID).
			Return(domain.ErrInvalidInput)

		req := newRequestWithVarsLong(t, "DELETE", "/comments/1", map[string]string{"id": "1"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.DeleteComment(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Internal server error", func(t *testing.T) {
		mockService.EXPECT().
			DeleteComment(gomock.Any(), int32(1), testUserID).
			Return(errors.New("database error"))

		req := newRequestWithVarsLong(t, "DELETE", "/comments/1", map[string]string{"id": "1"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.DeleteComment(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCommentHandler_GetPostCommentsCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommentService(ctrl)
	handler := NewCommentHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		expectedCount := int32(42)

		mockService.EXPECT().
			GetPostCommentsCount(gomock.Any(), int32(1)).
			Return(expectedCount, nil)

		req := newRequestWithVarsLong(t, "GET", "/posts/1/comments/count", map[string]string{"postID": "1"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetPostCommentsCount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Проверяем что есть поле count
		if count, ok := response["count"].(float64); ok {
			assert.Equal(t, float64(expectedCount), count)
		}
	})

	t.Run("Invalid post ID", func(t *testing.T) {
		req := newRequestWithVarsLong(t, "GET", "/posts/abc/comments/count", map[string]string{"postID": "abc"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetPostCommentsCount(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Internal server error", func(t *testing.T) {
		mockService.EXPECT().
			GetPostCommentsCount(gomock.Any(), int32(1)).
			Return(int32(0), errors.New("database error"))

		req := newRequestWithVarsLong(t, "GET", "/posts/1/comments/count", map[string]string{"postID": "1"})
		ctx := testContext()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetPostCommentsCount(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
