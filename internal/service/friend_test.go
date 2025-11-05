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

func newFriendServiceMocks(t *testing.T) (*FriendService,
	*mocks.MockFriendStore,
	*mocks.MockUserStore,
	*gomock.Controller) {

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

	t.Run("Success", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{ID: 2}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipStatus(""), domain.ErrNotFound)
		friendStore.EXPECT().CreateFriendship(ctx, 1, 2).Return(nil)
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.NoError(t, err)
	})

	t.Run("Self request", func(t *testing.T) {
		err := svc.SendFriendRequest(ctx, 1, 1)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Friend not found", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{}, domain.ErrNotFound)
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error on get user", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{}, errors.New("db"))
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Already friends", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{ID: 2}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipAccepted, nil)
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})
}

func TestFriendService_AcceptFriendRequest(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: 2,
		}, nil)
		friendStore.EXPECT().UpdateFriendshipStatus(ctx, 1, 2, domain.FriendshipAccepted).Return(nil)
		err := svc.AcceptFriendRequest(ctx, 1, 2)
		assert.NoError(t, err)
	})

	t.Run("Request not found", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(nil, domain.ErrNotFound)
		err := svc.AcceptFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(nil, errors.New("db"))
		err := svc.AcceptFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Invalid pending status", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipAccepted,
			ActionUserID: 2,
		}, nil)
		err := svc.AcceptFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestFriendService_RejectFriendRequest(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: 2,
		}, nil)

		friendStore.EXPECT().
			UpdateFriendshipStatus(ctx, 1, 2, domain.FriendshipRejected).
			Return(nil)

		err := svc.RejectFriendRequest(ctx, 1, 2)
		assert.NoError(t, err)
	})

	t.Run("Request not found", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(nil, domain.ErrNotFound)
		err := svc.RejectFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestFriendService_RemoveFriend(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().AreFriends(ctx, 1, 2).Return(true, nil)
		friendStore.EXPECT().DeleteFriendship(ctx, 1, 2).Return(nil)
		err := svc.RemoveFriend(ctx, 1, 2)
		assert.NoError(t, err)
	})

	t.Run("Not friends", func(t *testing.T) {
		friendStore.EXPECT().AreFriends(ctx, 1, 2).Return(false, nil)
		err := svc.RemoveFriend(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestFriendService_GetFriends(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, 1, 10, 0).Return([]domain.ShortProfile{{UserID: 2}}, nil)
		res, err := svc.GetFriends(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, 1, 10, 0).Return(nil, errors.New("db"))
		res, err := svc.GetFriends(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

//func TestFriendService_CountUserRelations(t *testing.T) {
//	svc, friendStore, userStore, ctrl := newFriendServiceMocks(t)
//	defer ctrl.Finish()
//	ctx := context.Background()
//
//	t.Run("Success", func(t *testing.T) {
//		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
//		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.FriendshipCountType("friends")).Return(5, nil)
//
//		count, err := svc.CountUserRelations(ctx, 1, domain.FriendshipCountType("friends"))
//		assert.NoError(t, err)
//		assert.Equal(t, 5, count)
//	})
//
//	t.Run("Invalid count type", func(t *testing.T) {
//		count, err := svc.CountUserRelations(ctx, 1, "invalid")
//		assert.ErrorIs(t, err, domain.ErrInvalidInput)
//		assert.Equal(t, 0, count)
//	})
//
//	t.Run("User not found", func(t *testing.T) {
//		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{}, domain.ErrNotFound)
//		count, err := svc.CountUserRelations(ctx, 1, domain.FriendshipCountType(""))
//		assert.ErrorIs(t, err, domain.ErrNotFound)
//		assert.Equal(t, 0, count)
//	})
//}
