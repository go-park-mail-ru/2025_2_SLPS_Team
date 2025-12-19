package service

import (
	"context"
	"errors"
	"log"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type CommunityService struct {
	communityStore        domain.CommunityStore
	postStore             domain.PostStore
	authService           pb.AuthServiceClient
	elasticCommunityStore domain.ElasticCommunityStore
	profileService        pb.ProfileServiceClient
}

func NewCommunityService(communityStore domain.CommunityStore, postStore domain.PostStore, authService pb.AuthServiceClient, elasticCommunityStore domain.ElasticCommunityStore, profileService pb.ProfileServiceClient) domain.CommunityService {
	return &CommunityService{
		communityStore:        communityStore,
		postStore:             postStore,
		authService:           authService,
		elasticCommunityStore: elasticCommunityStore,
		profileService:        profileService,
	}
}

func (s *CommunityService) CreateCommunity(ctx context.Context, userID int32, req domain.CommunityRequest, avatarFiles []*domain.File, coverFiles []*domain.File) (*domain.Community, error) {
	// Валидация
	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		domain.Warn(ctx, "Community validation failed", zap.Error(err))
		return nil, domain.ErrInvalidInput
	}

	domain.Info(ctx, "Creating community", zap.Int32("userID", userID))

	// Обработка аватара
	var avatarPath *string
	if avatarFiles != nil {
		path, err := UploadFile(avatarFiles[0])
		if err != nil {
			domain.Error(ctx, "Failed to upload avatar", err)
			return nil, domain.ErrService
		}
		avatarPath = &path
	}

	// Обработка обложки
	var coverPath *string
	if coverFiles != nil {
		path, err := UploadFile(coverFiles[0])
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

	err = s.elasticCommunityStore.CreateCommunity(ctx, community.Name, community.ID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to create community index in es", zap.Error(err))
	}

	// Автоматически подписываем создателя
	if err := s.communityStore.Subscribe(ctx, community.ID, userID); err != nil {
		domain.Error(ctx, "Failed to auto-subscribe creator", err)
		// Не прерываем выполнение, так как сообщество уже создано
	}

	domain.Info(ctx, "Community created successfully", zap.Int32("communityID", community.ID))
	return community, nil
}

func (s *CommunityService) UpdateCommunity(ctx context.Context, communityID int32, userID int32, req domain.CommunityRequest, avatarFiles []*domain.File, coverFiles []*domain.File) error {
	// Валидация
	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		domain.Warn(ctx, "Community validation failed", zap.Error(err))
		return domain.ErrInvalidInput
	}

	domain.Info(ctx, "Updating community", zap.Int32("communityID", communityID), zap.Int32("userID", userID))

	// Получаем текущее сообщество
	existingCommunity, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int32("communityID", communityID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return domain.ErrDB
	}

	// Проверяем права доступа
	if existingCommunity.CreatorID != userID {
		domain.Warn(ctx, "Access denied: user is not community creator",
			zap.Int32("communityID", communityID),
			zap.Int32("userID", userID),
			zap.Int32("creatorID", existingCommunity.CreatorID))
		return domain.ErrAccessDenied
	}

	// Подготавливаем старые пути для удаления
	var oldAvatarPath, oldCoverPath *string

	// Обрабатываем новый аватар
	var newAvatarPath *string
	if avatarFiles != nil {
		oldAvatarPath = existingCommunity.AvatarPath
		path, err := UploadFile(avatarFiles[0])
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
	if coverFiles != nil {
		oldCoverPath = existingCommunity.CoverPath
		path, err := UploadFile(coverFiles[0])
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

	err = s.elasticCommunityStore.UpdateCommunity(ctx, updatedCommunity.Name, updatedCommunity.ID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to update community index in es", zap.Error(err))
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

func (s *CommunityService) DeleteCommunity(ctx context.Context, communityID int32, userID int32) error {
	domain.Info(ctx, "Deleting community", zap.Int32("communityID", communityID), zap.Int32("userID", userID))

	// Получаем сообщество для проверки прав и получения путей файлов
	existingCommunity, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int32("communityID", communityID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return domain.ErrDB
	}

	// Проверяем права доступа
	if existingCommunity.CreatorID != userID {
		domain.Warn(ctx, "Access denied: user is not community creator",
			zap.Int32("communityID", communityID),
			zap.Int32("userID", userID),
			zap.Int32("creatorID", existingCommunity.CreatorID))
		return domain.ErrAccessDenied
	}

	// Удаляем сообщество
	if err := s.communityStore.DeleteCommunity(ctx, communityID, userID); err != nil {
		domain.Error(ctx, "Failed to delete community", err)
		return domain.ErrDB
	}

	err = s.elasticCommunityStore.DeleteCommunity(ctx, communityID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to delete community index in es", zap.Error(err))
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

func (s *CommunityService) GetCommunity(ctx context.Context, userID int32, communityID int32) (*domain.CommunityForView, error) {
	domain.Info(ctx, "Getting community", zap.Int32("communityID", communityID))

	community, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int32("communityID", communityID))
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

func (s *CommunityService) GetUserCommunities(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.ShortCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting user communities", zap.Int32("userID", userID))
	communities, err := s.communityStore.GetUserCommunities(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get user communities", err)
		return nil, domain.ErrDB
	}

	return communities, nil
}

func (s *CommunityService) GetOtherCommunities(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.ShortCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)
	domain.Info(ctx, "Getting other communities", zap.Int32("userID", userID))

	communities, err := s.communityStore.GetOtherCommunities(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get other communities", err)
		return nil, domain.ErrDB
	}

	return communities, nil
}

func (s *CommunityService) GetUserCommunitiesByID(ctx context.Context, targetUserID int32, params domain.PaginateQueryParams) ([]domain.ShortCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting user communities by ID", zap.Int32("targetUserID", targetUserID))

	// Проверяем существование пользователя
	resp, err := s.authService.IsUserExists(ctx, &pb.UserIDRequest{UserId: targetUserID})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check user existence", zap.Error(err))
		return nil, domain.ErrDB
	}
	isUserExist := resp.Exists
	if !isUserExist {
		domain.FromContext(ctx).Warn("User not found")
		return nil, domain.ErrNotExist
	}

	communities, err := s.communityStore.GetUserCommunitiesByID(ctx, targetUserID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get user communities by ID", err)
		return nil, domain.ErrDB
	}

	return communities, nil
}

func (s *CommunityService) GetUserSubscribedCommunityIDs(ctx context.Context, targetUserID int32) ([]int32, error) {
	domain.Info(ctx, "Getting user subscribed community IDs", zap.Int32("targetUserID", targetUserID))

	// Проверяем существование пользователя
	resp, err := s.authService.IsUserExists(ctx, &pb.UserIDRequest{UserId: targetUserID})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check user existence", zap.Error(err))
		return nil, domain.ErrDB
	}
	isUserExist := resp.Exists
	if !isUserExist {
		domain.FromContext(ctx).Warn("User not found")
		return nil, domain.ErrNotExist
	}

	communityIDs, err := s.communityStore.GetUserSubscribedCommunityIDs(ctx, targetUserID)
	if err != nil {
		domain.Error(ctx, "Failed to get user subscribed community IDs", err)
		return nil, domain.ErrDB
	}

	return communityIDs, nil
}

func (s *CommunityService) GetCreatedCommunities(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.CommunityForMyCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting created communities", zap.Int32("userID", userID))
	communities, err := s.communityStore.GetCreatedCommunities(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get created communities", err)
		return nil, domain.ErrDB
	}

	return communities, nil
}

func (s *CommunityService) GetCommunitySubscribers(ctx context.Context, communityID int32, params domain.PaginateQueryParams) ([]domain.CommunitySubscriber, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting community subscribers", zap.Int32("communityID", communityID))

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

	// 1. Получаем только ID подписчиков из community store
	subscriberIDs, err := s.communityStore.GetCommunitySubscribers(ctx, communityID, int32(limit), int32(offset))
	if err != nil {
		domain.Error(ctx, "Failed to get community subscriber IDs", err)
		return nil, domain.ErrDB
	}

	if len(subscriberIDs) == 0 {
		return []domain.CommunitySubscriber{}, nil
	}

	// 2. Делаем gRPC запрос к профильному сервису
	profileResp, err := s.profileService.GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
		UserIDs: subscriberIDs,
	})
	if err != nil {
		domain.Error(ctx, "Failed to get subscriber profiles via gRPC", err)
		return nil, domain.ErrService
	}

	// 3. Конвертируем protobuf в доменные модели
	profiles := generated.FromProtoShortProfileMap(profileResp)

	// 4. Формируем результат
	subscribers := make([]domain.CommunitySubscriber, 0, len(subscriberIDs))
	for _, userID := range subscriberIDs {
		if profile, exists := profiles[userID]; exists {
			subscribers = append(subscribers, domain.CommunitySubscriber{
				UserID:     userID,
				FullName:   profile.FullName,
				AvatarPath: profile.AvatarPath,
			})
		}
	}

	domain.Info(ctx, "Community subscribers retrieved successfully",
		zap.Int("count", len(subscribers)))
	return subscribers, nil
}

func (s *CommunityService) Subscribe(ctx context.Context, communityID int32, userID int32) error {
	domain.Info(ctx, "Subscribing to community", zap.Int32("communityID", communityID), zap.Int32("userID", userID))

	// Проверяем существование сообщества
	_, err := s.communityStore.GetCommunityByID(ctx, communityID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Community not found", zap.Int32("communityID", communityID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get community", err)
		return domain.ErrDB
	}

	// Проверяем существование пользователя
	resp, err := s.authService.IsUserExists(ctx, &pb.UserIDRequest{UserId: userID})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check user existence", zap.Error(err))
		return domain.ErrDB
	}
	isUserExist := resp.Exists
	if !isUserExist {
		domain.FromContext(ctx).Warn("User not found")
		return domain.ErrNotExist
	}

	if err := s.communityStore.Subscribe(ctx, communityID, userID); err != nil {
		domain.Error(ctx, "Failed to subscribe", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Subscribed successfully")
	return nil
}

func (s *CommunityService) Unsubscribe(ctx context.Context, communityID int32, userID int32) error {
	domain.Info(ctx, "Unsubscribing from community", zap.Int32("communityID", communityID), zap.Int32("userID", userID))

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

func (s *CommunityService) SearchShortCommunityByNameAndType(ctx context.Context, userID int32, params domain.PaginateQueryParams, name string, cType domain.CommunityType) ([]domain.ShortCommunity, error) {
	offset, limit := domain.ValidatePaginationParams(params)
	isTerms := true
	if cType == domain.Recommended {
		isTerms = false
	}
	log.Println(isTerms)
	filterIDs, err := s.communityStore.GetUserSubscribedCommunityIDs(ctx, userID)
	if err != nil {
		domain.FromContext(ctx).Error("Fail find user relations by type", zap.Error(err))
		return nil, domain.ErrDB
	}
	log.Println(filterIDs)
	foundIDs, err := s.elasticCommunityStore.SearchCommunityIDsByName(ctx, name, filterIDs, isTerms, limit, offset)
	if err != nil {
		domain.FromContext(ctx).Error("Fail find user IDs by FullName", zap.Error(err))
		return nil, domain.ErrDB
	}
	log.Println(foundIDs)
	com, err := s.communityStore.GetCommunitiesByIDs(ctx, foundIDs)
	if err != nil {
		domain.FromContext(ctx).Error("Fail get short Profiles by user IDs", zap.Error(err))
		return nil, domain.ErrDB
	}

	return com, nil
}
