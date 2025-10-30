package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"project/domain"
	"project/internal/service"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type PostsHandler struct {
	postStore domain.PostStore
	userStore domain.UserStore
}

func NewPostsHandler(postStore domain.PostStore, userStore domain.UserStore) *PostsHandler {
	return &PostsHandler{
		postStore: postStore,
		userStore: userStore,
	}
}

// PostsRequest - запрос для пагинации постов
type PostsRequest struct {
	Page  int `schema:"page"`
	Limit int `schema:"limit"`
}

// PostsResponse - ответ с постами и пагинацией
type PostsResponse struct {
	Posts      []domain.Post `json:"posts"`
	Page       int           `json:"page"`
	TotalPages int           `json:"totalPages"`
	HasNext    bool          `json:"hasNext"`
}

// CreatePostRequest - запрос на создание поста
type CreatePostRequest struct {
	Text        string   `json:"text" valid:"required,stringlength(24|4096)"`
	Attachments []string `json:"attachments"`
	Photos      []string `json:"photos"`
}

// UpdatePostRequest - запрос на обновление поста
type UpdatePostRequest struct {
	Text        string   `json:"text" valid:"required,stringlength(24|4096)"`
	Attachments []string `json:"attachments"`
	Photos      []string `json:"photos"`
}

// PostsPaginate - получение постов с пагинацией (публичный endpoint)
func (h *PostsHandler) PostsPaginate(w http.ResponseWriter, r *http.Request) {
	var req PostsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONError(w, domain.InvalidParams, http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	// Устанавливаем значения по умолчанию
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	service.Info(r.Context(), "Getting paginated posts", zap.Int("page", req.Page), zap.Int("limit", req.Limit))
	// Получаем посты из хранилища
	posts, totalPages, err := h.postStore.PostsPaginatedList(r.Context(), req.Page, req.Limit)
	if err != nil {
		service.Error(r.Context(), "Failed to get posts", err)
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := PostsResponse{
		Posts:      posts,
		Page:       req.Page,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", "PostsResponse"))
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	service.Info(r.Context(), "Posts retrieved successfully", zap.Int("postsCount", len(posts)), zap.Int("totalPages", totalPages))
}

// GetPost - получение конкретного поста по ID
// @Summary Получить пост по ID
// @Description Возвращает пост по его идентификатору
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int true "ID поста"
// @Success 200 {object} domain.Post
// @Failure 400 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /posts/{id} [get]
func (h *PostsHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		sendJSONError(w, "Invalid post ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	service.Info(r.Context(), "Getting post by ID", zap.Uint64("postID", postID))

	// Бизнес-логика: получение поста
	post, err := h.postStore.GetPostByID(r.Context(), uint(postID))
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			sendJSONError(w, "Post not found", http.StatusNotFound)
			service.Warn(r.Context(), "Post not found", zap.Uint64("postID", postID))
		} else {
			service.Error(r.Context(), "Failed to get post", err, zap.Uint64("postID", postID))
			sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(post); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", "Post"))
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}
}

// CreatePost - создание нового поста
// @Summary Создать новый пост
// @Description Создает новый пост от имени текущего пользователя
// @Tags posts
// @Accept json
// @Produce json
// @Param post body CreatePostRequest true "Данные поста"
// @Success 201 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 401 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /posts [post]
func (h *PostsHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, domain.InvalidJSON, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", "CreatePostRequest"))
		return
	}

	// Бизнес-логика: валидация данных
	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		sendJSONError(w, domain.InvalidData, http.StatusBadRequest)
		service.Warn(r.Context(), "Create post validation failed ", zap.Error(err))
		return
	}

	// Получаем userID из контекста (установлен в auth middleware)
	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Creating new post", zap.Int("userID", userID))

	// Бизнес-логика: создание объекта поста
	post := &domain.Post{
		AuthorID:    uint(userID),
		Text:        req.Text,
		Attachments: req.Attachments,
		PhotosPath:  req.Photos,
	}

	// Сохраняем пост в хранилище
	if err := h.postStore.CreatePost(r.Context(), post); err != nil {
		service.Error(r.Context(), "Failed to create post", err, zap.Int("userID", userID))
		sendJSONError(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Post created successfully", http.StatusCreated)
	service.Info(r.Context(), "Post created successfully", zap.Uint("postID", post.ID), zap.Int("userID", userID))
}

// UpdatePost - обновление поста
// @Summary Обновить пост
// @Description Обновляет существующий пост (только автор)
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int true "ID поста"
// @Param post body UpdatePostRequest true "Обновленные данные поста"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 401 {object} JSONResponse
// @Failure 403 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /posts/{id} [put]
func (h *PostsHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		sendJSONError(w, "Invalid post ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	var req UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, domain.InvalidJSON, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", "UpdatePostRequest"))
		return
	}

	// Бизнес-логика: валидация данных
	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		sendJSONError(w, domain.InvalidData, http.StatusBadRequest)
		service.Warn(r.Context(), "Update post validation failed", zap.Error(err))
		return
	}
	// Получаем userID из контекста
	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	// Бизнес-логика: проверяем существование поста и права доступа
	service.Info(r.Context(), "Updating post", zap.Uint64("postID", postID), zap.Int("userID", userID))

	existingPost, err := h.postStore.GetPostByID(r.Context(), uint(postID))
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			sendJSONError(w, "Post not found", http.StatusNotFound)
			service.Warn(r.Context(), "Post not found for update", zap.Uint64("postID", postID))
		} else {
			service.Error(r.Context(), "Failed to get post for update", err, zap.Uint64("postID", postID))

			sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Проверяем, что пользователь является автором поста
	if existingPost.AuthorID != uint(userID) {
		sendJSONError(w, domain.Forbidden, http.StatusForbidden)
		service.Warn(r.Context(), "Access denied: user is not post author",
			zap.Uint64("postID", postID),
			zap.Int("userID", userID),
			zap.Uint("authorID", existingPost.AuthorID))
		return
	}

	// Обновляем данные поста
	updatedPost := &domain.Post{
		ID:          uint(postID),
		AuthorID:    uint(userID),
		Text:        req.Text,
		CreatedAt:   existingPost.CreatedAt,
		Attachments: req.Attachments,
		PhotosPath:  req.Photos,
	}

	if err := h.postStore.UpdatePost(r.Context(), updatedPost); err != nil {
		service.Error(r.Context(), "Failed to update post", err, zap.Uint64("postID", postID))
		sendJSONError(w, "Failed to update post", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Post updated successfully", http.StatusOK)
	service.Info(r.Context(), "Post updated successfully", zap.Uint64("postID", postID))
}

// DeletePost - удаление поста
// @Summary Удалить пост
// @Description Удаляет пост (только автор)
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int true "ID поста"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 401 {object} JSONResponse
// @Failure 403 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /posts/{id} [delete]
func (h *PostsHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		sendJSONError(w, "Invalid post ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	// Получаем userID из контекста
	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	// Бизнес-логика: проверяем существование поста перед удалением
	service.Info(r.Context(), "Deleting post", zap.Uint64("postID", postID), zap.Int("userID", userID))

	existingPost, err := h.postStore.GetPostByID(r.Context(), uint(postID))
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			sendJSONError(w, "Post not found", http.StatusNotFound)
			service.Warn(r.Context(), "Post not found for deletion", zap.Uint64("postID", postID))
		} else {
			service.Error(r.Context(), "Failed to get post for deletion", err, zap.Uint64("postID", postID))
			sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Проверяем, что пользователь является автором поста
	if existingPost.AuthorID != uint(userID) {
		sendJSONError(w, domain.Forbidden, http.StatusForbidden)
		service.Warn(r.Context(), "Access denied: user is not post author",
			zap.Uint64("postID", postID),
			zap.Int("userID", userID),
			zap.Uint("authorID", existingPost.AuthorID))
		return
	}

	// Удаляем пост
	if err := h.postStore.DeletePost(r.Context(), uint(postID), uint(userID)); err != nil {
		service.Error(r.Context(), "Failed to delete post", err, zap.Uint64("postID", postID))
		sendJSONError(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Post deleted successfully", http.StatusOK)
	service.Info(r.Context(), "Post deleted successfully", zap.Uint64("postID", postID))
}

// GetUserPosts - получение постов конкретного пользователя
// @Summary Получить посты пользователя
// @Description Возвращает посты конкретного пользователя с пагинацией
// @Tags posts
// @Accept json
// @Produce json
// @Param userID path int true "ID пользователя"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество постов на странице" default(20)
// @Success 200 {object} PostsResponse
// @Failure 400 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /users/{userID}/posts [get]
func (h *PostsHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["userID"], 10, 32)
	if err != nil {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid user ID", zap.String("userID", vars["userID"]))
		return
	}

	var req PostsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONError(w, domain.InvalidParams, http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	// Бизнес-логика: валидация параметров пагинации
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	service.Info(r.Context(), "Getting user posts",
		zap.Uint64("userID", userID),
		zap.Int("page", req.Page),
		zap.Int("limit", req.Limit))

	// Бизнес-логика: проверяем существование пользователя
	_, err = h.userStore.GetUserByID(int(userID))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			sendJSONError(w, "User not found", http.StatusNotFound)
			service.Warn(r.Context(), "User not found", zap.Uint64("userID", userID))
		} else {
			service.Error(r.Context(), "Failed to get user", err, zap.Uint64("userID", userID))
			sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Получаем посты пользователя
	posts, totalPages, err := h.postStore.GetPostsByUser(r.Context(), uint(userID), req.Page, req.Limit)
	if err != nil {
		service.Error(r.Context(), "Failed to get user posts", err, zap.Uint64("userID", userID))
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := PostsResponse{
		Posts:      posts,
		Page:       req.Page,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", "PostsResponse"))
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	service.Info(r.Context(), "User posts retrieved successfully",
		zap.Uint64("userID", userID),
		zap.Int("postsCount", len(posts)),
		zap.Int("totalPages", totalPages))
}
