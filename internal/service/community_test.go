package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"project/domain"
	"project/internal/service/mocks"
	"project/shared/pb"
)

func TestNewCommunityService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockPostStore := mocks.NewMockPostStore(ctrl)
	mockAuthService := mocks.NewMockAuthServiceClient(ctrl)
	mockElasticStore := mocks.NewMockElasticCommunityStore(ctrl)
	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)

	service := NewCommunityService(
		mockCommunityStore,
		mockPostStore,
		mockAuthService,
		mockElasticStore,
		mockProfileService,
	)

	assert.NotNil(t, service)
}

func TestCommunityService_CreateCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockAuthService := mocks.NewMockAuthServiceClient(ctrl)
	mockElasticStore := mocks.NewMockElasticCommunityStore(ctrl)
	service := NewCommunityService(
		mockCommunityStore,
		nil, // postStore
		mockAuthService,
		mockElasticStore,
		nil, // profileService
	)

	ctx := context.Background()
	userID := int32(1)
	req := domain.CommunityRequest{
		Name:        "Test Community",
		Description: "Test Description",
	}

	t.Run("Success without files", func(t *testing.T) {
		expectedCommunity := &domain.Community{
			ID:          1,
			Name:        req.Name,
			Description: req.Description,
			CreatorID:   userID,
			CreatedAt:   time.Now(),
		}

		mockCommunityStore.EXPECT().
			CreateCommunity(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, c *domain.Community) error {
				c.ID = expectedCommunity.ID
				c.CreatedAt = expectedCommunity.CreatedAt
				return nil
			})

		mockElasticStore.EXPECT().
			CreateCommunity(ctx, req.Name, int32(1)).
			Return(nil)

		mockCommunityStore.EXPECT().
			Subscribe(ctx, int32(1), userID).
			Return(nil)

		community, err := service.CreateCommunity(ctx, userID, req, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, community)
		assert.Equal(t, req.Name, community.Name)
		assert.Equal(t, req.Description, community.Description)
		assert.Equal(t, userID, community.CreatorID)
	})

	t.Run("Invalid input", func(t *testing.T) {
		invalidReq := domain.CommunityRequest{
			Name:        "", // Пустое имя
			Description: "Test",
		}

		community, err := service.CreateCommunity(ctx, userID, invalidReq, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, community)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("Store error", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			CreateCommunity(ctx, gomock.Any()).
			Return(errors.New("store error"))

		community, err := service.CreateCommunity(ctx, userID, req, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, community)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Elastic error", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			CreateCommunity(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, c *domain.Community) error {
				c.ID = 1
				return nil
			})

		mockElasticStore.EXPECT().
			CreateCommunity(ctx, req.Name, int32(1)).
			Return(errors.New("elastic error"))

		mockCommunityStore.EXPECT().
			Subscribe(ctx, int32(1), userID).
			Return(nil)

		community, err := service.CreateCommunity(ctx, userID, req, nil, nil)
		// Elastic ошибка не должна ломать создание сообщества
		assert.NoError(t, err)
		assert.NotNil(t, community)
	})

	t.Run("Subscribe error", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			CreateCommunity(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, c *domain.Community) error {
				c.ID = 1
				return nil
			})

		mockElasticStore.EXPECT().
			CreateCommunity(ctx, req.Name, int32(1)).
			Return(nil)

		mockCommunityStore.EXPECT().
			Subscribe(ctx, int32(1), userID).
			Return(errors.New("subscribe error"))

		community, err := service.CreateCommunity(ctx, userID, req, nil, nil)
		// Ошибка подписки не должна ломать создание сообщества
		assert.NoError(t, err)
		assert.NotNil(t, community)
	})
}

func TestCommunityService_UpdateCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockElasticStore := mocks.NewMockElasticCommunityStore(ctrl)
	service := NewCommunityService(
		mockCommunityStore,
		nil, nil, mockElasticStore, nil,
	)

	ctx := context.Background()
	communityID := int32(1)
	userID := int32(1)
	req := domain.CommunityRequest{
		Name:        "Updated Community",
		Description: "Updated Description",
	}

	existingCommunity := &domain.Community{
		ID:          communityID,
		Name:        "Old Community",
		Description: "Old Description",
		CreatorID:   userID,
		AvatarPath:  strPtr("old/avatar.jpg"),
		CoverPath:   strPtr("old/cover.jpg"),
	}

	t.Run("Success", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(existingCommunity, nil)

		mockCommunityStore.EXPECT().
			UpdateCommunity(ctx, gomock.Any()).
			Return(nil)

		mockElasticStore.EXPECT().
			UpdateCommunity(ctx, req.Name, communityID).
			Return(nil)

		err := service.UpdateCommunity(ctx, communityID, userID, req, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("Community not found", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(nil, domain.ErrNotFound)

		err := service.UpdateCommunity(ctx, communityID, userID, req, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("Access denied - not creator", func(t *testing.T) {
		otherUserCommunity := &domain.Community{
			ID:        communityID,
			CreatorID: 999, // Другой создатель
		}

		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(otherUserCommunity, nil)

		err := service.UpdateCommunity(ctx, communityID, userID, req, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrAccessDenied, err)
	})

	t.Run("Invalid input", func(t *testing.T) {
		invalidReq := domain.CommunityRequest{
			Name:        "", // Пустое имя
			Description: "Test",
		}

		err := service.UpdateCommunity(ctx, communityID, userID, invalidReq, nil, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})
}

func TestCommunityService_DeleteCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockElasticStore := mocks.NewMockElasticCommunityStore(ctrl)
	service := NewCommunityService(
		mockCommunityStore,
		nil, nil, mockElasticStore, nil,
	)

	ctx := context.Background()
	communityID := int32(1)
	userID := int32(1)

	existingCommunity := &domain.Community{
		ID:         communityID,
		CreatorID:  userID,
		AvatarPath: strPtr("avatar.jpg"),
		CoverPath:  strPtr("cover.jpg"),
	}

	t.Run("Success", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(existingCommunity, nil)

		mockCommunityStore.EXPECT().
			DeleteCommunity(ctx, communityID, userID).
			Return(nil)

		mockElasticStore.EXPECT().
			DeleteCommunity(ctx, communityID).
			Return(nil)

		err := service.DeleteCommunity(ctx, communityID, userID)
		assert.NoError(t, err)
	})

	t.Run("Community not found", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(nil, domain.ErrNotFound)

		err := service.DeleteCommunity(ctx, communityID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("Access denied", func(t *testing.T) {
		otherUserCommunity := &domain.Community{
			ID:        communityID,
			CreatorID: 999,
		}

		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(otherUserCommunity, nil)

		err := service.DeleteCommunity(ctx, communityID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrAccessDenied, err)
	})

	t.Run("Delete store error", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(existingCommunity, nil)

		mockCommunityStore.EXPECT().
			DeleteCommunity(ctx, communityID, userID).
			Return(errors.New("delete error"))

		err := service.DeleteCommunity(ctx, communityID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestCommunityService_GetCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, nil, nil, nil)

	ctx := context.Background()
	communityID := int32(1)
	userID := int32(1)

	community := &domain.Community{
		ID:               communityID,
		Name:             "Test Community",
		Description:      "Test Description",
		CreatorID:        2,
		AvatarPath:       strPtr("avatar.jpg"),
		CoverPath:        strPtr("cover.jpg"),
		CreatedAt:        time.Now(),
		SubscribersCount: 100,
	}

	t.Run("Success with subscription", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(community, nil)

		mockCommunityStore.EXPECT().
			IsSubscribed(ctx, communityID, userID).
			Return(true, nil)

		result, err := service.GetCommunity(ctx, userID, communityID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, community.Name, result.Name)
		assert.Equal(t, community.Description, result.Description)
		assert.True(t, result.IsSubscribed)
	})

	t.Run("Success without subscription", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(community, nil)

		mockCommunityStore.EXPECT().
			IsSubscribed(ctx, communityID, userID).
			Return(false, nil)

		result, err := service.GetCommunity(ctx, userID, communityID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsSubscribed)
	})

	t.Run("Community not found", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(nil, domain.ErrNotFound)

		result, err := service.GetCommunity(ctx, userID, communityID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("Subscription check error", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(community, nil)

		mockCommunityStore.EXPECT().
			IsSubscribed(ctx, communityID, userID).
			Return(false, errors.New("check error"))

		result, err := service.GetCommunity(ctx, userID, communityID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestCommunityService_GetUserCommunities(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, nil, nil, nil)

	ctx := context.Background()
	userID := int32(1)
	params := domain.PaginateQueryParams{
		Page:  1,
		Limit: 20,
	}

	t.Run("Success", func(t *testing.T) {
		expectedCommunities := []domain.ShortCommunity{
			{ID: 1, Name: "Community 1"},
			{ID: 2, Name: "Community 2"},
		}

		mockCommunityStore.EXPECT().
			GetUserCommunities(ctx, userID, int32(20), int32(0)).
			Return(expectedCommunities, nil)

		communities, err := service.GetUserCommunities(ctx, userID, params)
		assert.NoError(t, err)
		assert.Equal(t, expectedCommunities, communities)
	})

	t.Run("Store error", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetUserCommunities(ctx, userID, int32(20), int32(0)).
			Return(nil, errors.New("store error"))

		communities, err := service.GetUserCommunities(ctx, userID, params)
		assert.Error(t, err)
		assert.Nil(t, communities)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestCommunityService_GetOtherCommunities(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, nil, nil, nil)

	ctx := context.Background()
	userID := int32(1)
	params := domain.PaginateQueryParams{
		Page:  1,
		Limit: 20,
	}

	t.Run("Success", func(t *testing.T) {
		expectedCommunities := []domain.ShortCommunity{
			{ID: 1, Name: "Community 1"},
		}

		mockCommunityStore.EXPECT().
			GetOtherCommunities(ctx, userID, int32(20), int32(0)).
			Return(expectedCommunities, nil)

		communities, err := service.GetOtherCommunities(ctx, userID, params)
		assert.NoError(t, err)
		assert.Equal(t, expectedCommunities, communities)
	})
}

func TestCommunityService_GetUserCommunitiesByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockAuthService := mocks.NewMockAuthServiceClient(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, mockAuthService, nil, nil)

	ctx := context.Background()
	targetUserID := int32(2)
	params := domain.PaginateQueryParams{
		Page:  1,
		Limit: 20,
	}

	t.Run("Success", func(t *testing.T) {
		expectedCommunities := []domain.ShortCommunity{
			{ID: 1, Name: "Community 1"},
		}

		mockAuthService.EXPECT().
			IsUserExists(ctx, &pb.UserIDRequest{UserId: targetUserID}).
			Return(&pb.UserExistsResponse{Exists: true}, nil)

		mockCommunityStore.EXPECT().
			GetUserCommunitiesByID(ctx, targetUserID, int32(20), int32(0)).
			Return(expectedCommunities, nil)

		communities, err := service.GetUserCommunitiesByID(ctx, targetUserID, params)
		assert.NoError(t, err)
		assert.Equal(t, expectedCommunities, communities)
	})

	t.Run("User not found", func(t *testing.T) {
		mockAuthService.EXPECT().
			IsUserExists(ctx, &pb.UserIDRequest{UserId: targetUserID}).
			Return(&pb.UserExistsResponse{Exists: false}, nil)

		communities, err := service.GetUserCommunitiesByID(ctx, targetUserID, params)
		assert.Error(t, err)
		assert.Nil(t, communities)
		assert.Equal(t, domain.ErrNotExist, err)
	})

	t.Run("Auth service error", func(t *testing.T) {
		mockAuthService.EXPECT().
			IsUserExists(ctx, &pb.UserIDRequest{UserId: targetUserID}).
			Return(nil, errors.New("auth error"))

		communities, err := service.GetUserCommunitiesByID(ctx, targetUserID, params)
		assert.Error(t, err)
		assert.Nil(t, communities)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestCommunityService_GetUserSubscribedCommunityIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockAuthService := mocks.NewMockAuthServiceClient(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, mockAuthService, nil, nil)

	ctx := context.Background()
	targetUserID := int32(2)

	t.Run("Success", func(t *testing.T) {
		expectedIDs := []int32{1, 2, 3}

		mockAuthService.EXPECT().
			IsUserExists(ctx, &pb.UserIDRequest{UserId: targetUserID}).
			Return(&pb.UserExistsResponse{Exists: true}, nil)

		mockCommunityStore.EXPECT().
			GetUserSubscribedCommunityIDs(ctx, targetUserID).
			Return(expectedIDs, nil)

		ids, err := service.GetUserSubscribedCommunityIDs(ctx, targetUserID)
		assert.NoError(t, err)
		assert.Equal(t, expectedIDs, ids)
	})

	t.Run("User not found", func(t *testing.T) {
		mockAuthService.EXPECT().
			IsUserExists(ctx, &pb.UserIDRequest{UserId: targetUserID}).
			Return(&pb.UserExistsResponse{Exists: false}, nil)

		ids, err := service.GetUserSubscribedCommunityIDs(ctx, targetUserID)
		assert.Error(t, err)
		assert.Nil(t, ids)
		assert.Equal(t, domain.ErrNotExist, err)
	})
}

func TestCommunityService_GetCreatedCommunities(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, nil, nil, nil)

	ctx := context.Background()
	userID := int32(1)
	params := domain.PaginateQueryParams{
		Page:  1,
		Limit: 20,
	}

	t.Run("Success", func(t *testing.T) {
		expectedCommunities := []domain.CommunityForMyCommunity{
			{ID: 1, Name: "My Community 1"},
		}

		mockCommunityStore.EXPECT().
			GetCreatedCommunities(ctx, userID, int32(20), int32(0)).
			Return(expectedCommunities, nil)

		communities, err := service.GetCreatedCommunities(ctx, userID, params)
		assert.NoError(t, err)
		assert.Equal(t, expectedCommunities, communities)
	})
}

func TestCommunityService_GetCommunitySubscribers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, nil, nil, mockProfileService)

	ctx := context.Background()
	communityID := int32(1)
	params := domain.PaginateQueryParams{
		Page:  1,
		Limit: 20,
	}

	t.Run("Success", func(t *testing.T) {
		subscriberIDs := []int32{1, 2}
		existingCommunity := &domain.Community{ID: communityID}

		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(existingCommunity, nil)

		mockCommunityStore.EXPECT().
			GetCommunitySubscribers(ctx, communityID, int32(20), int32(0)).
			Return(subscriberIDs, nil)
		mockProfileService.EXPECT().
			GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
				UserIDs: subscriberIDs,
			}).
			Return(&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					1: {FullName: "User 1", AvatarPath: &ava},
					2: {FullName: "User 2", AvatarPath: &ava},
				},
			}, nil)

		subscribers, err := service.GetCommunitySubscribers(ctx, communityID, params)
		assert.NoError(t, err)
		assert.Len(t, subscribers, 2)
		assert.Equal(t, "User 1", subscribers[0].FullName)
	})

	t.Run("Community not found", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(nil, domain.ErrNotFound)

		subscribers, err := service.GetCommunitySubscribers(ctx, communityID, params)
		assert.Error(t, err)
		assert.Nil(t, subscribers)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("No subscribers", func(t *testing.T) {
		existingCommunity := &domain.Community{ID: communityID}

		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(existingCommunity, nil)

		mockCommunityStore.EXPECT().
			GetCommunitySubscribers(ctx, communityID, int32(20), int32(0)).
			Return([]int32{}, nil)

		subscribers, err := service.GetCommunitySubscribers(ctx, communityID, params)
		assert.NoError(t, err)
		assert.Empty(t, subscribers)
	})
}

func TestCommunityService_Subscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockAuthService := mocks.NewMockAuthServiceClient(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, mockAuthService, nil, nil)

	ctx := context.Background()
	communityID := int32(1)
	userID := int32(1)

	t.Run("Success", func(t *testing.T) {
		existingCommunity := &domain.Community{ID: communityID}

		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(existingCommunity, nil)

		mockAuthService.EXPECT().
			IsUserExists(ctx, &pb.UserIDRequest{UserId: userID}).
			Return(&pb.UserExistsResponse{Exists: true}, nil)

		mockCommunityStore.EXPECT().
			Subscribe(ctx, communityID, userID).
			Return(nil)

		err := service.Subscribe(ctx, communityID, userID)
		assert.NoError(t, err)
	})

	t.Run("Community not found", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(nil, domain.ErrNotFound)

		err := service.Subscribe(ctx, communityID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("User not found", func(t *testing.T) {
		existingCommunity := &domain.Community{ID: communityID}

		mockCommunityStore.EXPECT().
			GetCommunityByID(ctx, communityID).
			Return(existingCommunity, nil)

		mockAuthService.EXPECT().
			IsUserExists(ctx, &pb.UserIDRequest{UserId: userID}).
			Return(&pb.UserExistsResponse{Exists: false}, nil)

		err := service.Subscribe(ctx, communityID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotExist, err)
	})
}

func TestCommunityService_Unsubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, nil, nil, nil)

	ctx := context.Background()
	communityID := int32(1)
	userID := int32(1)

	t.Run("Success", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			Unsubscribe(ctx, communityID, userID).
			Return(nil)

		err := service.Unsubscribe(ctx, communityID, userID)
		assert.NoError(t, err)
	})

	t.Run("Subscription not found", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			Unsubscribe(ctx, communityID, userID).
			Return(domain.ErrNotFound)

		err := service.Unsubscribe(ctx, communityID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})
}

func TestCommunityService_SearchShortCommunityByNameAndType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCommunityStore := mocks.NewMockCommunityStore(ctrl)
	mockElasticStore := mocks.NewMockElasticCommunityStore(ctrl)
	service := NewCommunityService(mockCommunityStore, nil, nil, mockElasticStore, nil)

	ctx := context.Background()
	userID := int32(1)
	name := "test"
	params := domain.PaginateQueryParams{
		Page:  1,
		Limit: 20,
	}

	t.Run("Success with subscriber type", func(t *testing.T) {
		filterIDs := []int32{1, 2}
		foundIDs := []int32{1, 3}
		expectedCommunities := []domain.ShortCommunity{
			{ID: 1, Name: "Test Community 1"},
			{ID: 3, Name: "Test Community 3"},
		}

		mockCommunityStore.EXPECT().
			GetUserSubscribedCommunityIDs(ctx, userID).
			Return(filterIDs, nil)

		mockElasticStore.EXPECT().
			SearchCommunityIDsByName(ctx, name, filterIDs, true, int32(20), int32(0)).
			Return(foundIDs, nil)

		mockCommunityStore.EXPECT().
			GetCommunitiesByIDs(ctx, foundIDs).
			Return(expectedCommunities, nil)

		communities, err := service.SearchShortCommunityByNameAndType(
			ctx, userID, params, name, domain.Subscriber,
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedCommunities, communities)
	})

	t.Run("Success with recommended type", func(t *testing.T) {
		filterIDs := []int32{1, 2}
		foundIDs := []int32{3, 4}
		expectedCommunities := []domain.ShortCommunity{
			{ID: 3, Name: "Test Community 3"},
			{ID: 4, Name: "Test Community 4"},
		}

		mockCommunityStore.EXPECT().
			GetUserSubscribedCommunityIDs(ctx, userID).
			Return(filterIDs, nil)

		mockElasticStore.EXPECT().
			SearchCommunityIDsByName(ctx, name, filterIDs, false, int32(20), int32(0)).
			Return(foundIDs, nil)

		mockCommunityStore.EXPECT().
			GetCommunitiesByIDs(ctx, foundIDs).
			Return(expectedCommunities, nil)

		communities, err := service.SearchShortCommunityByNameAndType(
			ctx, userID, params, name, domain.Recommended,
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedCommunities, communities)
	})

	t.Run("Error getting user subscriptions", func(t *testing.T) {
		mockCommunityStore.EXPECT().
			GetUserSubscribedCommunityIDs(ctx, userID).
			Return(nil, errors.New("store error"))

		communities, err := service.SearchShortCommunityByNameAndType(
			ctx, userID, params, name, domain.Subscriber,
		)
		assert.Error(t, err)
		assert.Nil(t, communities)
		assert.Equal(t, domain.ErrDB, err)
	})
}

// Helper function
func strPtr(s string) *string {
	return &s
}
