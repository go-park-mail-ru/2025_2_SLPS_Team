package handler

import (
	"encoding/json"
	"net/http"
	"project/domain"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type CommentHandler struct {
	commentService domain.CommentService
}

func NewCommentHandler(commentService domain.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// CreateComment создает новый комментарий
// @Summary Создать комментарий
// @Description Создает новый комментарий к посту
// @Tags comments
// @Accept json
// @Produce json
// @Param request body domain.CommentCreateRequest true "Данные для создания комментария"
// @Success 201 {object} domain.CommentView "Комментарий успешно создан"
// @Failure 400 {object} JSONResponse "Неверные данные запроса"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 404 {object} JSONResponse "Пост не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /comments [post]
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var req domain.CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid JSON in request", zap.Error(err))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	comment, err := h.commentService.CreateComment(r.Context(), userID, req)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	// Используем sendJSONData для возврата структуры
	response := map[string]interface{}{
		"message": "Comment created successfully",
		"comment": comment,
	}

	if err := sendJSONData(r.Context(), w, response); err != nil {
		return
	}
}

// GetComment возвращает комментарий по ID
// @Summary Получить комментарий по ID
// @Description Возвращает информацию о комментарии по его идентификатору
// @Tags comments
// @Accept json
// @Produce json
// @Param id path int32 true "ID комментария" minimum(1)
// @Success 200 {object} domain.CommentView "Комментарий найден"
// @Failure 400 {object} JSONResponse "Неверный ID комментария"
// @Failure 404 {object} JSONResponse "Комментарий не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /comments/{id} [get]
func (h *CommentHandler) GetComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	commentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid comment ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid comment ID", zap.String("commentID", vars["id"]))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	comment, err := h.commentService.GetComment(r.Context(), userID, int32(commentID))
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, comment); err != nil {
		return
	}
}

// GetPostComments возвращает комментарии поста
// @Summary Получить комментарии поста
// @Description Возвращает список комментариев к указанному посту с пагинацией
// @Tags comments
// @Accept json
// @Produce json
// @Param postID path int32 true "ID поста" minimum(1)
// @Param page query int32 false "Номер страницы" default(1) minimum(1)
// @Param limit query int32 false "Количество комментариев на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.CommentView "Успешный ответ с комментариями"
// @Failure 400 {object} JSONResponse "Неверные параметры запроса"
// @Failure 404 {object} JSONResponse "Пост не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /posts/{postID}/comments [get]
func (h *CommentHandler) GetPostComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.Atoi(vars["postID"])
	if err != nil {
		sendJSONResponse(w, "Invalid post ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["postID"]))
		return
	}

	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	comments, err := h.commentService.GetPostComments(r.Context(), userID, int32(postID), qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, comments); err != nil {
		return
	}
}

// UpdateComment обновляет комментарий
// @Summary Обновить комментарий
// @Description Обновляет существующий комментарий (только автор)
// @Tags comments
// @Accept json
// @Produce json
// @Param id path int32 true "ID комментария" minimum(1)
// @Param request body domain.CommentUpdateRequest true "Новые данные комментария"
// @Success 200 {object} JSONResponse "Комментарий успешно обновлен"
// @Failure 400 {object} JSONResponse "Неверные данные запроса"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 403 {object} JSONResponse "Доступ запрещен (не автор комментария)"
// @Failure 404 {object} JSONResponse "Комментарий не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /comments/{id} [put]
func (h *CommentHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	commentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid comment ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid comment ID", zap.String("commentID", vars["id"]))
		return
	}

	var req domain.CommentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid JSON in request", zap.Error(err))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.commentService.UpdateComment(r.Context(), int32(commentID), userID, req.Text)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Comment updated successfully", http.StatusOK)
}

// DeleteComment удаляет комментарий
// @Summary Удалить комментарий
// @Description Удаляет комментарий (только автор)
// @Tags comments
// @Accept json
// @Produce json
// @Param id path int32 true "ID комментария" minimum(1)
// @Success 200 {object} JSONResponse "Комментарий успешно удален"
// @Failure 400 {object} JSONResponse "Неверный ID комментария"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 403 {object} JSONResponse "Доступ запрещен (не автор комментария)"
// @Failure 404 {object} JSONResponse "Комментарий не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /comments/{id} [delete]
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	commentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid comment ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid comment ID", zap.String("commentID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.commentService.DeleteComment(r.Context(), int32(commentID), userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Comment deleted successfully", http.StatusOK)
}

// GetPostCommentsCount возвращает количество комментариев поста
// @Summary Получить количество комментариев поста
// @Description Возвращает общее количество комментариев к указанному посту
// @Tags comments
// @Accept json
// @Produce json
// @Param postID path int32 true "ID поста" minimum(1)
// @Success 200 {object} map[string]int32 "Количество комментариев"
// @Failure 400 {object} JSONResponse "Неверный ID поста"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /posts/{postID}/comments/count [get]
func (h *CommentHandler) GetPostCommentsCount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.Atoi(vars["postID"])
	if err != nil {
		sendJSONResponse(w, "Invalid post ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["postID"]))
		return
	}

	count, err := h.commentService.GetPostCommentsCount(r.Context(), int32(postID))
	if err != nil {
		sendJSONError(w, err)
		return
	}

	response := map[string]int32{
		"count": count,
	}

	if err := sendJSONData(r.Context(), w, response); err != nil {
		return
	}
}
