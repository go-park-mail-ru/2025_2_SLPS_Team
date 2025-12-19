package handler

import (
	"net/http"
	"project/domain"
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
	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	posts, err := h.postService.PostsPaginate(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, domain.PostViewList(posts))
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
	postID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	post, err := h.postService.GetPost(r.Context(), userID, uint(postID))
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, post)
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
	err := ParseMultipart(r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	text := r.FormValue("text")
	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	attachmentFiles, err := domain.MultipartFiles(r, "attachments")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	photoFiles, err := domain.MultipartFiles(r, "photos")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	communityID, err := parseIntParam(r, "communityID")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	post, err := h.postService.CreatePost(r.Context(), userID, text, communityID, attachmentFiles, photoFiles)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	response := domain.PostCreateResponse{Message: "Post created successfully", Post: post}
	sendJSONData(r.Context(), w, response)
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

	postID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	err = ParseMultipart(r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	text := r.FormValue("text")
	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	attachmentFiles, err := domain.MultipartFiles(r, "attachments")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	photoFiles, err := domain.MultipartFiles(r, "photos")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	err = h.postService.UpdatePost(r.Context(), uint(postID), userID, text, attachmentFiles, photoFiles)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "Post updated successfully")
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
	postID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	err = h.postService.DeletePost(r.Context(), uint(postID), userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "Post deleted successfully")
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
	userID, err := PathInt32(r, "userID")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int32)
	posts, err := h.postService.GetUserPosts(r.Context(), selfUserID, uint(userID), qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, domain.PostViewList(posts))
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
	postID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	err = h.postService.UpdateLikeOnPostByUserID(r.Context(), userID, postID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "like updated")
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
	communityID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)
	posts, err := h.postService.GetCommunityPosts(r.Context(), userID, communityID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, domain.PostViewList(posts))
}
