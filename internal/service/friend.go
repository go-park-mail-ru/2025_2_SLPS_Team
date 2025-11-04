package service

import (
	"context"
	"errors"
	"project/domain"

	"go.uber.org/zap"
)

type FriendService struct {
	friendStore domain.FriendStore
	userStore   domain.UserStore
}

func NewFriendService(friendStore domain.FriendStore, userStore domain.UserStore) domain.FriendService {
	return &FriendService{
		friendStore: friendStore,
		userStore:   userStore,
	}
}

// SendFriendRequest отправляет запрос в друзья
func (s *FriendService) SendFriendRequest(ctx context.Context, userID, friendID int) error {
	// Нельзя отправить запрос самому себе
	if userID == friendID {
		domain.Warn(ctx, "User tried to send friend request to themselves")
		return domain.ErrInvalidInput
	}

	domain.Info(ctx, "Sending friend request",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

	// Проверяем существование пользователя
	_, err := s.userStore.GetUserByID(ctx, friendID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Friend user not found", zap.Int("friendID", friendID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get user", err, zap.Int("friendID", friendID))
		return domain.ErrDB
	}

	// Проверяем текущий статус дружбы
	currentStatus, err := s.friendStore.GetFriendshipStatus(ctx, userID, friendID)
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
		friendship, err := s.friendStore.GetFriendship(ctx, userID, friendID)
		if err != nil {
			domain.Error(ctx, "Failed to get friendship details", err)
			return domain.ErrDB
		}

		if friendship.FirstUserID == userID {
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
	err = s.friendStore.CreateFriendship(ctx, userID, friendID)
	if err != nil {
		domain.Error(ctx, "Failed to send friend request", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Friend request sent successfully")
	return nil
}

// AcceptFriendRequest принимает запрос в друзья
func (s *FriendService) AcceptFriendRequest(ctx context.Context, userID, friendID int) error {
	domain.Info(ctx, "Accepting friend request",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

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
func (s *FriendService) RejectFriendRequest(ctx context.Context, userID, friendID int) error {
	domain.Info(ctx, "Rejecting friend request",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

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
	err = s.friendStore.DeleteFriendship(ctx, userID, friendID)
	if err != nil {
		domain.Error(ctx, "Failed to reject friend request", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Friend request rejected successfully")
	return nil
}

// RemoveFriend удаляет из друзей
func (s *FriendService) RemoveFriend(ctx context.Context, userID, friendID int) error {
	domain.Info(ctx, "Removing friend",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

	// Проверяем что пользователи действительно друзья
	areFriends, err := s.friendStore.AreFriends(ctx, userID, friendID)
	if err != nil {
		domain.Error(ctx, "Failed to check friendship", err)
		return domain.ErrDB
	}

	if !areFriends {
		domain.Warn(ctx, "Attempt to remove non-friend")
		return domain.ErrNotFound
	}

	err = s.friendStore.DeleteFriendship(ctx, userID, friendID)
	if err != nil {
		domain.Error(ctx, "Failed to remove friend", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Friend removed successfully")
	return nil
}

// GetFriends получает список друзей
func (s *FriendService) GetFriends(ctx context.Context, userID int, params domain.PaginateQueryParams) ([]domain.ShortProfile, error) {
	// Валидация параметров
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting user friends",
		zap.Int("userID", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit))

	friends, err := s.friendStore.GetUserFriends(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get user friends", err)
		return nil, domain.ErrDB
	}

	return friends, nil
}

// GetFriendRequests получает входящие запросы в друзья
func (s *FriendService) GetFriendRequests(ctx context.Context, userID int, params domain.PaginateQueryParams) ([]domain.FriendshipWithProfile, error) {
	// Валидация параметров
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting friendship requests",
		zap.Int("userID", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit))

	requests, err := s.friendStore.GetFriendshipRequests(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get friendship requests", err)
		return nil, domain.ErrDB
	}

	return requests, nil
}

// GetSentRequests получает отправленные запросы в друзья
func (s *FriendService) GetSentRequests(ctx context.Context, userID int, params domain.PaginateQueryParams) ([]domain.FriendshipWithProfile, error) {
	// Валидация параметров
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting sent friend requests",
		zap.Int("userID", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit))

	requests, err := s.friendStore.GetSentRequests(ctx, userID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get sent requests", err)
		return nil, domain.ErrDB
	}

	return requests, nil
}

// GetFriendshipStatus получает статус дружбы с пользователем
func (s *FriendService) GetFriendshipStatus(ctx context.Context, userID, friendID int) (domain.FriendshipStatus, error) {
	domain.Info(ctx, "Getting friendship status",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

	status, err := s.friendStore.GetFriendshipStatus(ctx, userID, friendID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		domain.Error(ctx, "Failed to get friendship status", err)
		return "", domain.ErrDB
	}

	domain.Info(ctx, "Friendship status retrieved successfully")
	return status, nil
}
