package service

import (
	"context"
	"errors"
	"project/domain"
	repo_mocks "project/internal/repository/mocks"
	service_mocks "project/internal/service/mocks"
	pb "project/shared/pb"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newFriendServiceMocks(t *testing.T) (*FriendService,
	*repo_mocks.MockFriendStore,
	*service_mocks.MockAuthServiceClient,
	*service_mocks.MockProfileServiceClient,
	*service_mocks.MockElasticProfileStore,
	*gomock.Controller) {

	ctrl := gomock.NewController(t)
	friendStore := repo_mocks.NewMockFriendStore(ctrl)
	authService := service_mocks.NewMockAuthServiceClient(ctrl)
	profileService := service_mocks.NewMockProfileServiceClient(ctrl)
	elasticProfileStore := service_mocks.NewMockElasticProfileStore(ctrl)

	svc := &FriendService{
		friendStore:         friendStore,
		authService:         authService,
		profileService:      profileService,
		elasticProfileStore: elasticProfileStore,
	}
	return svc, friendStore, authService, profileService, elasticProfileStore, ctrl
}

func TestFriendService_SendFriendRequest(t *testing.T) {
	svc, friendStore, authService, _, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipStatus(""), domain.ErrNotFound)
		friendStore.EXPECT().CreateFriendship(ctx, int32(1), int32(2)).Return(nil)
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.NoError(t, err)
	})

	t.Run("Self request", func(t *testing.T) {
		err := svc.SendFriendRequest(ctx, int32(1), int32(1))
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Friend not found", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: false}, nil)
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrNotExist)
	})

	t.Run("DB error on get user", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(nil, errors.New("dbconn"))
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Already friends", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipAccepted, nil)
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("Pending request from target user", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipPending, nil)
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			FirstUserID:  int32(1),
			SecondUserID: int32(2),
			ActionUserID: int32(2), // target user sent request
			Status:       domain.FriendshipPending,
		}, nil)
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("Pending request from action user", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipPending, nil)
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			FirstUserID:  int32(1),
			SecondUserID: int32(2),
			ActionUserID: int32(1), // action user already sent request
			Status:       domain.FriendshipPending,
		}, nil)
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("Blocked friendship", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipBlocked, nil)
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})

	t.Run("DB error on get friendship status", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipStatus(""), errors.New("dbconn"))
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("DB error on get friendship details", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipPending, nil)
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(nil, errors.New("dbconn"))
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("DB error on create friendship", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(2)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipStatus(""), domain.ErrNotFound)
		friendStore.EXPECT().CreateFriendship(ctx, int32(1), int32(2)).Return(errors.New("dbconn"))
		err := svc.SendFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestFriendService_AcceptFriendRequest(t *testing.T) {
	svc, friendStore, _, _, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: int32(2),
		}, nil)
		friendStore.EXPECT().UpdateFriendshipStatus(ctx, int32(1), int32(2), domain.FriendshipAccepted).Return(nil)
		err := svc.AcceptFriendRequest(ctx, int32(1), int32(2))
		assert.NoError(t, err)
	})

	t.Run("Request not found", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(nil, domain.ErrNotFound)
		err := svc.AcceptFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error on get friendship", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(nil, errors.New("dbconn"))
		err := svc.AcceptFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Invalid pending status - already accepted", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			Status:       domain.FriendshipAccepted,
			ActionUserID: int32(2),
		}, nil)
		err := svc.AcceptFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("Invalid pending status - user is sender", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: int32(1), // user is sender, not receiver
		}, nil)
		err := svc.AcceptFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error on update friendship status", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: int32(2),
		}, nil)
		friendStore.EXPECT().UpdateFriendshipStatus(ctx, int32(1), int32(2), domain.FriendshipAccepted).Return(errors.New("dbconn"))
		err := svc.AcceptFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestFriendService_RejectFriendRequest(t *testing.T) {
	svc, friendStore, _, _, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: int32(2),
		}, nil)
		friendStore.EXPECT().UpdateFriendshipStatus(ctx, int32(1), int32(2), domain.FriendshipRejected).Return(nil)
		err := svc.RejectFriendRequest(ctx, int32(1), int32(2))
		assert.NoError(t, err)
	})

	t.Run("Request not found", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(nil, domain.ErrNotFound)
		err := svc.RejectFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error on get friendship", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(nil, errors.New("dbconn"))
		err := svc.RejectFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Invalid pending status - already rejected", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			Status:       domain.FriendshipRejected,
			ActionUserID: int32(2),
		}, nil)
		err := svc.RejectFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("Invalid pending status - user is sender", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: int32(1), // user is sender, not receiver
		}, nil)
		err := svc.RejectFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error on update friendship status", func(t *testing.T) {
		friendStore.EXPECT().GetFriendship(ctx, int32(1), int32(2)).Return(&domain.Friendship{
			Status:       domain.FriendshipPending,
			ActionUserID: int32(2),
		}, nil)
		friendStore.EXPECT().UpdateFriendshipStatus(ctx, int32(1), int32(2), domain.FriendshipRejected).Return(errors.New("dbconn"))
		err := svc.RejectFriendRequest(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestFriendService_RemoveFriend(t *testing.T) {
	svc, friendStore, _, _, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().DeleteFriendship(ctx, int32(1), int32(2)).Return(nil)
		err := svc.RemoveFriend(ctx, int32(1), int32(2))
		assert.NoError(t, err)
	})

	t.Run("DB error on delete friendship", func(t *testing.T) {
		friendStore.EXPECT().DeleteFriendship(ctx, int32(1), int32(2)).Return(errors.New("dbconn"))
		err := svc.RemoveFriend(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestFriendService_GetFriends(t *testing.T) {
	svc, friendStore, _, profileService, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(10), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Default pagination", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(20), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(10), int32(0)).Return(nil, errors.New("dbconn"))
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})

	t.Run("Profile service error", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(10), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(nil, errors.New("profile error"))
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestFriendService_GetAllUsers(t *testing.T) {
	svc, friendStore, _, profileService, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetAllUsers(ctx, int32(1)).Return([]int32{int32(2), int32(3)}, nil)
		profileService.EXPECT().GetOtherShortProfileByUserIDs(ctx, &pb.GetOtherShortProfileByUserIDsRequest{UserIDs: []int32{int32(2), int32(3), int32(1)}, Limit: int32(10), Offset: int32(0)}).Return(&pb.GetOtherShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)
		res, err := svc.GetAllUsers(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Default pagination", func(t *testing.T) {
		friendStore.EXPECT().GetAllUsers(ctx, int32(1)).Return([]int32{int32(2), int32(3)}, nil)
		profileService.EXPECT().GetOtherShortProfileByUserIDs(ctx, &pb.GetOtherShortProfileByUserIDsRequest{UserIDs: []int32{int32(2), int32(3), int32(1)}, Limit: int32(20), Offset: int32(0)}).Return(&pb.GetOtherShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)
		res, err := svc.GetAllUsers(ctx, int32(1), domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetAllUsers(ctx, int32(1)).Return(nil, errors.New("dbconn"))
		res, err := svc.GetAllUsers(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestFriendService_GetFriendRequests(t *testing.T) {
	svc, friendStore, _, profileService, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipRequests(ctx, int32(1), int32(10), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)
		res, err := svc.GetFriendRequests(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Default pagination", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipRequests(ctx, int32(1), int32(20), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)
		res, err := svc.GetFriendRequests(ctx, int32(1), domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipRequests(ctx, int32(1), int32(10), int32(0)).Return(nil, errors.New("dbconn"))
		res, err := svc.GetFriendRequests(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestFriendService_GetSentRequests(t *testing.T) {
	svc, friendStore, _, profileService, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		friendStore.EXPECT().GetSentRequests(ctx, int32(1), int32(10), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)
		res, err := svc.GetSentRequests(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Default pagination", func(t *testing.T) {
		friendStore.EXPECT().GetSentRequests(ctx, int32(1), int32(20), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)
		res, err := svc.GetSentRequests(ctx, int32(1), domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetSentRequests(ctx, int32(1), int32(10), int32(0)).Return(nil, errors.New("dbconn"))
		res, err := svc.GetSentRequests(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestFriendService_GetFriendshipStatus(t *testing.T) {
	svc, friendStore, _, _, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success with status", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipAccepted, nil)
		status, err := svc.GetFriendshipStatus(ctx, int32(1), int32(2))
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipAccepted, status)
	})

	t.Run("Success no friendship", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipStatus(""), nil)
		status, err := svc.GetFriendshipStatus(ctx, int32(1), int32(2))
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipStatus(""), status)
	})

	t.Run("DB error", func(t *testing.T) {
		friendStore.EXPECT().GetFriendshipStatus(ctx, int32(1), int32(2)).Return(domain.FriendshipStatus(""), errors.New("dbconn"))
		status, err := svc.GetFriendshipStatus(ctx, int32(1), int32(2))
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Equal(t, domain.FriendshipStatus(""), status)
	})
}

func TestFriendService_CountUserRelations(t *testing.T) {
	svc, friendStore, authService, _, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(1)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, int32(1)).Return(&domain.UserRelationsCounts{
			Accepted: int32(5),
			Pending:  int32(3),
			Sent:     int32(2),
			Blocked:  int32(1),
		}, nil)
		counts, err := svc.CountUserRelations(ctx, int32(1))
		assert.NoError(t, err)
		assert.Equal(t, int32(5), counts.Accepted)
		assert.Equal(t, int32(3), counts.Pending)
		assert.Equal(t, int32(2), counts.Sent)
		assert.Equal(t, int32(1), counts.Blocked)
	})

	t.Run("User not found", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(1)}).Return(&pb.UserExistsResponse{Exists: false}, nil)
		counts, err := svc.CountUserRelations(ctx, int32(1))
		assert.ErrorIs(t, err, domain.ErrNotExist)
		assert.Nil(t, counts)
	})

	t.Run("DB error on get user", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(1)}).Return(nil, errors.New("dbconn"))
		counts, err := svc.CountUserRelations(ctx, int32(1))
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, counts)
	})

	t.Run("DB error on count relations", func(t *testing.T) {
		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(1)}).Return(&pb.UserExistsResponse{Exists: true}, nil)
		friendStore.EXPECT().CountUserRelations(ctx, int32(1)).Return(nil, errors.New("dbconn"))
		counts, err := svc.CountUserRelations(ctx, int32(1))
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, counts)
	})
}

func TestFriendService_SearchShortProfilesByFullNameAndRelationType(t *testing.T) {
	svc, friendStore, _, profileService, elasticProfileStore, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success with terms", func(t *testing.T) {
		friendStore.EXPECT().GetUserIDsByFriendType(ctx, int32(1), domain.CountAccepted).Return([]int32{int32(2), int32(3)}, nil)
		elasticProfileStore.EXPECT().SearchUserIDsByFullNameWithFilter(ctx, "John", []int32{int32(2), int32(3)}, true, int32(10), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)

		res, err := svc.SearchShortProfilesByFullNameAndRelationType(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)}, "John", domain.CountAccepted)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Success without terms", func(t *testing.T) {
		friendStore.EXPECT().GetUserIDsByFriendType(ctx, int32(1), domain.CountNotFriends).Return([]int32{int32(2), int32(3)}, nil)
		elasticProfileStore.EXPECT().SearchUserIDsByFullNameWithFilter(ctx, "John", []int32{int32(2), int32(3)}, false, int32(10), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{{UserID: int32(2)}}}, nil)

		res, err := svc.SearchShortProfilesByFullNameAndRelationType(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)}, "John", domain.CountNotFriends)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("DB error on get user IDs", func(t *testing.T) {
		friendStore.EXPECT().GetUserIDsByFriendType(ctx, int32(1), domain.CountAccepted).Return(nil, errors.New("dbconn"))

		res, err := svc.SearchShortProfilesByFullNameAndRelationType(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)}, "John", domain.CountAccepted)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})

	t.Run("DB error on elastic search", func(t *testing.T) {
		friendStore.EXPECT().GetUserIDsByFriendType(ctx, int32(1), domain.CountAccepted).Return([]int32{int32(2), int32(3)}, nil)
		elasticProfileStore.EXPECT().SearchUserIDsByFullNameWithFilter(ctx, "John", []int32{int32(2), int32(3)}, true, int32(10), int32(0)).Return(nil, errors.New("elastic error"))

		res, err := svc.SearchShortProfilesByFullNameAndRelationType(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)}, "John", domain.CountAccepted)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})

	t.Run("DB error on get profiles", func(t *testing.T) {
		friendStore.EXPECT().GetUserIDsByFriendType(ctx, int32(1), domain.CountAccepted).Return([]int32{int32(2), int32(3)}, nil)
		elasticProfileStore.EXPECT().SearchUserIDsByFullNameWithFilter(ctx, "John", []int32{int32(2), int32(3)}, true, int32(10), int32(0)).Return([]int32{int32(2)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2)}}).Return(nil, errors.New("profile error"))

		res, err := svc.SearchShortProfilesByFullNameAndRelationType(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)}, "John", domain.CountAccepted)
		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, res)
	})
}

func TestFriendService_IsValidCountType(t *testing.T) {
	svc, _, _, _, _, ctrl := newFriendServiceMocks(t)
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

func TestFriendService_EdgeCases(t *testing.T) {
	svc, friendStore, _, profileService, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Empty results", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(10), int32(0)).Return([]int32{}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{}}, nil)
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Multiple friends", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(10), int32(0)).Return([]int32{int32(2), int32(3), int32(4)}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{int32(2), int32(3), int32(4)}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{
			{UserID: int32(2)}, {UserID: int32(3)}, {UserID: int32(4)},
		}}, nil)
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(1)})
		assert.NoError(t, err)
		assert.Len(t, res, 3)
	})
}

func TestFriendService_PaginationEdgeCases(t *testing.T) {
	svc, friendStore, _, profileService, _, ctrl := newFriendServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Zero limit - should use default limit (20)", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(20), int32(0)).Return([]int32{}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{}}, nil)
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(0), Page: int32(1)})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Negative page - should default to page 1", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(10), int32(0)).Return([]int32{}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{}}, nil)
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(-1)})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Large page number", func(t *testing.T) {
		friendStore.EXPECT().GetUserFriends(ctx, int32(1), int32(10), int32(90)).Return([]int32{}, nil)
		profileService.EXPECT().GetShortProfileByUserIDs(ctx, &pb.GetShortProfileByUserIDsRequest{UserIDs: []int32{}}).Return(&pb.GetShortProfileByUserIDsResponse{Profiles: []*pb.ShortProfile{}}, nil)
		res, err := svc.GetFriends(ctx, int32(1), domain.PaginateQueryParams{Limit: int32(10), Page: int32(10)})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})
}
