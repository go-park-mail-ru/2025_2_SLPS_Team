package service

import (
	"context"
	"errors"
	"mime/multipart"
	"project/domain"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type PostService struct {
	postStore domain.PostStore
	userStore domain.UserStore
}

func NewPostService(postStore domain.PostStore, userStore domain.UserStore) domain.PostService {
	return &PostService{
		postStore: postStore,
		userStore: userStore,
	}
}

// PostsPaginate возвращает посты с пагинацией
func (s *PostService) PostsPaginate(ctx context.Context, userID int, params domain.PaginateQueryParams) ([]domain.PostWithShortUser, error) {
	offset, limit := domain.ValidatePaginationParams(params)
	domain.Info(ctx, "Getting paginated posts", zap.Int("offset", offset), zap.Int("limit", limit))

	postsWithAuthor, err := s.postStore.PostsPaginatedList(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get posts", err)
		return nil, domain.ErrDB
	}

	return postsWithAuthor, nil
}

// GetPost возвращает пост по ID
func (s *PostService) GetPost(ctx context.Context, userID int, postID uint) (*domain.Post, error) {
	domain.Info(ctx, "Getting post by ID", zap.Uint("postID", postID))

	post, err := s.postStore.GetPostByID(ctx, userID, postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			domain.Warn(ctx, "Post not found", zap.Uint("postID", postID))
			return nil, domain.ErrPostNotFound
		}
		domain.Error(ctx, "Failed to get post", err, zap.Uint("postID", postID))
		return nil, domain.ErrDB
	}

	return post, nil
}

// CreatePost создает новый пост
func (s *PostService) CreatePost(ctx context.Context, userID int, text string, attachmentFiles []*multipart.FileHeader, photoFiles []*multipart.FileHeader) (*domain.Post, error) {

	// Создаем структуру для валидации
	createRequest := domain.PostCreateRequest{
		Text: text,
	}

	// Валидация структуры
	ok, err := govalidator.ValidateStruct(createRequest)
	if !ok || err != nil {
		domain.Warn(ctx, "Post validation failed", zap.Error(err))
		return nil, domain.ErrInvalidInput
	}

	domain.Info(ctx, "Creating new post", zap.Int("userID", userID))

	// Обработка вложений
	var attachmentPaths []string
	if len(attachmentFiles) > 0 {
		attachmentPaths, err = UploadFiles(attachmentFiles)
		if err != nil {
			domain.Error(ctx, "Failed to upload attachments", err)
			return nil, domain.ErrService
		}
		createRequest.Attachments = attachmentPaths
	}

	// Обработка фото
	var photoPaths []string
	if len(photoFiles) > 0 {
		photoPaths, err = UploadFiles(photoFiles)
		if err != nil {
			if len(attachmentPaths) > 0 {
				DeleteFiles(convertToPointerSlice(attachmentPaths))
			}
			domain.Error(ctx, "Failed to upload photos", err)
			return nil, domain.ErrService
		}
		createRequest.Photos = photoPaths
	}

	// Создаем объект поста
	post := &domain.Post{
		AuthorID:    uint(userID),
		Text:        createRequest.Text,
		Attachments: createRequest.Attachments,
		PhotosPath:  createRequest.Photos,
	}

	// Сохраняем в БД
	if err := s.postStore.CreatePost(ctx, post); err != nil {
		if len(attachmentPaths) > 0 {
			DeleteFiles(convertToPointerSlice(attachmentPaths))
		}
		if len(photoPaths) > 0 {
			DeleteFiles(convertToPointerSlice(photoPaths))
		}
		domain.Error(ctx, "Failed to create post", err, zap.Int("userID", userID))
		return nil, domain.ErrDB
	}

	domain.Info(ctx, "Post created successfully",
		zap.Uint("postID", post.ID),
		zap.Int("userID", userID),
		zap.Int("attachmentsCount", len(attachmentPaths)),
		zap.Int("photosCount", len(photoPaths)))

	return post, nil
}

// UpdatePost обновляет пост
func (s *PostService) UpdatePost(ctx context.Context, postID uint, userID int, text string, attachmentFiles []*multipart.FileHeader, photoFiles []*multipart.FileHeader) error {
	// Создаем структуру для валидации
	updateRequest := domain.PostUpdateRequest{
		Text: text,
	}

	// Валидация с структуры
	ok, err := govalidator.ValidateStruct(updateRequest)
	if !ok || err != nil {
		domain.Warn(ctx, "Post validation failed", zap.Error(err))
		return domain.ErrInvalidInput
	}

	domain.Info(ctx, "Updating post", zap.Uint("postID", postID), zap.Int("userID", userID))

	// Получаем текущий пост
	existingPost, err := s.postStore.GetPostByID(ctx, userID, postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			domain.Warn(ctx, "Post not found for update", zap.Uint("postID", postID))
			return domain.ErrPostNotFound
		}
		domain.Error(ctx, "Failed to get post for update", err, zap.Uint("postID", postID))
		return domain.ErrDB
	}

	// Проверяем права доступа
	if existingPost.AuthorID != uint(userID) {
		domain.Warn(ctx, "Access denied: user is not post author",
			zap.Uint("postID", postID),
			zap.Int("userID", userID),
			zap.Uint("authorID", existingPost.AuthorID))
		return domain.ErrAccessDenied
	}

	// Подготавливаем старые пути для удаления
	var oldAttachments []*string
	var oldPhotos []*string

	// Обрабатываем новые вложения
	var newAttachmentPaths []string
	if len(attachmentFiles) > 0 {
		// Сохраняем старые пути
		for i := range existingPost.Attachments {
			oldAttachments = append(oldAttachments, &existingPost.Attachments[i])
		}
		newAttachmentPaths, err = UploadFiles(attachmentFiles)
		if err != nil {
			domain.Error(ctx, "Failed to upload new attachments", err)
			return domain.ErrService
		}
		updateRequest.Attachments = newAttachmentPaths
	} else {
		newAttachmentPaths = existingPost.Attachments
		updateRequest.Attachments = existingPost.Attachments
	}

	// Обрабатываем новые фотографии
	var newPhotoPaths []string
	if len(photoFiles) > 0 {
		// Сохраняем старые пути
		for i := range existingPost.PhotosPath {
			oldPhotos = append(oldPhotos, &existingPost.PhotosPath[i])
		}

		newPhotoPaths, err = UploadFiles(photoFiles)
		if err != nil {
			if len(newAttachmentPaths) > len(existingPost.Attachments) {
				newFiles := newAttachmentPaths[len(existingPost.Attachments):]
				DeleteFiles(convertToPointerSlice(newFiles))
			}
			domain.Error(ctx, "Failed to upload new photos", err)
			return domain.ErrService
		}
		updateRequest.Photos = newPhotoPaths
	} else {
		newPhotoPaths = existingPost.PhotosPath
		updateRequest.Photos = existingPost.PhotosPath
	}

	// Обновляем данные поста
	updatedPost := &domain.Post{
		ID:          postID,
		AuthorID:    uint(userID),
		Text:        updateRequest.Text,
		CreatedAt:   existingPost.CreatedAt,
		Attachments: updateRequest.Attachments,
		PhotosPath:  updateRequest.Photos,
	}

	if err := s.postStore.UpdatePost(ctx, updatedPost); err != nil {
		if len(newAttachmentPaths) > len(existingPost.Attachments) {
			newFiles := newAttachmentPaths[len(existingPost.Attachments):]
			DeleteFiles(convertToPointerSlice(newFiles))
		}
		if len(newPhotoPaths) > len(existingPost.PhotosPath) {
			newFiles := newPhotoPaths[len(existingPost.PhotosPath):]
			DeleteFiles(convertToPointerSlice(newFiles))
		}
		domain.Error(ctx, "Failed to update post", err, zap.Uint("postID", postID))
		return domain.ErrDB
	}

	// Удаляем старые файлы
	if len(oldAttachments) > 0 {
		if err := DeleteFiles(oldAttachments); err != nil {
			domain.Error(ctx, "Failed to delete old attachments", err)
		}
	}
	if len(oldPhotos) > 0 {
		if err := DeleteFiles(oldPhotos); err != nil {
			domain.Error(ctx, "Failed to delete old photos", err)
		}
	}

	domain.Info(ctx, "Post updated successfully", zap.Uint("postID", postID))
	return nil
}

// DeletePost удаляет пост
func (s *PostService) DeletePost(ctx context.Context, postID uint, userID int) error {
	domain.Info(ctx, "Deleting post", zap.Uint("postID", postID), zap.Int("userID", userID))

	// Получаем пост для проверки прав и получения путей файлов
	existingPost, err := s.postStore.GetPostByID(ctx, userID, postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			domain.Warn(ctx, "Post not found for deletion", zap.Uint("postID", postID))
			return domain.ErrPostNotFound
		}
		domain.Error(ctx, "Failed to get post for deletion", err, zap.Uint("postID", postID))
		return domain.ErrDB
	}

	// Проверяем права доступа
	if existingPost.AuthorID != uint(userID) {
		domain.Warn(ctx, "Access denied: user is not post author",
			zap.Uint("postID", postID),
			zap.Int("userID", userID),
			zap.Uint("authorID", existingPost.AuthorID))
		return domain.ErrAccessDenied
	}

	// Удаляем пост из БД
	if err := s.postStore.DeletePost(ctx, postID, uint(userID)); err != nil {
		domain.Error(ctx, "Failed to delete post", err, zap.Uint("postID", postID))
		return domain.ErrDB
	}

	// Подготавливаем пути файлов для удаления
	var filesToDelete []*string
	for i := range existingPost.Attachments {
		filesToDelete = append(filesToDelete, &existingPost.Attachments[i])
	}
	for i := range existingPost.PhotosPath {
		filesToDelete = append(filesToDelete, &existingPost.PhotosPath[i])
	}

	// Удаляем файлы
	if len(filesToDelete) > 0 {
		if err := DeleteFiles(filesToDelete); err != nil {
			domain.Error(ctx, "Failed to delete post files", err)
			// Не прерываем выполнение
		}
	}

	domain.Info(ctx, "Post deleted successfully",
		zap.Uint("postID", postID),
		zap.Int("deletedFiles", len(filesToDelete)))
	return nil
}

// GetUserPosts возвращает посты пользователя
func (s *PostService) GetUserPosts(ctx context.Context, selfUserID int, userID uint, params domain.PaginateQueryParams) ([]domain.Post, error) {
	// Валидация параметров
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting user posts",
		zap.Uint("userID", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit))

	// Проверяем существование пользователя
	_, err := s.userStore.GetUserByID(ctx, int(userID))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			domain.Warn(ctx, "User not found", zap.Uint("userID", userID))
			return nil, domain.ErrUserNotFound
		}
		domain.Error(ctx, "Failed to get user", err, zap.Uint("userID", userID))
		return nil, domain.ErrDB
	}

	// Получаем посты
	posts, err := s.postStore.GetPostsByUser(ctx, selfUserID, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get user posts", err, zap.Uint("userID", userID))
		return nil, domain.ErrDB
	}

	return posts, nil
}

func (s *PostService) UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int) error {
	err := s.postStore.UpdateLikeOnPostByUserID(ctx, userID, postID)
	if err != nil {

		domain.FromContext(ctx).Error("Failed update like on post", zap.Error(err))
		return domain.ErrDB

	}

	domain.FromContext(ctx).Info("like on post updated")
	return nil
}

// convertToPointerSlice вспомогательная функция для конвертации
func convertToPointerSlice(slice []string) []*string {
	result := make([]*string, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}
