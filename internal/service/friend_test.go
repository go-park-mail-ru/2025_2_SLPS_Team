package service

import (
	"context"
	"errors"
	"project/domain"
	"project/internal/repository/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newFriendServiceMocks(t *testing.T) (*FriendService, *mocks.MockFriendStore, *mocks.MockUserStore, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	friendStore := mocks.NewMockFriendStore(ctrl)
	userStore := mocks.NewMockUserStore(ctrl)
	svc := &FriendService{
		friendStore: friendStore,
		userStore:   userStore,
	}
	return svc, friendStore, userStore, ctrl
}

func TestFriendService_SendFriendRequest(t *testing.T) {
	svc, friendStore, userStore, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	friendID := 2

	t.Run("Success", func(t *testing.T) {
		userStore.EXPECT().
			GetUserByID(ctx, friendID).
			Return(&domain.User{ID: friendID}, nil)

		friendStore.EXPECT().
			GetFriendshipStatus(ctx, userID, friendID).
			Return("", nil) // Нет существующей дружбы

		friendStore.EXPECT().
			CreateFriendship(ctx, userID, friendID).
			Return(nil)

		err := svc.SendFriendRequest(ctx, userID, friendID)

		assert.NoError(t, err)
	})

	t.Run("Self friend request", func(t *testing.T) {
		err := svc.SendFriendRequest(ctx, userID, userID)

		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Friend user not found", func(t *testing.T) {
		userStore.EXPECT().
			GetUserByID(ctx, friendID).
			Return(nil, domain.ErrNotFound)

		err := svc.SendFriendRequest(ctx, userID, friendID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("Already friends", func(t *testing.T) {
		userStore.EXPECT().
			GetUserByID(ctx, friendID).
			Return(&domain.User{ID: friendID}, nil)

		friendStore.EXPECT().
			GetFriendshipStatus(ctx, userID, friendID).
			Return(domain.FriendshipAccepted, nil)

		err := svc.SendFriendRequest(ctx, userID, friendID)

		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("Pending request exists", func(t *testing.T) {
		userStore.EXPECT().
			GetUserByID(ctx, friendID).
			Return(&domain.User{ID: friendID}, nil)

		friendStore.EXPECT().
			GetFriendshipStatus(ctx, userID, friendID).
			Return(domain.FriendshipPending, nil)

		friendship := &domain.Friendship{
			FirstUserID:  userID,
			SecondUserID: friendID,
			ActionUserID: friendID, // Другой пользователь отправил запрос
		}

		friendStore.EXPECT().
			GetFriendship(ctx, userID, friendID).
			Return(friendship, nil)

		err := svc.SendFriendRequest(ctx, userID, friendID)

		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})
}

func TestFriendService_AcceptFriendRequest(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	friendID := 2

	t.Run("Success", func(t *testing.T) {
		friendship := &domain.Friendship{
			FirstUserID:  userID,
			SecondUserID: friendID,
			ActionUserID: friendID, // Другой пользователь отправил запрос
			Status:       domain.FriendshipPending,
		}

		friendStore.EXPECT().
			GetFriendship(ctx, userID, friendID).
			Return(friendship, nil)

		friendStore.EXPECT().
			UpdateFriendshipStatus(ctx, userID, friendID, domain.FriendshipAccepted).
			Return(nil)

		err := svc.AcceptFriendRequest(ctx, userID, friendID)

		assert.NoError(t, err)
	})

	t.Run("Friendship not found", func(t *testing.T) {
		friendStore.EXPECT().
			GetFriendship(ctx, userID, friendID).
			Return(nil, domain.ErrNotFound)

		err := svc.AcceptFriendRequest(ctx, userID, friendID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("User is sender not receiver", func(t *testing.T) {
		friendship := &domain.Friendship{
			FirstUserID:  userID,
			SecondUserID: friendID,
			ActionUserID: userID, // Текущий пользователь отправил запрос
			Status:       domain.FriendshipPending,
		}

		friendStore.EXPECT().
			GetFriendship(ctx, userID, friendID).
			Return(friendship, nil)

		err := svc.AcceptFriendRequest(ctx, userID, friendID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestFriendService_RejectFriendRequest(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	friendID := 2

	t.Run("Success", func(t *testing.T) {
		friendship := &domain.Friendship{
			FirstUserID:  userID,
			SecondUserID: friendID,
			ActionUserID: friendID, // Другой пользователь отправил запрос
			Status:       domain.FriendshipPending,
		}

		friendStore.EXPECT().
			GetFriendship(ctx, userID, friendID).
			Return(friendship, nil)

		friendStore.EXPECT().
			DeleteFriendship(ctx, userID, friendID).
			Return(nil)

		err := svc.RejectFriendRequest(ctx, userID, friendID)

		assert.NoError(t, err)
	})
}

func TestFriendService_RemoveFriend(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	friendID := 2

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().
			AreFriends(ctx, userID, friendID).
			Return(true, nil)

		friendStore.EXPECT().
			DeleteFriendship(ctx, userID, friendID).
			Return(nil)

		err := svc.RemoveFriend(ctx, userID, friendID)

		assert.NoError(t, err)
	})

	t.Run("Not friends", func(t *testing.T) {
		friendStore.EXPECT().
			AreFriends(ctx, userID, friendID).
			Return(false, nil)

		err := svc.RemoveFriend(ctx, userID, friendID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestFriendService_GetFriends(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1

	t.Run("Success", func(t *testing.T) {
		friends := []domain.ShortProfile{
			{UserID: 2, FullName: "Friend One"},
			{UserID: 3, FullName: "Friend Two"},
		}
		totalPages := 1

		friendStore.EXPECT().
			GetUserFriends(ctx, userID, 1, 20).
			Return(friends, totalPages, nil)

		result, pages, err := svc.GetFriends(ctx, userID, 1, 20)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, totalPages, pages)
	})

	t.Run("Default pagination", func(t *testing.T) {
		friends := []domain.ShortProfile{}
		totalPages := 0

		friendStore.EXPECT().
			GetUserFriends(ctx, userID, 1, 20).
			Return(friends, totalPages, nil)

		result, pages, err := svc.GetFriends(ctx, userID, 0, 0)

		assert.NoError(t, err)
		assert.Len(t, result, 0)
		assert.Equal(t, totalPages, pages)
	})
}

func TestFriendService_GetFriendRequests(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1

	t.Run("Success", func(t *testing.T) {
		requests := []domain.FriendshipWithProfile{
			{
				Friendship: domain.Friendship{
					ID:           1,
					FirstUserID:  1,
					SecondUserID: 2,
					Status:       domain.FriendshipPending,
				},
				Friend: domain.ShortProfile{UserID: 2, FullName: "Requester"},
			},
		}
		totalPages := 1

		friendStore.EXPECT().
			GetFriendshipRequests(ctx, userID, 1, 20).
			Return(requests, totalPages, nil)

		result, pages, err := svc.GetFriendRequests(ctx, userID, 1, 20)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, totalPages, pages)
	})
}

func TestFriendService_GetSentRequests(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1

	t.Run("Success", func(t *testing.T) {
		requests := []domain.FriendshipWithProfile{
			{
				Friendship: domain.Friendship{
					ID:           1,
					FirstUserID:  1,
					SecondUserID: 2,
					Status:       domain.FriendshipPending,
				},
				Friend: domain.ShortProfile{UserID: 2, FullName: "Receiver"},
			},
		}
		totalPages := 1

		friendStore.EXPECT().
			GetSentRequests(ctx, userID, 1, 20).
			Return(requests, totalPages, nil)

		result, pages, err := svc.GetSentRequests(ctx, userID, 1, 20)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, totalPages, pages)
	})
}

func TestFriendService_GetFriendshipStatus(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	friendID := 2

	t.Run("Success with status", func(t *testing.T) {
		friendStore.EXPECT().
			GetFriendshipStatus(ctx, userID, friendID).
			Return(domain.FriendshipAccepted, nil)

		status, err := svc.GetFriendshipStatus(ctx, userID, friendID)

		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipAccepted, status)
	})

	t.Run("No friendship record", func(t *testing.T) {
		friendStore.EXPECT().
			GetFriendshipStatus(ctx, userID, friendID).
			Return("", domain.ErrNotFound)

		status, err := svc.GetFriendshipStatus(ctx, userID, friendID)

		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipStatus(""), status)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().
			GetFriendshipStatus(ctx, userID, friendID).
			Return("", errors.New("db error"))

		status, err := svc.GetFriendshipStatus(ctx, userID, friendID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Equal(t, domain.FriendshipStatus(""), status)
	})
}
