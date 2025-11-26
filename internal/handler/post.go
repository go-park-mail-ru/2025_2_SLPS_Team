package handler

import (
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
	Page  int32 `schema:"page" example:"1"`   // Номер страницы
	Limit int32 `schema:"limit" example:"20"` // Количество постов на странице
}

// PostsResponse - ответ с постами и пагинацией
// @Description Ответ с пагинированным списком постов
type PostsResponse struct {
	Posts []domain.Post `json:"posts"` // Список постов
}

// PostsPaginate возвращает посты с пагинацией
// @Summary Получить посты с пагинацией
// @Description Возвращает список постов с поддержкой пагинации (включая посты из сообществ)
// @Tags posts
// @Accept json
// @Produce json
// @Param page query int32 false "Номер страницы" default(1) minimum(1)
// @Param limit query int32 false "Количество постов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.PostView "Успешный ответ с постами"
// @Failure 400 {object} JSONResponse "Неверные параметры запроса"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /posts [get]
func (h *PostsHandler) PostsPaginate(w http.ResponseWriter, r *http.Request) {
	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	posts, err := h.postService.PostsPaginate(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, posts); err != nil {
		return
	}
}

// GetPost возвращает пост по ID
// @Summary Получить пост по ID
// @Description Возвращает пост по его идентификатору
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int32 true "ID поста" minimum(1)
// @Success 200 {object} domain.PostView "Пост найден"
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
	userID, _ := r.Context().Value(domain.UserIDKey).(int32)
	post, err := h.postService.GetPost(r.Context(), userID, uint(postID))
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
// @Param text formData string false "Текст поста"
// @Param communityID formData int32 false "ID сообщества (если пост в сообществе)"
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
	communityIDStr := r.FormValue("communityID")

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	var attachmentFiles, photoFiles []*domain.File
	if attachments, ok := r.MultipartForm.File["attachments"]; ok {
		attachmentFiles, err = domain.MultipartListToFiles(attachments)
		if err != nil {
			sendJSONResponse(w, "Can't parse multipart form to files", http.StatusBadRequest)
			domain.Error(r.Context(), "Failed to parse multipart form to files", err)
			return
		}
	}
	if photos, ok := r.MultipartForm.File["photos"]; ok {
		photoFiles, err = domain.MultipartListToFiles(photos)
		if err != nil {
			sendJSONResponse(w, "Can't parse multipart form to files", http.StatusBadRequest)
			domain.Error(r.Context(), "Failed to parse multipart form to files", err)
			return
		}
	}

	var communityID *int32
	if communityIDStr != "" {
		id, err := strconv.Atoi(communityIDStr)
		if err != nil {
			sendJSONResponse(w, "Invalid community ID", http.StatusBadRequest)
			domain.Warn(r.Context(), "Invalid community ID", zap.String("communityID", communityIDStr))
			return
		}
		*communityID = int32(id)
	}

	post, err := h.postService.CreatePost(r.Context(), userID, text, communityID, attachmentFiles, photoFiles)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	response := map[string]interface{}{
		"message": "Post created successfully",
		"post":    post,
	}

	if err := sendJSONData(r.Context(), w, response); err != nil {
		return
	}
}

// UpdatePost - обновление поста с файлами
// @Summary Обновить пост
// @Description Обновляет существующий пост (только автор) с возможностью замены файлов
// @Tags posts
// @Accept multipart/form-data
// @Produce json
// @Param id path int32 true "ID поста" minimum(1)
// @Param text formData string false "Текст поста"
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, "Unauthorized", http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	var attachmentFiles, photoFiles []*domain.File
	if attachments, ok := r.MultipartForm.File["attachments"]; ok {
		attachmentFiles, err = domain.MultipartListToFiles(attachments)
		if err != nil {
			sendJSONResponse(w, "Can't parse multipart form to files", http.StatusBadRequest)
			domain.Error(r.Context(), "Failed to parse multipart form to files", err)
			return
		}
	}
	if photos, ok := r.MultipartForm.File["photos"]; ok {
		photoFiles, err = domain.MultipartListToFiles(photos)
		if err != nil {
			sendJSONResponse(w, "Can't parse multipart form to files", http.StatusBadRequest)
			domain.Error(r.Context(), "Failed to parse multipart form to files", err)
			return
		}
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
// @Param id path int32 true "ID поста" minimum(1)
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
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
// @Param userID path int32 true "ID пользователя" minimum(1)
// @Param page query int32 false "Номер страницы" default(1) minimum(1)
// @Param limit query int32 false "Количество постов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.PostView "Успешный ответ с постами пользователя"
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

	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}
	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int32)
	posts, err := h.postService.GetUserPosts(r.Context(), selfUserID, uint(userID), qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, posts); err != nil {
		return
	}
}

// UpdateLikeOnPost ставит или убирает лайк пользователя на посте.
//
// @Summary Поставить или убрать лайк на посте
// @Description Переключает лайк текущего (аутентифицированного) пользователя на указанном посте.
// Если лайк уже есть → убирается, если нет → ставится.
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int32 true "ID поста"
// @Success 200 {object} JSONResponse "Информация о результате операции: лайк поставлен или снят"
// @Failure 400 {object} JSONResponse "Некорректный ID поста"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /posts/{id}/like [put]
func (h *PostsHandler) UpdateLikeOnPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postIDStr := vars["id"]
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		sendJSONResponse(w, "invalid postID", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse postID", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	err = h.postService.UpdateLikeOnPostByUserID(r.Context(), userID, int32(postID))
	if err != nil {
		domain.FromContext(r.Context()).Error("Failed update like on post", zap.Error(err))
		return
	}

	domain.FromContext(r.Context()).Info("like updated")
	sendJSONResponse(w, "like updated", http.StatusOK)
}

// GetCommunityPosts возвращает посты сообщества
// @Summary Получить посты сообщества
// @Description Возвращает посты конкретного сообщества с пагинацией
// @Tags posts
// @Accept json
// @Produce json
// @Param id path int32 true "ID сообщества" minimum(1)
// @Param page query int32 false "Номер страницы" default(1) minimum(1)
// @Param limit query int32 false "Количество постов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.PostView "Успешный ответ с постами сообщества"
// @Failure 400 {object} JSONResponse "Неверные параметры запроса"
// @Failure 404 {object} JSONResponse "Сообщество не найдено"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /posts/communities/{id} [get]
func (h *PostsHandler) GetCommunityPosts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid community ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid community ID", zap.String("communityID", vars["id"]))
		return
	}

	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)
	posts, err := h.postService.GetCommunityPosts(r.Context(), userID, int32(communityID), qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, posts); err != nil {
		return
	}
}
