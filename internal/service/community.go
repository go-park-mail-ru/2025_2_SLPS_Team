package service

import (
	"context"
	"errors"
	"mime/multipart"
	"project/domain"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type CommunityService struct {
	communityStore domain.CommunityStore
	postStore      domain.PostStore
	userStore      domain.UserStore
}

func NewCommunityService(communityStore domain.CommunityStore, postStore domain.PostStore, userStore domain.UserStore) domain.CommunityService {
	return &CommunityService{
		communityStore: communityStore,
		postStore:      postStore,
		userStore:      userStore,
	}
}

func (s *CommunityService) CreateCommunity(ctx context.Context, userID int, req domain.CommunityRequest, avatarFile *multipart.FileHeader, coverFile *multipart.FileHeader) (*domain.Community, error) {
	// Валидация
	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		domain.Warn(ctx, "Community validation failed", zap.Error(err))
		return nil, domain.ErrInvalidInput
	}

	domain.Info(ctx, "Creating community", zap.Int("userID", userID))

	// Обработка аватара
	var avatarPath *string
	if avatarFile != nil {
		path, err := UploadFile(avatarFile)
		if err != nil {
			domain.Error(ctx, "Failed to upload avatar", err)
			return nil, domain.ErrService
		}
		avatarPath = &path
	}

	// Обработка обложки
	var coverPath *string
	if coverFile != nil {
		path, err := UploadFile(coverFile)
		if err != nil {
			if avatarPath != nil {
				DeleteFile(*avatarPath)
			}
			domain.Error(ctx, "Failed to upload cover", err)
			return nil, domain.ErrService
		}
		coverPath = &path
	}

	community := &domain.Community{
		Name:        req.Name,
		Description: req.Description,
		CreatorID:   userID,
		AvatarPath:  avatarPath,
		CoverPath:   coverPath,
	}

	if err := s.communityStore.CreateCommunity(ctx, community); err != nil {
		if avatarPath != nil {
			DeleteFile(*avatarPath)
		}
		if coverPath != nil {
			DeleteFile(*coverPath)
		}
		domain.Error(ctx, "Failed to create community", err)
		return nil, domain.ErrDB
	}

	// Автоматически подписываем создателя
	if err := s.communityStore.Subscribe(ctx, community.ID, userID); err != nil {
		domain.Error(ctx, "Failed to auto-subscribe creator", err)
		// Не прерываем выполнение, так как сообщество уже создано
	}

	domain.Info(ctx, "Community created successfully", zap.Int("communityID", community.ID))
	return community, nil
}

func (s *CommunityService) UpdateCommunity(ctx context.Context, communityID int, userID int, req domain.CommunityRequest, avatarFile *multipart.FileHeader, coverFile *multipart.FileHeader) error {
	// Валидация
	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		domain.Warn(ctx, "Community validation failed", zap.Error(err))
		return domain.ErrInvalidInput
	}

	domain.Info(ctx, "Updating community", zap.Int("communityID", communityID), zap.Int("userID", userID))

	// Получаем текущее сообщество
	existingCommunity, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int("communityID", communityID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return domain.ErrDB
	}

	// Проверяем права доступа
	if existingCommunity.CreatorID != userID {
		domain.Warn(ctx, "Access denied: user is not community creator",
			zap.Int("communityID", communityID),
			zap.Int("userID", userID),
			zap.Int("creatorID", existingCommunity.CreatorID))
		return domain.ErrAccessDenied
	}

	// Подготавливаем старые пути для удаления
	var oldAvatarPath, oldCoverPath *string

	// Обрабатываем новый аватар
	var newAvatarPath *string
	if avatarFile != nil {
		oldAvatarPath = existingCommunity.AvatarPath
		path, err := UploadFile(avatarFile)
		if err != nil {
			domain.Error(ctx, "Failed to upload new avatar", err)
			return domain.ErrService
		}
		newAvatarPath = &path
	} else {
		newAvatarPath = existingCommunity.AvatarPath
	}

	// Обрабатываем новую обложку
	var newCoverPath *string
	if coverFile != nil {
		oldCoverPath = existingCommunity.CoverPath
		path, err := UploadFile(coverFile)
		if err != nil {
			if newAvatarPath != existingCommunity.AvatarPath {
				DeleteFile(*newAvatarPath)
			}
			domain.Error(ctx, "Failed to upload new cover", err)
			return domain.ErrService
		}
		newCoverPath = &path
	} else {
		newCoverPath = existingCommunity.CoverPath
	}

	// Обновляем данные
	updatedCommunity := &domain.Community{
		ID:          communityID,
		Name:        req.Name,
		Description: req.Description,
		CreatorID:   userID,
		AvatarPath:  newAvatarPath,
		CoverPath:   newCoverPath,
	}

	if err := s.communityStore.UpdateCommunity(ctx, updatedCommunity); err != nil {
		if newAvatarPath != existingCommunity.AvatarPath {
			DeleteFile(*newAvatarPath)
		}
		if newCoverPath != existingCommunity.CoverPath {
			DeleteFile(*newCoverPath)
		}
		domain.Error(ctx, "Failed to update community", err)
		return domain.ErrDB
	}

	// Удаляем старые файлы
	if oldAvatarPath != nil {
		DeleteFile(*oldAvatarPath)
	}
	if oldCoverPath != nil {
		DeleteFile(*oldCoverPath)
	}

	domain.Info(ctx, "Community updated successfully")
	return nil
}

func (s *CommunityService) DeleteCommunity(ctx context.Context, communityID int, userID int) error {
	domain.Info(ctx, "Deleting community", zap.Int("communityID", communityID), zap.Int("userID", userID))

	// Получаем сообщество для проверки прав и получения путей файлов
	existingCommunity, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int("communityID", communityID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return domain.ErrDB
	}

	// Проверяем права доступа
	if existingCommunity.CreatorID != userID {
		domain.Warn(ctx, "Access denied: user is not community creator",
			zap.Int("communityID", communityID),
			zap.Int("userID", userID),
			zap.Int("creatorID", existingCommunity.CreatorID))
		return domain.ErrAccessDenied
	}

	// Удаляем сообщество
	if err := s.communityStore.DeleteCommunity(ctx, communityID, userID); err != nil {
		domain.Error(ctx, "Failed to delete community", err)
		return domain.ErrDB
	}

	// Удаляем файлы
	if existingCommunity.AvatarPath != nil {
		DeleteFile(*existingCommunity.AvatarPath)
	}
	if existingCommunity.CoverPath != nil {
		DeleteFile(*existingCommunity.CoverPath)
	}

	domain.Info(ctx, "Community deleted successfully")
	return nil
}

func (s *CommunityService) GetCommunity(ctx context.Context, userID int, communityID int) (*domain.CommunityForView, error) {
	domain.Info(ctx, "Getting community", zap.Int("communityID", communityID))

	community, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int("communityID", communityID))
			return nil, domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return nil, domain.ErrDB
	}

	// Проверяем подписку пользователя
	isSubscribed, err := s.communityStore.IsSubscribed(ctx, communityID, userID)
	if err != nil {
		domain.Error(ctx, "Failed to check subscription", err)
		return nil, domain.ErrDB
	}

	// Преобразуем в структуру без CreatorID
	result := &domain.CommunityForView{
		ID:               community.ID,
		Name:             community.Name,
		Description:      community.Description,
		CreatorID:        community.CreatorID,
		AvatarPath:       community.AvatarPath,
		CoverPath:        community.CoverPath,
		CreatedAt:        community.CreatedAt,
		SubscribersCount: community.SubscribersCount,
		IsSubscribed:     isSubscribed,
	}

	return result, nil
}

func (s *CommunityService) GetUserCommunities(ctx context.Context, userID int, params domain.PaginateQueryParams) ([]domain.ShortCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting user communities", zap.Int("userID", userID))
	communities, err := s.communityStore.GetUserCommunities(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get user communities", err)
		return nil, domain.ErrDB
	}

	return communities, nil
}

func (s *CommunityService) GetOtherCommunities(ctx context.Context, userID int, params domain.PaginateQueryParams) ([]domain.ShortCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)
	domain.Info(ctx, "Getting other communities", zap.Int("userID", userID))

	communities, err := s.communityStore.GetOtherCommunities(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get other communities", err)
		return nil, domain.ErrDB
	}

	return communities, nil
}

func (s *CommunityService) GetUserCommunitiesByID(ctx context.Context, targetUserID int, params domain.PaginateQueryParams) ([]domain.ShortCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting user communities by ID", zap.Int("targetUserID", targetUserID))

	// Проверяем существование пользователя
	_, err := s.userStore.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "User not found", zap.Int("targetUserID", targetUserID))
			return nil, domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get user", err)
		return nil, domain.ErrDB
	}

	communities, err := s.communityStore.GetUserCommunitiesByID(ctx, targetUserID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get user communities by ID", err)
		return nil, domain.ErrDB
	}

	return communities, nil
}

func (s *CommunityService) GetMyCommunityIDs(ctx context.Context, userID int) ([]int, error) {
	domain.Info(ctx, "Getting my community IDs", zap.Int("userID", userID))

	communityIDs, err := s.communityStore.GetMyCommunityIDs(ctx, userID)
	if err != nil {
		domain.Error(ctx, "Failed to get my community IDs", err)
		return nil, domain.ErrDB
	}

	return communityIDs, nil
}

func (s *CommunityService) GetCreatedCommunities(ctx context.Context, userID int, params domain.PaginateQueryParams) ([]domain.CommunityForMyCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting created communities", zap.Int("userID", userID))
	communities, err := s.communityStore.GetCreatedCommunities(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get created communities", err)
		return nil, domain.ErrDB
	}

	return communities, nil
}

func (s *CommunityService) GetCommunitySubscribers(ctx context.Context, communityID int, params domain.PaginateQueryParams) ([]domain.CommunitySubscriber, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting community subscribers", zap.Int("communityID", communityID))

	// Проверяем существование сообщества
	_, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int("communityID", communityID))
			return nil, domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return nil, domain.ErrDB
	}

	subscribers, err := s.communityStore.GetCommunitySubscribers(ctx, communityID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get community subscribers", err)
		return nil, domain.ErrDB
	}

	return subscribers, nil
}

func (s *CommunityService) Subscribe(ctx context.Context, communityID int, userID int) error {
	domain.Info(ctx, "Subscribing to community", zap.Int("communityID", communityID), zap.Int("userID", userID))

	// Проверяем существование сообщества
	_, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int("communityID", communityID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return domain.ErrDB
	}

	// Проверяем существование пользователя
	_, err = s.userStore.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "User not found", zap.Int("userID", userID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get user", err)
		return domain.ErrDB
	}

	if err := s.communityStore.Subscribe(ctx, communityID, userID); err != nil {
		domain.Error(ctx, "Failed to subscribe", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Subscribed successfully")
	return nil
}

func (s *CommunityService) Unsubscribe(ctx context.Context, communityID int, userID int) error {
	domain.Info(ctx, "Unsubscribing from community", zap.Int("communityID", communityID), zap.Int("userID", userID))

	if err := s.communityStore.Unsubscribe(ctx, communityID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Subscription not found")
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to unsubscribe", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Unsubscribed successfully")
	return nil
}
