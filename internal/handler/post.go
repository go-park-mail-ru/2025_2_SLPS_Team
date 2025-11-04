package handler

import (
	"mime/multipart"
	"net/http"
	"project/domain"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type PostsHandler struct {
	postService domain.PostService
}

func NewPostsHandler(postService domain.PostService) *PostsHandler {
	return &PostsHandler{
		postService: postService,
	}
}

// PostsRequest - запрос для пагинации постов
// @Description Параметры пагинации для постов
type PostsRequest struct {
	Page  int `schema:"page" example:"1"`   // Номер страницы
	Limit int `schema:"limit" example:"20"` // Количество постов на странице
}

// PostsResponse - ответ с постами и пагинацией
// @Description Ответ с пагинированным списком постов
type PostsResponse struct {
	Posts []domain.Post `json:"posts"`            // Список постов
}

// PostsPaginate возвращает посты с пагинацией
// @Summary Получить посты с пагинацией
// @Description Возвращает список постов с поддержкой пагинации
// @Tags posts
// @Accept json
// @Produce json
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество постов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {object} PostsResponse "Успешный ответ с постами"
// @Failure 400 {object} JSONResponse "Неверные параметры запроса"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /posts [get]
func (h *PostsHandler) PostsPaginate(w http.ResponseWriter, r *http.Request) {
	var req PostsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	posts, err := h.postService.PostsPaginate(r.Context(), req.Page, req.Limit)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	response := PostsResponse{
		Posts: posts,

	}

	if err := sendJSONData(r.Context(), w, response); err != nil {
		return
	}
}

// GetPost возвращает пост по ID
// @Summary Получить пост по ID
// @Description Возвращает пост по его идентификатору
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int true "ID поста" minimum(1)
// @Success 200 {object} domain.Post "Пост найден"
// @Failure 400 {object} JSONResponse "Неверный ID поста"
// @Failure 404 {object} JSONResponse "Пост не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /posts/{id} [get]
func (h *PostsHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		sendJSONResponse(w, "Invalid post ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	post, err := h.postService.GetPost(r.Context(), uint(postID))
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, post); err != nil {
		return
	}
}

// CreatePost - создание нового поста с файлами
// @Summary Создать новый пост
// @Description Создает новый пост от имени текущего пользователя с возможностью загрузки файлов
// @Tags posts
// @Accept multipart/form-data
// @Produce json
// @Param text formData string true "Текст поста"
// @Param attachments formData []file false "Вложения" collectionFormat(multi)
// @Param photos formData []file false "Фотографии" collectionFormat(multi)
// @Success 201 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 401 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /posts [post]
func (h *PostsHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		sendJSONResponse(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	text := r.FormValue("text")
	if text == "" {
		sendJSONResponse(w, "Text is required", http.StatusBadRequest)
		domain.Warn(r.Context(), "Text is required for post creation")
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	var attachmentFiles, photoFiles []*multipart.FileHeader
	if attachments, ok := r.MultipartForm.File["attachments"]; ok {
		attachmentFiles = attachments
	}
	if photos, ok := r.MultipartForm.File["photos"]; ok {
		photoFiles = photos
	}

	post, err := h.postService.CreatePost(r.Context(), userID, text, attachmentFiles, photoFiles)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Post created successfully", http.StatusCreated)
	domain.Info(r.Context(), "Post created successfully", zap.Uint("postID", post.ID))
}

// UpdatePost - обновление поста с файлами
// @Summary Обновить пост
// @Description Обновляет существующий пост (только автор) с возможностью замены файлов
// @Tags posts
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID поста" minimum(1)
// @Param text formData string true "Текст поста"
// @Param attachments formData []file false "Новые вложения" collectionFormat(multi)
// @Param photos formData []file false "Новые фотографии" collectionFormat(multi)
// @Success 200 {object} JSONResponse "Пост успешно обновлен"
// @Failure 400 {object} JSONResponse "Неверные данные запроса"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 403 {object} JSONResponse "Доступ запрещен (не автор поста)"
// @Failure 404 {object} JSONResponse "Пост не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /posts/{id} [put]
func (h *PostsHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		sendJSONResponse(w, "Invalid post ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	err = r.ParseMultipartForm(50 << 20)
	if err != nil {
		sendJSONResponse(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	text := r.FormValue("text")
	if text == "" {
		sendJSONResponse(w, "Text is required", http.StatusBadRequest)
		domain.Warn(r.Context(), "Text is required for post update")
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, "Unauthorized", http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	var attachmentFiles, photoFiles []*multipart.FileHeader
	if attachments, ok := r.MultipartForm.File["attachments"]; ok {
		attachmentFiles = attachments
	}
	if photos, ok := r.MultipartForm.File["photos"]; ok {
		photoFiles = photos
	}

	err = h.postService.UpdatePost(r.Context(), uint(postID), userID, text, attachmentFiles, photoFiles)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Post updated successfully", http.StatusOK)
}

// DeletePost удаляет пост
// @Summary Удалить пост
// @Description Удаляет пост (только автор)
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int true "ID поста" minimum(1)
// @Success 200 {object} JSONResponse "Пост успешно удален"
// @Failure 400 {object} JSONResponse "Неверный ID поста"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 403 {object} JSONResponse "Доступ запрещен (не автор поста)"
// @Failure 404 {object} JSONResponse "Пост не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /posts/{id} [delete]
func (h *PostsHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		sendJSONResponse(w, "Invalid post ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, "Unauthorized", http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.postService.DeletePost(r.Context(), uint(postID), userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Post deleted successfully", http.StatusOK)
}

// GetUserPosts возвращает посты пользователя
// @Summary Получить посты пользователя
// @Description Возвращает посты конкретного пользователя с пагинацией
// @Tags posts
// @Accept json
// @Produce json
// @Param userID path int true "ID пользователя" minimum(1)
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество постов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {object} PostsResponse "Успешный ответ с постами пользователя"
// @Failure 400 {object} JSONResponse "Неверные параметры запроса"
// @Failure 404 {object} JSONResponse "Пользователь не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /users/{userID}/posts [get]
func (h *PostsHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.ParseUint(vars["userID"], 10, 32)
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("userID", vars["userID"]))
		return
	}

	var req PostsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	posts, err := h.postService.GetUserPosts(r.Context(), uint(userID), req.Page, req.Limit)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	response := PostsResponse{
		Posts:      posts,
	}

	if err := sendJSONData(r.Context(), w, response); err != nil {
		return
	}
}
