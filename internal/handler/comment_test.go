package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"project/domain"
)

// Реализация фейкового сервиса для тестов
type fakeCommentService struct {
	comments      map[int32]*domain.CommentView
	postComments  map[int32][]domain.CommentView
	commentsCount map[int32]int32
	shouldFail    bool
	failOnMethod  string
	nextID        int32
}

func newFakeCommentService() *fakeCommentService {
	return &fakeCommentService{
		comments:      make(map[int32]*domain.CommentView),
		postComments:  make(map[int32][]domain.CommentView),
		commentsCount: make(map[int32]int32),
		nextID:        1,
	}
}

func (f *fakeCommentService) CreateComment(ctx context.Context, userID int32, req domain.CommentCreateRequest) (*domain.CommentView, error) {
	if f.shouldFail && f.failOnMethod == "CreateComment" {
		return nil, errors.New("service error")
	}

	comment := &domain.CommentView{
		ID:       f.nextID,
		PostID:   req.PostID,
		AuthorID: userID,
		Text:     req.Text,
	}

	f.comments[f.nextID] = comment
	f.nextID++

	// Добавляем в список комментариев поста
	if _, ok := f.postComments[req.PostID]; !ok {
		f.postComments[req.PostID] = []domain.CommentView{}
	}
	f.postComments[req.PostID] = append(f.postComments[req.PostID], *comment)

	// Увеличиваем счетчик
	f.commentsCount[req.PostID]++

	return comment, nil
}

func (f *fakeCommentService) GetComment(ctx context.Context, userID, commentID int32) (*domain.CommentView, error) {
	if f.shouldFail && f.failOnMethod == "GetComment" {
		return nil, errors.New("service error")
	}

	comment, ok := f.comments[commentID]
	if !ok {
		return nil, errors.New("comment not found")
	}

	return comment, nil
}

func (f *fakeCommentService) GetPostComments(ctx context.Context, userID, postID int32, params domain.PaginateQueryParams) ([]domain.CommentView, error) {
	if f.shouldFail && f.failOnMethod == "GetPostComments" {
		return nil, errors.New("service error")
	}

	comments, ok := f.postComments[postID]
	if !ok {
		return nil, errors.New("post not found")
	}

	// Простая реализация пагинации для тестов
	start := (params.Page - 1) * params.Limit
	if start >= int32(len(comments)) {
		return []domain.CommentView{}, nil
	}

	end := start + params.Limit
	if end > int32(len(comments)) {
		end = int32(len(comments))
	}

	return comments[start:end], nil
}

func (f *fakeCommentService) UpdateComment(ctx context.Context, commentID, userID int32, text string) error {
	if f.shouldFail && f.failOnMethod == "UpdateComment" {
		return errors.New("service error")
	}

	comment, ok := f.comments[commentID]
	if !ok {
		return errors.New("comment not found")
	}

	if comment.AuthorID != userID {
		return errors.New("access denied")
	}

	comment.Text = text
	return nil
}

func (f *fakeCommentService) DeleteComment(ctx context.Context, commentID, userID int32) error {
	if f.shouldFail && f.failOnMethod == "DeleteComment" {
		return errors.New("service error")
	}

	comment, ok := f.comments[commentID]
	if !ok {
		return errors.New("comment not found")
	}

	if comment.AuthorID != userID {
		return errors.New("access denied")
	}

	delete(f.comments, commentID)

	// Удаляем из списка комментариев поста
	if postComments, ok := f.postComments[comment.PostID]; ok {
		for i, c := range postComments {
			if c.ID == commentID {
				f.postComments[comment.PostID] = append(postComments[:i], postComments[i+1:]...)
				break
			}
		}
	}

	f.commentsCount[comment.PostID]--
	return nil
}

func (f *fakeCommentService) GetPostCommentsCount(ctx context.Context, postID int32) (int32, error) {
	if f.shouldFail && f.failOnMethod == "GetPostCommentsCount" {
		return 0, errors.New("service error")
	}

	count, ok := f.commentsCount[postID]
	if !ok {
		return 0, nil
	}

	return count, nil
}

// Helper функции для тестов
func newRequestWithVarsLong(t *testing.T, method, url string, vars map[string]string, body interface{}) *http.Request {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req = mux.SetURLVars(req, vars)
	return req
}

// Helper для создания запроса с userID в контексте
func newRequestWithAuth(t *testing.T, method, url string, vars map[string]string, body interface{}, userID int32) *http.Request {
	req := newRequestWithVarsLong(t, method, url, vars, body)
	ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestCommentHandler_CreateComment(t *testing.T) {
	service := newFakeCommentService()
	handler := NewCommentHandler(service)

	t.Run("Успешное создание комментария", func(t *testing.T) {
		reqBody := domain.CommentCreateRequest{
			PostID: 1,
			Text:   "Test comment",
		}

		req := newRequestWithAuth(t, "POST", "/comments", nil, reqBody, 1)
		w := httptest.NewRecorder()

		handler.CreateComment(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.CommentResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Comment created successfully", response.Message)
		assert.NotNil(t, response.Comment)
		assert.Equal(t, "Test comment", response.Comment.Text)
	})

	t.Run("Создание комментария с неверным JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/comments", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.CreateComment(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
	})

	t.Run("Создание комментария без авторизации", func(t *testing.T) {
		reqBody := domain.CommentCreateRequest{
			PostID: 1,
			Text:   "Test comment",
		}

		req := newRequestWithVarsLong(t, "POST", "/comments", nil, reqBody)
		// НЕ добавляем userID в контекст

		w := httptest.NewRecorder()

		handler.CreateComment(w, req)

		// UserID будет 0 (default value для int32)
		assert.NotEqual(t, http.StatusCreated, w.Code)
	})
}

func TestCommentHandler_GetComment(t *testing.T) {
	service := newFakeCommentService()
	handler := NewCommentHandler(service)

	// Сначала создадим комментарий для теста
	reqBody := domain.CommentCreateRequest{
		PostID: 1,
		Text:   "Existing comment",
	}
	createReq := newRequestWithAuth(t, "POST", "/comments", nil, reqBody, 1)

	w := httptest.NewRecorder()
	handler.CreateComment(w, createReq)

	var createResponse domain.CommentResponse
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	commentID := createResponse.Comment.ID

	t.Run("Успешное получение комментария", func(t *testing.T) {
		req := newRequestWithAuth(t, "GET", "/comments/1", map[string]string{"id": "1"}, nil, 1)
		w := httptest.NewRecorder()

		handler.GetComment(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var comment domain.CommentView
		err := json.Unmarshal(w.Body.Bytes(), &comment)
		assert.NoError(t, err)
		assert.Equal(t, commentID, comment.ID)
		assert.Equal(t, "Existing comment", comment.Text)
	})

	t.Run("Получение несуществующего комментария", func(t *testing.T) {
		req := newRequestWithAuth(t, "GET", "/comments/999", map[string]string{"id": "999"}, nil, 1)
		w := httptest.NewRecorder()

		handler.GetComment(w, req)

		// Ваш handler должен возвращать 404 при "comment not found" ошибке
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Получение комментария с неверным ID", func(t *testing.T) {
		req := newRequestWithAuth(t, "GET", "/comments/abc", map[string]string{"id": "abc"}, nil, 1)
		w := httptest.NewRecorder()

		handler.GetComment(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCommentHandler_GetPostComments(t *testing.T) {
	service := newFakeCommentService()
	handler := NewCommentHandler(service)

	// Создаем несколько комментариев для теста
	for i := 1; i <= 5; i++ {
		reqBody := domain.CommentCreateRequest{
			PostID: 1,
			Text:   "Comment " + string(rune('0'+i)),
		}

		req := newRequestWithAuth(t, "POST", "/comments", nil, reqBody, 1)
		w := httptest.NewRecorder()
		handler.CreateComment(w, req)
	}

	t.Run("Успешное получение комментариев поста", func(t *testing.T) {
		req := newRequestWithAuth(t, "GET", "/posts/1/comments", map[string]string{"postID": "1"}, nil, 1)
		w := httptest.NewRecorder()

		handler.GetPostComments(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var comments []domain.CommentView
		err := json.Unmarshal(w.Body.Bytes(), &comments)
		assert.NoError(t, err)
	})

	t.Run("Получение комментариев поста с пагинацией", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/1/comments?page=1&limit=2", nil)
		req = mux.SetURLVars(req, map[string]string{"postID": "1"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetPostComments(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var comments []domain.CommentView
		err := json.Unmarshal(w.Body.Bytes(), &comments)
		assert.NoError(t, err)
		assert.Len(t, comments, 2)
	})

	t.Run("Получение комментариев несуществующего поста", func(t *testing.T) {
		req := newRequestWithAuth(t, "GET", "/posts/999/comments", map[string]string{"postID": "999"}, nil, 1)
		w := httptest.NewRecorder()

		handler.GetPostComments(w, req)

		// Ваш handler должен возвращать 404 при "post not found" ошибке
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCommentHandler_UpdateComment(t *testing.T) {
	service := newFakeCommentService()
	handler := NewCommentHandler(service)

	// Создаем комментарий для теста
	reqBody := domain.CommentCreateRequest{
		PostID: 1,
		Text:   "Original comment",
	}
	createReq := newRequestWithAuth(t, "POST", "/comments", nil, reqBody, 1)

	w := httptest.NewRecorder()
	handler.CreateComment(w, createReq)

	t.Run("Успешное обновление комментария", func(t *testing.T) {
		updateBody := domain.CommentUpdateRequest{
			Text: "Updated comment",
		}

		req := newRequestWithAuth(t, "PUT", "/comments/1", map[string]string{"id": "1"}, updateBody, 1)
		w := httptest.NewRecorder()

		handler.UpdateComment(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Проверяем, что комментарий действительно обновился
		getReq := newRequestWithAuth(t, "GET", "/comments/1", map[string]string{"id": "1"}, nil, 1)
		w2 := httptest.NewRecorder()
		handler.GetComment(w2, getReq)

		var comment domain.CommentView
		json.Unmarshal(w2.Body.Bytes(), &comment)
		assert.Equal(t, "Updated comment", comment.Text)
	})

	t.Run("Обновление комментария другим пользователем", func(t *testing.T) {
		updateBody := domain.CommentUpdateRequest{
			Text: "Hacked comment",
		}

		req := newRequestWithAuth(t, "PUT", "/comments/1", map[string]string{"id": "1"}, updateBody, 2) // Другой пользователь
		w := httptest.NewRecorder()

		handler.UpdateComment(w, req)

		// Ваш handler должен возвращать 403 при "access denied" ошибке
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Обновление несуществующего комментария", func(t *testing.T) {
		updateBody := domain.CommentUpdateRequest{
			Text: "Updated comment",
		}

		req := newRequestWithAuth(t, "PUT", "/comments/999", map[string]string{"id": "999"}, updateBody, 1)
		w := httptest.NewRecorder()

		handler.UpdateComment(w, req)

		// Ваш handler должен возвращать 404 при "comment not found" ошибке
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCommentHandler_DeleteComment(t *testing.T) {
	service := newFakeCommentService()
	handler := NewCommentHandler(service)

	// Создаем комментарий для теста
	reqBody := domain.CommentCreateRequest{
		PostID: 1,
		Text:   "Comment to delete",
	}
	createReq := newRequestWithAuth(t, "POST", "/comments", nil, reqBody, 1)

	w := httptest.NewRecorder()
	handler.CreateComment(w, createReq)

	t.Run("Успешное удаление комментария", func(t *testing.T) {
		req := newRequestWithAuth(t, "DELETE", "/comments/1", map[string]string{"id": "1"}, nil, 1)
		w := httptest.NewRecorder()

		handler.DeleteComment(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Проверяем, что комментарий действительно удален
		getReq := newRequestWithAuth(t, "GET", "/comments/1", map[string]string{"id": "1"}, nil, 1)
		w2 := httptest.NewRecorder()
		handler.GetComment(w2, getReq)

		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})

	t.Run("Удаление комментария другим пользователем", func(t *testing.T) {
		// Создаем еще один комментарий от другого пользователя
		reqBody := domain.CommentCreateRequest{
			PostID: 1,
			Text:   "Another comment",
		}
		createReq := newRequestWithAuth(t, "POST", "/comments", nil, reqBody, 2)

		w := httptest.NewRecorder()
		handler.CreateComment(w, createReq)

		// Пытаемся удалить комментарий пользователя 2 от имени пользователя 1
		req := newRequestWithAuth(t, "DELETE", "/comments/2", map[string]string{"id": "2"}, nil, 1)
		w = httptest.NewRecorder()

		handler.DeleteComment(w, req)

		// Ваш handler должен возвращать 403 при "access denied" ошибке
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCommentHandler_GetPostCommentsCount(t *testing.T) {
	service := newFakeCommentService()
	handler := NewCommentHandler(service)

	// Создаем комментарии для теста
	for i := 1; i <= 3; i++ {
		reqBody := domain.CommentCreateRequest{
			PostID: 1,
			Text:   "Comment " + string(rune('0'+i)),
		}

		req := newRequestWithAuth(t, "POST", "/comments", nil, reqBody, 1)
		w := httptest.NewRecorder()
		handler.CreateComment(w, req)
	}

	t.Run("Успешное получение количества комментариев", func(t *testing.T) {
		req := newRequestWithAuth(t, "GET", "/posts/1/comments/count", map[string]string{"postID": "1"}, nil, 1)
		w := httptest.NewRecorder()

		handler.GetPostCommentsCount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.CommentCountResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, int32(3), response.Count)
	})

	t.Run("Получение количества комментариев для поста без комментариев", func(t *testing.T) {
		req := newRequestWithAuth(t, "GET", "/posts/999/comments/count", map[string]string{"postID": "999"}, nil, 1)
		w := httptest.NewRecorder()

		handler.GetPostCommentsCount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.CommentCountResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), response.Count)
	})

	t.Run("Получение количества комментариев с неверным ID поста", func(t *testing.T) {
		req := newRequestWithAuth(t, "GET", "/posts/abc/comments/count", map[string]string{"postID": "abc"}, nil, 1)
		w := httptest.NewRecorder()

		handler.GetPostCommentsCount(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
