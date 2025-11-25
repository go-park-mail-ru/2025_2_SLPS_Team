package service

import (
	"context"
	"errors"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"

	"go.uber.org/zap"
)

type FriendService struct {
	friendStore         domain.FriendStore
	authService         pb.AuthServiceClient
	elasticProfileStore domain.ElasticProfileStore
	profileService      pb.ProfileServiceClient
}

func NewFriendService(friendStore domain.FriendStore, authService pb.AuthServiceClient, elasticProfileStore domain.ElasticProfileStore, profileService pb.ProfileServiceClient) domain.FriendService {
	return &FriendService{
		friendStore:         friendStore,
		authService:         authService,
		profileService:      profileService,
		elasticProfileStore: elasticProfileStore,
	}
}

// SendFriendRequest отправляет запрос в друзья
func (s *FriendService) SendFriendRequest(ctx context.Context, actionUserID, targetUserID int32) error {
	// Нельзя отправить запрос самому себе
	if actionUserID == targetUserID {
		domain.Warn(ctx, "User tried to send friend request to themselves")
		return domain.ErrInvalidInput
	}

	domain.Info(ctx, "Sending friend request",
		zap.Int32("actionUserID", actionUserID),
		zap.Int32("targetUserID", targetUserID))

	// Проверяем существование пользователя
	resp, err := s.authService.IsUserExists(ctx, &pb.UserIDRequest{UserId: targetUserID})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check user existence", zap.Error(err))
		return domain.ErrDB
	}
	isUserExist := resp.Exists
	if !isUserExist {
		domain.FromContext(ctx).Warn("User not found")
		return domain.ErrNotExist
	}

	// Проверяем текущий статус дружбы
	currentStatus, err := s.friendStore.GetFriendshipStatus(ctx, actionUserID, targetUserID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		domain.Error(ctx, "Failed to check friendship status", err)
		return domain.ErrDB
	}

	// Обработка различных статусов
	switch currentStatus {
	case domain.FriendshipAccepted:
		domain.Warn(ctx, "Friend request to existing friend")
		return domain.ErrAlreadyExists
	case domain.FriendshipPending:
		// Определяем кто отправитель запроса
		friendship, err := s.friendStore.GetFriendship(ctx, actionUserID, targetUserID)
		if err != nil {
			domain.Error(ctx, "Failed to get friendship details", err)
			return domain.ErrDB
		}

		if friendship.FirstUserID == actionUserID {
			domain.Warn(ctx, "Duplicate friend request")
			return domain.ErrAlreadyExists
		} else {
			domain.Warn(ctx, "Friend request to user who already sent request")
			return domain.ErrAlreadyExists
		}
	case domain.FriendshipBlocked:
		domain.Warn(ctx, "Friend request to blocked user")
		return domain.ErrAccessDenied
	}

	// Создаем запрос в друзья
	err = s.friendStore.CreateFriendship(ctx, actionUserID, targetUserID)
	if err != nil {
		domain.Error(ctx, "Failed to send friend request", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Friend request sent successfully")
	return nil
}

// AcceptFriendRequest принимает запрос в друзья
func (s *FriendService) AcceptFriendRequest(ctx context.Context, userID, friendID int32) error {
	domain.Info(ctx, "Accepting friend request",
		zap.Int32("userID", userID),
		zap.Int32("friendID", friendID))

	// Проверяем существование запроса
	friendship, err := s.friendStore.GetFriendship(ctx, userID, friendID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Friend request not found")
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get friendship", err)
		return domain.ErrDB
	}

	// Проверяем что запрос pending, пользователь является получателем (не отправителем)
	if friendship.Status != domain.FriendshipPending || friendship.ActionUserID == userID {
		domain.Warn(ctx, "No pending friend request or user is sender (not receiver)")
		return domain.ErrNotFound
	}

	err = s.friendStore.UpdateFriendshipStatus(ctx, userID, friendID, domain.FriendshipAccepted)
	if err != nil {
		domain.Error(ctx, "Failed to accept friend request", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Friend request accepted successfully")
	return nil
}

// RejectFriendRequest отклоняет запрос в друзья
func (s *FriendService) RejectFriendRequest(ctx context.Context, userID, friendID int32) error {
	domain.Info(ctx, "Rejecting friend request",
		zap.Int32("userID", userID),
		zap.Int32("friendID", friendID))

	// Проверяем существование запроса
	friendship, err := s.friendStore.GetFriendship(ctx, userID, friendID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Friend request not found")
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get friendship", err)
		return domain.ErrDB
	}

	// Проверяем что запрос pending, пользователь является получателем (не отправителем)
	if friendship.Status != domain.FriendshipPending || friendship.ActionUserID == userID {
		domain.Warn(ctx, "No pending friend request or user is sender (not receiver)")
		return domain.ErrNotFound
	}

	// Удаляем запись вместо установки статуса rejected

	err = s.friendStore.UpdateFriendshipStatus(ctx, userID, friendID, domain.FriendshipRejected)
	if err != nil {
		domain.Error(ctx, "Failed to reject friend request", err)
		return domain.ErrDB
	}
	domain.Info(ctx, "Friend request rejected successfully")
	return nil
}

// RemoveFriend удаляет из друзей
func (s *FriendService) RemoveFriend(ctx context.Context, userID, friendID int32) error {
	domain.Info(ctx, "Removing friend",
		zap.Int32("userID", userID),
		zap.Int32("friendID", friendID))

	err := s.friendStore.DeleteFriendship(ctx, userID, friendID)
	if err != nil {
		domain.Error(ctx, "Failed to remove friend", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Friend removed successfully")
	return nil
}

// GetFriends получает список друзей
func (s *FriendService) GetFriends(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.ShortProfile, error) {
	// Валидация параметров
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting user friends",
		zap.Int32("userID", userID),
		zap.Int32("offset", offset),
		zap.Int32("limit", limit))

	friendIDs, err := s.friendStore.GetUserFriends(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get user friends", err)
		return nil, domain.ErrDB
	}

	friends, err := s.profileService.GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: friendIDs})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get profiles", zap.Error(err), zap.Int32s("authorIDs", friendIDs))
		return nil, domain.ErrDB
	}
	return generated.FromProtoShortProfileSlice(friends), nil
}

// GetAllUsers получает всех пользователей с пагинацией
func (s *FriendService) GetAllUsers(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.ShortProfile, error) {
	// Валидация параметров
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting all users except current",
		zap.Int32("currentUserID", userID),
		zap.Int32("offset", offset),
		zap.Int32("limit", limit))

	userIDs, err := s.friendStore.GetAllUsers(ctx, userID)
	if err != nil {
		domain.Error(ctx, "Failed to get all users", err)
		return nil, domain.ErrDB
	}
	friends, err := s.profileService.GetOtherShortProfileByUserIDs(ctx, &pb.GetOtherShortProfileByUserIDsRequest{UserIDs: userIDs, Limit: limit, Offset: offset})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get profiles", zap.Error(err), zap.Int32s("authorIDs", userIDs))
		return nil, domain.ErrDB
	}
	return generated.FromProtoOtherShortProfileSlice(friends), nil
}

// GetFriendRequests получает входящие запросы в друзья
func (s *FriendService) GetFriendRequests(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.ShortProfile, error) {
	// Валидация параметров
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting friendship requests",
		zap.Int32("userID", userID),
		zap.Int32("offset", offset),
		zap.Int32("limit", limit))

	requestIDs, err := s.friendStore.GetFriendshipRequests(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get friendship requests", err)
		return nil, domain.ErrDB
	}

	friends, err := s.profileService.GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: requestIDs})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get profiles", zap.Error(err), zap.Int32s("authorIDs", requestIDs))
		return nil, domain.ErrDB
	}
	return generated.FromProtoShortProfileSlice(friends), nil
}

// GetSentRequests получает отправленные запросы в друзья
func (s *FriendService) GetSentRequests(ctx context.Context, userID int32, params domain.PaginateQueryParams) ([]domain.ShortProfile, error) {
	// Валидация параметров
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting sent friend requests",
		zap.Int32("userID", userID),
		zap.Int32("offset", offset),
		zap.Int32("limit", limit))

	requestIDs, err := s.friendStore.GetSentRequests(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get sent requests", err)
		return nil, domain.ErrDB
	}

	friends, err := s.profileService.GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: requestIDs})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to get profiles", zap.Error(err), zap.Int32s("authorIDs", requestIDs))
		return nil, domain.ErrDB
	}
	return generated.FromProtoShortProfileSlice(friends), nil
}

// GetFriendshipStatus получает статус дружбы с пользователем
func (s *FriendService) GetFriendshipStatus(ctx context.Context, userID, friendID int32) (domain.FriendshipStatus, error) {
	domain.Info(ctx, "Getting friendship status",
		zap.Int32("userID", userID),
		zap.Int32("friendID", friendID))

	status, err := s.friendStore.GetFriendshipStatus(ctx, userID, friendID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		domain.Error(ctx, "Failed to get friendship status", err)
		return "", domain.ErrDB
	}

	domain.Info(ctx, "Friendship status retrieved successfully")
	return status, nil
}

// CountUserRelations подсчитывает количество отношений пользователя по типу
func (s *FriendService) CountUserRelations(ctx context.Context, userID int32) (*domain.UserRelationsCounts, error) {
	domain.Info(ctx, "Counting user relations",
		zap.Int32("userID", userID))

	// Проверяем существование пользователя
	resp, err := s.authService.IsUserExists(ctx, &pb.UserIDRequest{UserId: userID})
	if err != nil {
		domain.FromContext(ctx).Error("Failed to check user existence", zap.Error(err))
		return nil, domain.ErrDB
	}
	isUserExist := resp.Exists
	if !isUserExist {
		domain.FromContext(ctx).Warn("User not found")
		return nil, domain.ErrNotExist
	}

	count, err := s.friendStore.CountUserRelations(ctx, userID)
	if err != nil {
		domain.Error(ctx, "Failed to count user relations", err)
		return nil, domain.ErrDB
	}

	domain.Info(ctx, "User relations counted successfully",
		zap.Int32("userID", userID))
	return count, nil
}

// isValidCountType проверяет валидность типа подсчета
func (s *FriendService) isValidCountType(countType domain.FriendshipCountType) bool {
	validTypes := map[domain.FriendshipCountType]bool{
		domain.CountAccepted: true,
		domain.CountPending:  true,
		domain.CountSent:     true,
		domain.CountBlocked:  true,
		domain.CountRejected: true,
	}
	return validTypes[countType]
}

func (s *FriendService) SearchShortProfilesByFullNameAndRelationType(ctx context.Context, userID int32, params domain.PaginateQueryParams, fullName string, fType domain.FriendshipCountType) ([]domain.ShortProfile, error) {

	offset, limit := domain.ValidatePaginationParams(params)
	isTerms := true
	if fType == domain.CountNotFriends {
		isTerms = false
	}
	filterIDs, err := s.friendStore.GetUserIDsByFriendType(ctx, userID, fType)
	if err != nil {
		domain.FromContext(ctx).Error("Fail find user relations by type", zap.Error(err))
		return nil, domain.ErrDB
	}
	foundIDs, err := s.elasticProfileStore.SearchUserIDsByFullNameWithFilter(ctx, fullName, filterIDs, isTerms, limit, offset)
	if err != nil {
		domain.FromContext(ctx).Error("Fail find user IDs by FullName", zap.Error(err))
		return nil, domain.ErrDB
	}
	resp, err := s.profileService.GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: foundIDs})
	if err != nil {
		domain.FromContext(ctx).Error("Fail get short Profiles by user IDs", zap.Error(err))
		return nil, domain.ErrDB
	}

	return generated.FromProtoShortProfileSlice(resp), nil
}
