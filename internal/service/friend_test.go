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

	t.Run("Pending request from target user", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{ID: 2}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipPending, nil)
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			FirstUserID:  1,
			SecondUserID: 2,
			ActionUserID: 2, // target user sent request
			Status:       domain.FriendshipPending,
		}, nil)
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("Pending request from action user", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{ID: 2}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipPending, nil)
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			FirstUserID:  1,
			SecondUserID: 2,
			ActionUserID: 1, // action user already sent request
			Status:       domain.FriendshipPending,
		}, nil)
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("Blocked friendship", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{ID: 2}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipBlocked, nil)
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})

	t.Run("DB error on get friendship status", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{ID: 2}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipStatus(""), errors.New("db"))
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("DB error on get friendship details", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{ID: 2}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipPending, nil)
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(nil, errors.New("db"))
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("DB error on create friendship", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 2).Return(domain.User{ID: 2}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipStatus(""), domain.ErrNotFound)
		friendStore.EXPECT().CreateFriendship(ctx, 1, 2).Return(errors.New("db"))
		err := svc.SendFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
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

	t.Run("DB error on get friendship", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(nil, errors.New("db"))
		err := svc.AcceptFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Invalid pending status - already accepted", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipAccepted,
			ActionUserID: 2,
		}, nil)
		err := svc.AcceptFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("Invalid pending status - user is sender", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: 1, // user is sender, not receiver
		}, nil)
		err := svc.AcceptFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error on update friendship status", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: 2,
		}, nil)
		friendStore.EXPECT().UpdateFriendshipStatus(ctx, 1, 2, domain.FriendshipAccepted).Return(errors.New("db"))
		err := svc.AcceptFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
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
		friendStore.EXPECT().UpdateFriendshipStatus(ctx, 1, 2, domain.FriendshipRejected).Return(nil)
		err := svc.RejectFriendRequest(ctx, 1, 2)
		assert.NoError(t, err)
	})

	t.Run("Request not found", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(nil, domain.ErrNotFound)
		err := svc.RejectFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error on get friendship", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(nil, errors.New("db"))
		err := svc.RejectFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Invalid pending status - already rejected", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipRejected,
			ActionUserID: 2,
		}, nil)
		err := svc.RejectFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("Invalid pending status - user is sender", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: 1, // user is sender, not receiver
		}, nil)
		err := svc.RejectFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error on update friendship status", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, 1, 2).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: 2,
		}, nil)
		friendStore.EXPECT().UpdateFriendshipStatus(ctx, 1, 2, domain.FriendshipRejected).Return(errors.New("db"))
		err := svc.RejectFriendRequest(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
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

	t.Run("DB error on are friends check", func(t *testing.T) {
		friendStore.EXPECT().AreFriends(ctx, 1, 2).Return(false, errors.New("db"))
		err := svc.RemoveFriend(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("DB error on delete friendship", func(t *testing.T) {
		friendStore.EXPECT().AreFriends(ctx, 1, 2).Return(true, nil)
		friendStore.EXPECT().DeleteFriendship(ctx, 1, 2).Return(errors.New("db"))
		err := svc.RemoveFriend(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
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

	t.Run("Default pagination", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, 1, 20, 0).Return([]domain.ShortProfile{{UserID: 2}}, nil)
		res, err := svc.GetFriends(ctx, 1, domain.PaginateQueryParams{})
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

func TestFriendService_GetAllUsers(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetAllUsers(ctx, 1, 10, 0).Return([]domain.ShortProfile{{UserID: 2}}, nil)
		res, err := svc.GetAllUsers(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Default pagination", func(t *testing.T) {
		friendStore.EXPECT().GetAllUsers(ctx, 1, 20, 0).Return([]domain.ShortProfile{{UserID: 2}}, nil)
		res, err := svc.GetAllUsers(ctx, 1, domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetAllUsers(ctx, 1, 10, 0).Return(nil, errors.New("db"))
		res, err := svc.GetAllUsers(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestFriendService_GetFriendRequests(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipRequests(ctx, 1, 10, 0).Return([]domain.ShortProfile{{UserID: 2}}, nil)
		res, err := svc.GetFriendRequests(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Default pagination", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipRequests(ctx, 1, 20, 0).Return([]domain.ShortProfile{{UserID: 2}}, nil)
		res, err := svc.GetFriendRequests(ctx, 1, domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipRequests(ctx, 1, 10, 0).Return(nil, errors.New("db"))
		res, err := svc.GetFriendRequests(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestFriendService_GetSentRequests(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetSentRequests(ctx, 1, 10, 0).Return([]domain.ShortProfile{{UserID: 2}}, nil)
		res, err := svc.GetSentRequests(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Default pagination", func(t *testing.T) {
		friendStore.EXPECT().GetSentRequests(ctx, 1, 20, 0).Return([]domain.ShortProfile{{UserID: 2}}, nil)
		res, err := svc.GetSentRequests(ctx, 1, domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetSentRequests(ctx, 1, 10, 0).Return(nil, errors.New("db"))
		res, err := svc.GetSentRequests(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestFriendService_GetFriendshipStatus(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success with status", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipAccepted, nil)
		status, err := svc.GetFriendshipStatus(ctx, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipAccepted, status)
	})

	t.Run("Success no friendship", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipStatus(""), nil)
		status, err := svc.GetFriendshipStatus(ctx, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipStatus(""), status)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipStatus(""), errors.New("db"))
		status, err := svc.GetFriendshipStatus(ctx, 1, 2)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Equal(t, domain.FriendshipStatus(""), status)
	})
}

func TestFriendService_CountUserRelations(t *testing.T) {
	svc, friendStore, userStore, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success accepted", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.CountAccepted).Return(5, nil)
		count, err := svc.CountUserRelations(ctx, 1, domain.CountAccepted)
		assert.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("Success pending", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.CountPending).Return(3, nil)
		count, err := svc.CountUserRelations(ctx, 1, domain.CountPending)
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("Success sent", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.CountSent).Return(2, nil)
		count, err := svc.CountUserRelations(ctx, 1, domain.CountSent)
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("Success blocked", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.CountBlocked).Return(1, nil)
		count, err := svc.CountUserRelations(ctx, 1, domain.CountBlocked)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Success rejected", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.CountRejected).Return(0, nil)
		count, err := svc.CountUserRelations(ctx, 1, domain.CountRejected)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Invalid count type", func(t *testing.T) {
		count, err := svc.CountUserRelations(ctx, 1, "invalid")
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Equal(t, 0, count)
	})

	t.Run("User not found", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{}, domain.ErrNotFound)
		count, err := svc.CountUserRelations(ctx, 1, domain.CountAccepted)
		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Equal(t, 0, count)
	})

	t.Run("DB error on get user", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{}, errors.New("db"))
		count, err := svc.CountUserRelations(ctx, 1, domain.CountAccepted)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Equal(t, 0, count)
	})

	t.Run("DB error on count relations", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.CountAccepted).Return(0, errors.New("db"))
		count, err := svc.CountUserRelations(ctx, 1, domain.CountAccepted)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Equal(t, 0, count)
	})
}

func TestFriendService_IsValidCountType(t *testing.T) {
	svc, _, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()

	// Test valid count types
	assert.True(t, svc.isValidCountType(domain.CountAccepted))
	assert.True(t, svc.isValidCountType(domain.CountPending))
	assert.True(t, svc.isValidCountType(domain.CountSent))
	assert.True(t, svc.isValidCountType(domain.CountBlocked))
	assert.True(t, svc.isValidCountType(domain.CountRejected))

	// Test invalid count types
	assert.False(t, svc.isValidCountType("invalid"))
	assert.False(t, svc.isValidCountType(""))
	assert.False(t, svc.isValidCountType("unknown"))
}

func TestFriendService_ValidatePaginationParams(t *testing.T) {
	svc, friendStore, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Pagination edge cases", func(t *testing.T) {
		// Test zero limit - should use default limit (20)
		friendStore.EXPECT().GetUserFriends(ctx, 1, 20, 0).Return([]domain.ShortProfile{}, nil)
		res, err := svc.GetFriends(ctx, 1, domain.PaginateQueryParams{Limit: 0, Page: 1})
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Test negative page - should default to page 1
		friendStore.EXPECT().GetUserFriends(ctx, 1, 10, 0).Return([]domain.ShortProfile{}, nil)
		res, err = svc.GetFriends(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: -1})
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Test large page number
		friendStore.EXPECT().GetUserFriends(ctx, 1, 10, 90).Return([]domain.ShortProfile{}, nil)
		res, err = svc.GetFriends(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 10})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestFriendService_EdgeCases(t *testing.T) {
	svc, friendStore, userStore, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Empty results", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, 1, 10, 0).Return([]domain.ShortProfile{}, nil)
		res, err := svc.GetFriends(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Multiple friends", func(t *testing.T) {
		friends := []domain.ShortProfile{
			{UserID: 2, FullName: "User 2"},
			{UserID: 3, FullName: "User 3"},
			{UserID: 4, FullName: "User 4"},
		}
		friendStore.EXPECT().GetUserFriends(ctx, 1, 10, 0).Return(friends, nil)
		res, err := svc.GetFriends(ctx, 1, domain.PaginateQueryParams{Limit: 10, Page: 1})
		assert.NoError(t, err)
		assert.Len(t, res, 3)
	})

	t.Run("Friendship status edge cases", func(t *testing.T) {
		// Test rejected status
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipRejected, nil)
		status, err := svc.GetFriendshipStatus(ctx, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipRejected, status)

		// Test blocked status
		friendStore.EXPECT().GetFriendshipStatus(ctx, 1, 2).Return(domain.FriendshipBlocked, nil)
		status, err = svc.GetFriendshipStatus(ctx, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipBlocked, status)
	})

	t.Run("Count relations edge cases", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.CountAccepted).Return(0, nil)
		count, err := svc.CountUserRelations(ctx, 1, domain.CountAccepted)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)

		userStore.EXPECT().GetUserByID(ctx, 1).Return(domain.User{ID: 1}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, 1, domain.CountPending).Return(100, nil)
		count, err = svc.CountUserRelations(ctx, 1, domain.CountPending)
		assert.NoError(t, err)
		assert.Equal(t, 100, count)
	})
}