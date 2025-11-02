package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"project/domain"
	"project/internal/service"
	"strconv"

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
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
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
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
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
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
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
		sendJSONResponse(w, "Invalid post ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	service.Info(r.Context(), "Getting post by ID", zap.Uint64("postID", postID))

	// Бизнес-логика: получение поста
	post, err := h.postStore.GetPostByID(r.Context(), uint(postID))
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			sendJSONResponse(w, "Post not found", http.StatusNotFound)
			service.Warn(r.Context(), "Post not found", zap.Uint64("postID", postID))
		} else {
			service.Error(r.Context(), "Failed to get post", err, zap.Uint64("postID", postID))
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(post); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", "Post"))
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
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
// @Param attachments formData file false "Вложения"
// @Param photos formData file false "Фотографии"
// @Success 201 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 401 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /posts [post]
func (h *PostsHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	// Парсим multipart форму
	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		sendJSONResponse(w, "Can't parse multipart form", http.StatusBadRequest)
		service.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	// Получаем текст поста
	text := r.FormValue("text")
	if text == "" {
		sendJSONResponse(w, "Text is required", http.StatusBadRequest)
		service.Warn(r.Context(), "Text is required for post creation")
		return
	}

	// Бизнес-логика: валидация данных
	if len(text) < 24 || len(text) > 4096 {
		sendJSONResponse(w, "Text length must be between 24 and 4096 characters", http.StatusBadRequest)
		service.Warn(r.Context(), "Post text validation failed", zap.Int("textLength", len(text)))
		return
	}

	// Получаем userID из контекста
	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Creating new post", zap.Int("userID", userID))

	// Обрабатываем вложения
	var attachmentPaths []string
	if attachmentFiles, ok := r.MultipartForm.File["attachments"]; ok && len(attachmentFiles) > 0 {
		attachmentPaths, err = service.UploadFiles(attachmentFiles)
		if err != nil {
			service.Error(r.Context(), "Failed to upload attachments", err)
			sendJSONResponse(w, "Failed to upload attachments", http.StatusInternalServerError)
			return
		}
	}

	// Обрабатываем фотографии
	var photoPaths []string
	if photoFiles, ok := r.MultipartForm.File["photos"]; ok && len(photoFiles) > 0 {
		photoPaths, err = service.UploadFiles(photoFiles)
		if err != nil {
			// Если загрузка фото не удалась, удаляем уже загруженные вложения
			if len(attachmentPaths) > 0 {
				service.DeleteFiles(convertToPointerSlice(attachmentPaths))
			}
			service.Error(r.Context(), "Failed to upload photos", err)
			sendJSONResponse(w, "Failed to upload photos", http.StatusInternalServerError)
			return
		}
	}

	// Бизнес-логика: создание объекта поста
	post := &domain.Post{
		AuthorID:    uint(userID),
		Text:        text,
		Attachments: attachmentPaths,
		PhotosPath:  photoPaths,
	}

	// Сохраняем пост в хранилище
	if err := h.postStore.CreatePost(r.Context(), post); err != nil {
		// Если сохранение в БД не удалось, удаляем загруженные файлы
		if len(attachmentPaths) > 0 {
			service.DeleteFiles(convertToPointerSlice(attachmentPaths))
		}
		if len(photoPaths) > 0 {
			service.DeleteFiles(convertToPointerSlice(photoPaths))
		}
		service.Error(r.Context(), "Failed to create post", err, zap.Int("userID", userID))
		sendJSONResponse(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "Post created successfully", http.StatusCreated)
	service.Info(r.Context(), "Post created successfully",
		zap.Uint("postID", post.ID),
		zap.Int("userID", userID),
		zap.Int("attachmentsCount", len(attachmentPaths)),
		zap.Int("photosCount", len(photoPaths)))
}

// UpdatePost - обновление поста с файлами
// @Summary Обновить пост
// @Description Обновляет существующий пост (только автор) с возможностью замены файлов
// @Tags posts
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID поста"
// @Param text formData string true "Текст поста"
// @Param attachments formData file false "Новые вложения"
// @Param photos formData file false "Новые фотографии"
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
		sendJSONResponse(w, "Invalid post ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	// Парсим multipart форму
	err = r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		sendJSONResponse(w, "Can't parse multipart form", http.StatusBadRequest)
		service.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	// Получаем текст поста
	text := r.FormValue("text")
	if text == "" {
		sendJSONResponse(w, "Text is required", http.StatusBadRequest)
		service.Warn(r.Context(), "Text is required for post update")
		return
	}

	// Бизнес-логика: валидация данных
	if len(text) < 24 || len(text) > 4096 {
		sendJSONResponse(w, "Text length must be between 24 and 4096 characters", http.StatusBadRequest)
		service.Warn(r.Context(), "Post text validation failed", zap.Int("textLength", len(text)))
		return
	}

	// Получаем userID из контекста
	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, "Unauthorized", http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	// Бизнес-логика: проверяем существование поста и права доступа
	service.Info(r.Context(), "Updating post", zap.Uint64("postID", postID), zap.Int("userID", userID))

	existingPost, err := h.postStore.GetPostByID(r.Context(), uint(postID))
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			sendJSONResponse(w, "Post not found", http.StatusNotFound)
			service.Warn(r.Context(), "Post not found for update", zap.Uint64("postID", postID))
		} else {
			service.Error(r.Context(), "Failed to get post for update", err, zap.Uint64("postID", postID))
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Проверяем, что пользователь является автором поста
	if existingPost.AuthorID != uint(userID) {
		sendJSONResponse(w, domain.Forbidden, http.StatusForbidden)
		service.Warn(r.Context(), "Access denied: user is not post author",
			zap.Uint64("postID", postID),
			zap.Int("userID", userID),
			zap.Uint("authorID", existingPost.AuthorID))
		return
	}

	// Подготавливаем старые пути для удаления
	var oldAttachments []*string
	var oldPhotos []*string

	// Обрабатываем новые вложения
	var newAttachmentPaths []string
	if attachmentFiles, ok := r.MultipartForm.File["attachments"]; ok && len(attachmentFiles) > 0 {
		// Сохраняем старые пути для последующего удаления
		for i := range existingPost.Attachments {
			oldAttachments = append(oldAttachments, &existingPost.Attachments[i])
		}

		newAttachmentPaths, err = service.UploadFiles(attachmentFiles)
		if err != nil {
			service.Error(r.Context(), "Failed to upload new attachments", err)
			sendJSONResponse(w, "Failed to upload attachments", http.StatusInternalServerError)
			return
		}
	} else {
		// Если новые вложения не загружены, оставляем старые
		newAttachmentPaths = existingPost.Attachments
	}

	// Обрабатываем новые фотографии
	var newPhotoPaths []string
	if photoFiles, ok := r.MultipartForm.File["photos"]; ok && len(photoFiles) > 0 {
		// Сохраняем старые пути для последующего удаления
		for i := range existingPost.PhotosPath {
			oldPhotos = append(oldPhotos, &existingPost.PhotosPath[i])
		}

		newPhotoPaths, err = service.UploadFiles(photoFiles)
		if err != nil {
			// Если загрузка новых фото не удалась, удаляем уже загруженные новые вложения
			if len(newAttachmentPaths) > len(existingPost.Attachments) {
				newFiles := newAttachmentPaths[len(existingPost.Attachments):]
				service.DeleteFiles(convertToPointerSlice(newFiles))
			}
			service.Error(r.Context(), "Failed to upload new photos", err)
			sendJSONResponse(w, "Failed to upload photos", http.StatusInternalServerError)
			return
		}
	} else {
		// Если новые фото не загружены, оставляем старые
		newPhotoPaths = existingPost.PhotosPath
	}

	// Обновляем данные поста
	updatedPost := &domain.Post{
		ID:          uint(postID),
		AuthorID:    uint(userID),
		Text:        text,
		CreatedAt:   existingPost.CreatedAt,
		Attachments: newAttachmentPaths,
		PhotosPath:  newPhotoPaths,
	}

	if err := h.postStore.UpdatePost(r.Context(), updatedPost); err != nil {
		// Если обновление в БД не удалось, удаляем загруженные новые файлы
		if len(newAttachmentPaths) > len(existingPost.Attachments) {
			newFiles := newAttachmentPaths[len(existingPost.Attachments):]
			service.DeleteFiles(convertToPointerSlice(newFiles))
		}
		if len(newPhotoPaths) > len(existingPost.PhotosPath) {
			newFiles := newPhotoPaths[len(existingPost.PhotosPath):]
			service.DeleteFiles(convertToPointerSlice(newFiles))
		}
		service.Error(r.Context(), "Failed to update post", err, zap.Uint64("postID", postID))
		sendJSONResponse(w, "Failed to update post", http.StatusInternalServerError)
		return
	}

	// Удаляем старые файлы после успешного обновления в БД
	if len(oldAttachments) > 0 {
		if err := service.DeleteFiles(oldAttachments); err != nil {
			service.Error(r.Context(), "Failed to delete old attachments", err)
			// Не прерываем выполнение, так как пост уже обновлен
		}
	}
	if len(oldPhotos) > 0 {
		if err := service.DeleteFiles(oldPhotos); err != nil {
			service.Error(r.Context(), "Failed to delete old photos", err)
			// Не прерываем выполнение, так как пост уже обновлен
		}
	}

	sendJSONResponse(w, "Post updated successfully", http.StatusOK)
	service.Info(r.Context(), "Post updated successfully", zap.Uint64("postID", postID))
}

// Вспомогательная функция для конвертации []string в []*string
func convertToPointerSlice(slice []string) []*string {
	result := make([]*string, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
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
		sendJSONResponse(w, "Invalid post ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid post ID", zap.String("postID", vars["id"]))
		return
	}

	// Получаем userID из контекста
	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, "Unauthorized", http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	// Бизнес-логика: проверяем существование поста перед удалением
	service.Info(r.Context(), "Deleting post", zap.Uint64("postID", postID), zap.Int("userID", userID))

	existingPost, err := h.postStore.GetPostByID(r.Context(), uint(postID))
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			sendJSONResponse(w, "Post not found", http.StatusNotFound)
			service.Warn(r.Context(), "Post not found for deletion", zap.Uint64("postID", postID))
		} else {
			service.Error(r.Context(), "Failed to get post for deletion", err, zap.Uint64("postID", postID))
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Проверяем, что пользователь является автором поста
	if existingPost.AuthorID != uint(userID) {
		sendJSONResponse(w, domain.Forbidden, http.StatusForbidden)
		service.Warn(r.Context(), "Access denied: user is not post author",
			zap.Uint64("postID", postID),
			zap.Int("userID", userID),
			zap.Uint("authorID", existingPost.AuthorID))
		return
	}

	// Подготавливаем пути файлов для удаления
	var filesToDelete []*string

	// Добавляем вложения для удаления
	for i := range existingPost.Attachments {
		filesToDelete = append(filesToDelete, &existingPost.Attachments[i])
	}

	// Добавляем фотографии для удаления
	for i := range existingPost.PhotosPath {
		filesToDelete = append(filesToDelete, &existingPost.PhotosPath[i])
	}

	// Удаляем пост из базы данных
	if err := h.postStore.DeletePost(r.Context(), uint(postID), uint(userID)); err != nil {
		service.Error(r.Context(), "Failed to delete post", err, zap.Uint64("postID", postID))
		sendJSONResponse(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}

	// Удаляем файлы после успешного удаления поста из БД
	if len(filesToDelete) > 0 {
		if err := service.DeleteFiles(filesToDelete); err != nil {
			service.Error(r.Context(), "Failed to delete post files", err)
			// Не прерываем выполнение, так как пост уже удален из БД
		}
	}

	sendJSONResponse(w, "Post deleted successfully", http.StatusOK)
	service.Info(r.Context(), "Post deleted successfully",
		zap.Uint64("postID", postID),
		zap.Int("deletedFiles", len(filesToDelete)))
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
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid user ID", zap.String("userID", vars["userID"]))
		return
	}

	var req PostsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
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
	_, err = h.userStore.GetUserByID(r.Context(), int(userID))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			sendJSONResponse(w, "User not found", http.StatusNotFound)
			service.Warn(r.Context(), "User not found", zap.Uint64("userID", userID))
		} else {
			service.Error(r.Context(), "Failed to get user", err, zap.Uint64("userID", userID))
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Получаем посты пользователя
	posts, totalPages, err := h.postStore.GetPostsByUser(r.Context(), uint(userID), req.Page, req.Limit)
	if err != nil {
		service.Error(r.Context(), "Failed to get user posts", err, zap.Uint64("userID", userID))
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
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
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	service.Info(r.Context(), "User posts retrieved successfully",
		zap.Uint64("userID", userID),
		zap.Int("postsCount", len(posts)),
		zap.Int("totalPages", totalPages))
}
