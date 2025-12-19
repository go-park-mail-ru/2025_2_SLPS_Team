package service

import (
	"context"
	"errors"
	"project/domain"
	"project/internal/repository/mocks"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newProfileServiceMocks(t *testing.T) (*ProfileService,
	*mocks.MockProfileStore,
	*mocks.MockFriendStore,
	*mocks.MockElasticProfileStore,
	*gomock.Controller) {

	ctrl := gomock.NewController(t)
	profileStore := mocks.NewMockProfileStore(ctrl)
	friendStore := mocks.NewMockFriendStore(ctrl)
	elasticProfileStore := mocks.NewMockElasticProfileStore(ctrl)

	svc := &ProfileService{
		profileStore:        profileStore,
		friendStore:         friendStore,
		elasticProfileStore: elasticProfileStore,
	}
	return svc, profileStore, friendStore, elasticProfileStore, ctrl
}

func TestProfileService_CreateProfile(t *testing.T) {
	svc, profileStore, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		profile := domain.Profile{
			UserID:    int32(1),
			FirstName: "Doe",
			LastName:  "Doe",
			Gender:    "male",
			Dob:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		profileStore.EXPECT().CreateProfile(ctx, profile).Return(nil)

		err := svc.CreateProfile(ctx, profile)
		assert.NoError(t, err)
	})

	t.Run("DB error", func(t *testing.T) {
		profile := domain.Profile{
			UserID:    int32(1),
			FirstName: "John",
			LastName:  "Doe",
			Gender:    "male",
			Dob:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		profileStore.EXPECT().CreateProfile(ctx, profile).Return(
			errors.New("db error"),
		)

		err := svc.CreateProfile(ctx, profile)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestProfileService_UpdateProfile(t *testing.T) {
	svc, profileStore, _, elasticProfileStore, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success without avatar", func(t *testing.T) {
		userID := int32(1)
		profile := domain.Profile{
			UserID:    userID,
			FirstName: "John",
			LastName:  "Doe",
			Gender:    "male",
			Dob:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		profileStore.EXPECT().UpdateProfile(ctx, profile, userID).Return(nil)

		elasticProfileStore.EXPECT().UpdateProfile(ctx, "John Doe", userID).Return(nil)

		err := svc.UpdateProfile(ctx, profile, userID, nil)
		assert.NoError(t, err)
	})

	t.Run("DB error on profile update", func(t *testing.T) {
		userID := int32(1)
		profile := domain.Profile{
			UserID:    userID,
			FirstName: "John",
			LastName:  "Doe",
			Gender:    "male",
			Dob:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		profileStore.EXPECT().UpdateProfile(ctx, profile, userID).Return(
			errors.New("db error"),
		)

		err := svc.UpdateProfile(ctx, profile, userID, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("ElasticSearch update error", func(t *testing.T) {
		userID := int32(1)
		profile := domain.Profile{
			UserID:    userID,
			FirstName: "John",
			LastName:  "Doe",
			Gender:    "male",
			Dob:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		profileStore.EXPECT().UpdateProfile(ctx, profile, userID).Return(nil)

		elasticProfileStore.EXPECT().UpdateProfile(ctx, "John Doe", userID).Return(
			errors.New("elastic error"),
		)

		err := svc.UpdateProfile(ctx, profile, userID, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestProfileService_UpdateAvatar(t *testing.T) {
	svc, _, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Missing files", func(t *testing.T) {
		userID := int32(1)

		err := svc.UpdateAvatar(ctx, userID, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("Empty files array", func(t *testing.T) {
		userID := int32(1)

		err := svc.UpdateAvatar(ctx, userID, []*domain.File{})
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

}

func TestProfileService_UpdateHeader(t *testing.T) {
	svc, _, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Missing files", func(t *testing.T) {
		userID := int32(1)

		err := svc.UpdateHeader(ctx, userID, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("Empty files array", func(t *testing.T) {
		userID := int32(1)

		err := svc.UpdateHeader(ctx, userID, []*domain.File{})
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})
}

func TestProfileService_GetProfileByUserID(t *testing.T) {
	svc, profileStore, friendStore, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := int32(2)

		profile := domain.Profile{
			UserID:    targetUserID,
			FirstName: "John",
			LastName:  "Doe",
			Gender:    "male",
			Dob:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		relationsCount := &domain.UserRelationsCounts{
			Accepted: int32(5),
			Pending:  int32(3),
			Sent:     int32(2),
			Blocked:  int32(1),
		}

		status := domain.FriendshipAccepted

		profileStore.EXPECT().GetProfileByUserID(ctx, targetUserID).Return(profile, nil)
		friendStore.EXPECT().CountUserRelations(ctx, targetUserID).Return(relationsCount, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, selfUserID, targetUserID).Return(status, nil)

		result, err := svc.GetProfileByUserID(ctx, selfUserID, targetUserID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, targetUserID, result.UserID)
		assert.Equal(t, int32(5), result.RelationsCount.Accepted)
		assert.Equal(t, status, result.RelationStatus)
	})

	t.Run("Profile not found", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := int32(999)

		profileStore.EXPECT().GetProfileByUserID(ctx, targetUserID).Return(
			domain.Profile{},
			domain.ErrNotFound,
		)

		result, err := svc.GetProfileByUserID(ctx, selfUserID, targetUserID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error on get profile", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := int32(2)

		profileStore.EXPECT().GetProfileByUserID(ctx, targetUserID).Return(
			domain.Profile{},
			errors.New("db error"),
		)

		result, err := svc.GetProfileByUserID(ctx, selfUserID, targetUserID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("DB error on count relations", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := int32(2)

		profile := domain.Profile{UserID: targetUserID}

		profileStore.EXPECT().GetProfileByUserID(ctx, targetUserID).Return(profile, nil)
		friendStore.EXPECT().CountUserRelations(ctx, targetUserID).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetProfileByUserID(ctx, selfUserID, targetUserID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("DB error on get friendship status", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := int32(2)

		profile := domain.Profile{UserID: targetUserID}
		relationsCount := &domain.UserRelationsCounts{}

		profileStore.EXPECT().GetProfileByUserID(ctx, targetUserID).Return(profile, nil)
		friendStore.EXPECT().CountUserRelations(ctx, targetUserID).Return(relationsCount, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, selfUserID, targetUserID).Return(
			domain.FriendshipStatus(""),
			errors.New("db error"),
		)

		result, err := svc.GetProfileByUserID(ctx, selfUserID, targetUserID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestProfileService_DeleteAvatarByUserID(t *testing.T) {
	svc, profileStore, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		avatarPath := "/avatars/1.jpg"

		profileStore.EXPECT().DeleteAvatarByUserID(ctx, userID).Return(
			&avatarPath,
			nil,
		)

		err := svc.DeleteAvatarByUserID(ctx, userID)
		assert.NoError(t, err)
	})

	t.Run("DB error on delete", func(t *testing.T) {
		userID := int32(1)

		profileStore.EXPECT().DeleteAvatarByUserID(ctx, userID).Return(
			nil,
			errors.New("db error"),
		)

		err := svc.DeleteAvatarByUserID(ctx, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})

}

func TestProfileService_GetShortProfileMapByUserIDs(t *testing.T) {
	svc, profileStore, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userIDs := []int32{1, 2, 3}

		expectedProfiles := map[int32]domain.ShortProfile{
			1: {
				UserID:     1,
				FullName:   "John Doe",
				AvatarPath: stringPtr("/avatar1.jpg"),
				Dob:        time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			2: {
				UserID:     2,
				FullName:   "Jane Doe",
				AvatarPath: stringPtr("/avatar2.jpg"),
				Dob:        time.Date(1992, 2, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		profileStore.EXPECT().GetShortProfileMapByUserIDs(ctx, userIDs).Return(
			expectedProfiles,
			nil,
		)

		result, err := svc.GetShortProfileMapByUserIDs(ctx, userIDs)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "John Doe", result[1].FullName)
	})

	t.Run("Users not found", func(t *testing.T) {
		userIDs := []int32{999}

		profileStore.EXPECT().GetShortProfileMapByUserIDs(ctx, userIDs).Return(
			nil,
			domain.ErrNotFound,
		)

		result, err := svc.GetShortProfileMapByUserIDs(ctx, userIDs)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error", func(t *testing.T) {
		userIDs := []int32{1, 2}

		profileStore.EXPECT().GetShortProfileMapByUserIDs(ctx, userIDs).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetShortProfileMapByUserIDs(ctx, userIDs)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Empty userIDs", func(t *testing.T) {
		userIDs := []int32{}

		profileStore.EXPECT().GetShortProfileMapByUserIDs(ctx, userIDs).Return(
			map[int32]domain.ShortProfile{},
			nil,
		)

		result, err := svc.GetShortProfileMapByUserIDs(ctx, userIDs)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestProfileService_GetShortProfileByUserIDs(t *testing.T) {
	svc, profileStore, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userIDs := []int32{1, 2}

		expectedProfiles := []domain.ShortProfile{
			{
				UserID:     1,
				FullName:   "John Doe",
				AvatarPath: stringPtr("/avatar1.jpg"),
				Dob:        time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				UserID:     2,
				FullName:   "Jane Doe",
				AvatarPath: stringPtr("/avatar2.jpg"),
				Dob:        time.Date(1992, 2, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		profileStore.EXPECT().GetShortProfileByUserIDs(ctx, userIDs).Return(
			expectedProfiles,
			nil,
		)

		result, err := svc.GetShortProfileByUserIDs(ctx, userIDs)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int32(1), result[0].UserID)
	})

	t.Run("Users not found", func(t *testing.T) {
		userIDs := []int32{999}

		profileStore.EXPECT().GetShortProfileByUserIDs(ctx, userIDs).Return(
			nil,
			domain.ErrNotFound,
		)

		result, err := svc.GetShortProfileByUserIDs(ctx, userIDs)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error", func(t *testing.T) {
		userIDs := []int32{1}

		profileStore.EXPECT().GetShortProfileByUserIDs(ctx, userIDs).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetShortProfileByUserIDs(ctx, userIDs)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestProfileService_GetOtherShortProfileByUserIDs(t *testing.T) {
	svc, profileStore, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success with pagination", func(t *testing.T) {
		userIDs := []int32{1, 2, 3}
		limit := int32(10)
		offset := int32(0)

		expectedProfiles := []domain.ShortProfile{
			{
				UserID:     1,
				FullName:   "John Doe",
				AvatarPath: stringPtr("/avatar1.jpg"),
				Dob:        time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		}

		profileStore.EXPECT().GetOtherShortProfileByUserIDs(ctx, userIDs, limit, offset).Return(
			expectedProfiles,
			nil,
		)

		result, err := svc.GetOtherShortProfileByUserIDs(ctx, userIDs, limit, offset)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("Users not found", func(t *testing.T) {
		userIDs := []int32{999}
		limit := int32(10)
		offset := int32(0)

		profileStore.EXPECT().GetOtherShortProfileByUserIDs(ctx, userIDs, limit, offset).Return(
			nil,
			domain.ErrNotFound,
		)

		result, err := svc.GetOtherShortProfileByUserIDs(ctx, userIDs, limit, offset)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error", func(t *testing.T) {
		userIDs := []int32{1}
		limit := int32(10)
		offset := int32(0)

		profileStore.EXPECT().GetOtherShortProfileByUserIDs(ctx, userIDs, limit, offset).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetOtherShortProfileByUserIDs(ctx, userIDs, limit, offset)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func stringPtr(s string) *string {
	return &s
}
