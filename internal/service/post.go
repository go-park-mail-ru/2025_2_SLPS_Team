package service

import (
	"context"
	"errors"
	"project/domain"
	"project/shared/pb"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type PostService struct {
	postStore      domain.PostStore
	userStore      domain.UserStore
	communityStore domain.CommunityStore
	profileClient  pb.ProfileServiceClient
}

func NewPostService(postStore domain.PostStore, userStore domain.UserStore, communityStore domain.CommunityStore, profileClient pb.ProfileServiceClient) domain.PostService {
	return &PostService{
		postStore:      postStore,
		userStore:      userStore,
		communityStore: communityStore,
		profileClient:  profileClient,
	}
}

// PostsPaginate возвращает посты с пагинацией с обогащением данных профилей через gRPC
func (s *PostService) PostsPaginate(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.PostView, error) {
	offset, limit := domain.ValidatePaginationParams(params)
	domain.Info(ctx, "Getting paginated posts", zap.Int32("offset", offset), zap.Int32("limit", limit))

	// Получаем посты из БД без информации о профиле
	postsDB, err := s.postStore.PostsPaginatedList(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get posts", err)
		return nil, domain.ErrDB
	}

	// Обогащаем данные профилями через gRPC
	postsView, err := s.enrichPostsWithProfiles(ctx, postsDB)
	if err != nil {
		return nil, err
	}

	return postsView, nil
}

// GetPost возвращает пост по ID с обогащением данных профиля через gRPC
func (s *PostService) GetPost(ctx context.Context, userID int32, postID uint) (*domain.PostView, error) {
	domain.Info(ctx, "Getting post by ID", zap.Uint("postID", postID))

	postDB, err := s.postStore.GetPostByID(ctx, userID, postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			domain.Warn(ctx, "Post not found", zap.Uint("postID", postID))
			return nil, domain.ErrPostNotFound
		}
		domain.Error(ctx, "Failed to get post", err, zap.Uint("postID", postID))
		return nil, domain.ErrDB
	}

	// Обогащаем данные профилем через gRPC
	postView, err := s.enrichPostWithProfile(ctx, postDB)
	if err != nil {
		return nil, err
	}

	return postView, nil
}

// CreatePost создает новый пост
func (s *PostService) CreatePost(ctx context.Context, userID int32, text string, communityID *int32, attachmentFiles []*domain.File, photoFiles []*domain.File) (*domain.Post, error) {

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

	domain.Info(ctx, "Creating new post", zap.Int32("userID", userID))

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

	// Если указан communityID, проверяем права
	if communityID != nil {
		community, err := s.communityStore.GetCommunityByID(ctx, *communityID)
		if err != nil {
			domain.Warn(ctx, "Community not found", zap.Int32("communityID", *communityID))
			return nil, domain.ErrNotFound
		}

		// Проверяем, является ли пользователь создателем сообщества
		if community.CreatorID != userID {
			domain.Warn(ctx, "User is not community creator",
				zap.Int32("userID", userID),
				zap.Int32("creatorID", community.CreatorID))
			return nil, domain.ErrAccessDenied
		}
	}

	// Создаем объект поста
	post := &domain.Post{
		AuthorID:    uint(userID),
		CommunityID: communityID,
		Text:        createRequest.Text,
		Attachments: createRequest.Attachments,
		Photos:      createRequest.Photos,
	}

	// Сохраняем в БД
	if err := s.postStore.CreatePost(ctx, post); err != nil {
		if len(attachmentPaths) > 0 {
			DeleteFiles(convertToPointerSlice(attachmentPaths))
		}
		if len(photoPaths) > 0 {
			DeleteFiles(convertToPointerSlice(photoPaths))
		}
		domain.Error(ctx, "Failed to create post", err, zap.Int32("userID", userID))
		return nil, domain.ErrDB
	}

	domain.Info(ctx, "Post created successfully",
		zap.Uint("postID", post.ID),
		zap.Int32("userID", userID),
		zap.Int("attachmentsCount", len(attachmentPaths)),
		zap.Int("photosCount", len(photoPaths)))

	return post, nil
}

// UpdatePost обновляет пост
func (s *PostService) UpdatePost(ctx context.Context, postID uint, userID int32, text string, attachmentFiles []*domain.File, photoFiles []*domain.File) error {
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

	domain.Info(ctx, "Updating post", zap.Uint("postID", postID), zap.Int32("userID", userID))

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
			zap.Int32("userID", userID),
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
		for i := range existingPost.Photos {
			oldPhotos = append(oldPhotos, &existingPost.Photos[i])
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
		newPhotoPaths = existingPost.Photos
		updateRequest.Photos = existingPost.Photos
	}

	// Обновляем данные поста
	updatedPost := &domain.Post{
		ID:          postID,
		AuthorID:    uint(userID),
		Text:        updateRequest.Text,
		CreatedAt:   existingPost.CreatedAt,
		Attachments: updateRequest.Attachments,
		Photos:      updateRequest.Photos,
	}

	if err := s.postStore.UpdatePost(ctx, updatedPost); err != nil {
		if len(newAttachmentPaths) > len(existingPost.Attachments) {
			newFiles := newAttachmentPaths[len(existingPost.Attachments):]
			DeleteFiles(convertToPointerSlice(newFiles))
		}
		if len(newPhotoPaths) > len(existingPost.Photos) {
			newFiles := newPhotoPaths[len(existingPost.Photos):]
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
func (s *PostService) DeletePost(ctx context.Context, postID uint, userID int32) error {
	domain.Info(ctx, "Deleting post", zap.Uint("postID", postID), zap.Int32("userID", userID))

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
			zap.Int32("userID", userID),
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
	for i := range existingPost.Photos {
		filesToDelete = append(filesToDelete, &existingPost.Photos[i])
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

// GetUserPosts возвращает посты пользователя с обогащением данных профилей через gRPC
func (s *PostService) GetUserPosts(ctx context.Context, selfUserID int32, userID uint, params domain.PaginateQueryParams) ([]domain.PostView, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting user posts", zap.Uint("userID", userID))

	// Проверяем существование пользователя
	_, err := s.userStore.GetUserByID(ctx, int32(userID))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			domain.Warn(ctx, "User not found", zap.Uint("userID", userID))
			return nil, domain.ErrUserNotFound
		}
		domain.Error(ctx, "Failed to get user", err, zap.Uint("userID", userID))
		return nil, domain.ErrDB
	}

	// Получаем посты из БД без информации о профиле
	postsDB, err := s.postStore.GetPostsByUser(ctx, selfUserID, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get user posts", err, zap.Uint("userID", userID))
		return nil, domain.ErrDB
	}

	// Обогащаем данные профилями через gRPC
	postsView, err := s.enrichPostsWithProfiles(ctx, postsDB)
	if err != nil {
		return nil, err
	}

	return postsView, nil
}

// GetCommunityPosts возвращает посты сообщества с обогащением данных профилей через gRPC
func (s *PostService) GetCommunityPosts(ctx context.Context, userID int32, communityID int32, params domain.PaginateQueryParams) ([]domain.PostView, error) {
	offset, limit := domain.ValidatePaginationParams(params)
	domain.Info(ctx, "Getting community posts", zap.Int32("communityID", communityID))

	// Проверяем существование сообщества
	_, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int32("communityID", communityID))
			return nil, domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return nil, domain.ErrDB
	}

	// Получаем посты из БД без информации о профиле
	postsDB, err := s.postStore.GetCommunityPosts(ctx, userID, communityID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get community posts", err)
		return nil, domain.ErrDB
	}

	// Обогащаем данные профилями через gRPC
	postsView, err := s.enrichPostsWithProfiles(ctx, postsDB)
	if err != nil {
		return nil, err
	}

	return postsView, nil
}

func (s *PostService) UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int32) error {
	err := s.postStore.UpdateLikeOnPostByUserID(ctx, userID, postID)
	if err != nil {

		domain.FromContext(ctx).Error("Failed update like on post", zap.Error(err))
		return domain.ErrDB

	}

	domain.FromContext(ctx).Info("like on post updated")
	return nil
}

//Вспомогательные методы

// convertToPointerSlice вспомогательная функция для конвертации
func convertToPointerSlice(slice []string) []*string {
	result := make([]*string, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}

// enrichPostsWithProfiles обогащает список постов данными профилей через gRPC
func (s *PostService) enrichPostsWithProfiles(ctx context.Context, postsDB []domain.PostDB) ([]domain.PostView, error) {
	if len(postsDB) == 0 {
		return []domain.PostView{}, nil
	}

	// Собираем ID авторов для запроса профилей
	authorIDs := make([]int32, 0, len(postsDB))
	for _, post := range postsDB {
		authorIDs = append(authorIDs, int32(post.AuthorID))
	}

	// Получаем профили через gRPC
	profilesResp, err := s.profileClient.GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
		UserIDs: authorIDs,
	})
	if err != nil {
		domain.Error(ctx, "Failed to get profiles via gRPC", err)
		return nil, domain.ErrService
	}

	// Преобразуем в доменную структуру
	profilesMap := make(map[int32]domain.ShortProfile)
	for userID, pbProfile := range profilesResp.Profiles {
		profilesMap[userID] = domain.ShortProfile{
			UserID:     pbProfile.UserID,
			FullName:   pbProfile.FullName,
			AvatarPath: pbProfile.AvatarPath,
			Dob:        pbProfile.Dob.AsTime(),
		}
	}

	// Собираем результат
	postsView := make([]domain.PostView, 0, len(postsDB))
	for _, postDB := range postsDB {
		postView := domain.PostView{
			ID:              postDB.ID,
			AuthorID:        postDB.AuthorID,
			CommunityID:     postDB.CommunityID,
			Text:            postDB.Text,
			Attachments:     postDB.Attachments,
			Photos:          postDB.Photos,
			LikeCount:       postDB.LikeCount,
			IsLiked:         postDB.IsLiked,
			CreatedAt:       postDB.CreatedAt,
			IsCommunityPost: postDB.CommunityID != nil,
			CommunityName:   postDB.CommunityName,
			CommunityAvatar: postDB.CommunityAvatar,
		}

		// Заполняем данные автора из профиля
		if profile, exists := profilesMap[int32(postDB.AuthorID)]; exists {
			postView.AuthorName = profile.FullName
			postView.AuthorAvatar = profile.AvatarPath
		} else {
			domain.Warn(ctx, "Profile not found for user", zap.Uint("authorID", postDB.AuthorID))
			// Устанавливаем значения по умолчанию
			postView.AuthorName = "Пользователь"
		}

		postsView = append(postsView, postView)
	}

	return postsView, nil
}

// enrichPostWithProfile обогащает один пост данными профиля через gRPC
func (s *PostService) enrichPostWithProfile(ctx context.Context, postDB *domain.PostDB) (*domain.PostView, error) {
	// Получаем профиль автора через gRPC
	profilesResp, err := s.profileClient.GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
		UserIDs: []int32{int32(postDB.AuthorID)},
	})
	if err != nil {
		domain.Error(ctx, "Failed to get profile via gRPC", err)
		return nil, domain.ErrService
	}

	postView := &domain.PostView{
		ID:              postDB.ID,
		AuthorID:        postDB.AuthorID,
		CommunityID:     postDB.CommunityID,
		Text:            postDB.Text,
		Attachments:     postDB.Attachments,
		Photos:          postDB.Photos,
		LikeCount:       postDB.LikeCount,
		IsLiked:         postDB.IsLiked,
		CreatedAt:       postDB.CreatedAt,
		IsCommunityPost: postDB.CommunityID != nil,
		CommunityName:   postDB.CommunityName,
		CommunityAvatar: postDB.CommunityAvatar,
	}

	// Заполняем данные автора из профиля
	if profile, exists := profilesResp.Profiles[int32(postDB.AuthorID)]; exists {
		postView.AuthorName = profile.FullName
		postView.AuthorAvatar = profile.AvatarPath
	} else {
		domain.Warn(ctx, "Profile not found for user", zap.Uint("authorID", postDB.AuthorID))
		postView.AuthorName = "Пользователь"
	}

	return postView, nil
}
