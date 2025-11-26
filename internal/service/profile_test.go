package service

import (
	"context"
	"errors"
	"project/domain"
	"project/internal/service/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newProfileServiceMocks(t *testing.T) (*ProfileService, *mocks.MockProfileStore, *mocks.MockFriendStore, *mocks.MockElasticProfileStore, *gomock.Controller) {
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

func TestProfileService_UpdateProfile(t *testing.T) {
	svc, profileStore, _, elasticProfileStore, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := int32(1)
	profile := domain.Profile{
		UserID:    userID,
		FirstName: "Test",
		LastName:  "User",
	}

	t.Run("Success without files", func(t *testing.T) {
		profileStore.EXPECT().UpdateProfile(ctx, profile, userID).Return(nil)
		elasticProfileStore.EXPECT().UpdateProfile(ctx, "Test User", userID).Return(nil)

		err := svc.UpdateProfile(ctx, profile, userID, nil)
		assert.NoError(t, err)
	})

	t.Run("Validation failed", func(t *testing.T) {
		invalidProfile := domain.Profile{}

		err := svc.UpdateProfile(ctx, invalidProfile, userID, nil)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("GetAvatar error when files provided", func(t *testing.T) {
		files := []*domain.File{{Filename: "avatar.png"}}
		profileStore.EXPECT().GetAvatarByUserID(ctx, userID).Return(nil, errors.New("db error"))

		err := svc.UpdateProfile(ctx, profile, userID, files)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("UpdateProfile store error", func(t *testing.T) {
		profileStore.EXPECT().UpdateProfile(ctx, profile, userID).Return(errors.New("db error"))

		err := svc.UpdateProfile(ctx, profile, userID, nil)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("ElasticSearch update error", func(t *testing.T) {
		profileStore.EXPECT().UpdateProfile(ctx, profile, userID).Return(nil)
		elasticProfileStore.EXPECT().UpdateProfile(ctx, "Test User", userID).Return(errors.New("es error"))

		err := svc.UpdateProfile(ctx, profile, userID, nil)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestProfileService_UpdateAvatar(t *testing.T) {
	svc, profileStore, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := int32(1)
	files := []*domain.File{{Filename: "avatar.png"}}
	oldPath := "old.png"

	t.Run("Missing file", func(t *testing.T) {
		err := svc.UpdateAvatar(ctx, userID, nil)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("GetAvatar error", func(t *testing.T) {
		profileStore.EXPECT().GetAvatarByUserID(ctx, userID).Return(nil, errors.New("db"))
		err := svc.UpdateAvatar(ctx, userID, files)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("File upload error", func(t *testing.T) {
		profileStore.EXPECT().GetAvatarByUserID(ctx, userID).Return(&oldPath, nil)
		err := svc.UpdateAvatar(ctx, userID, files)
		assert.ErrorIs(t, err, domain.ErrService)
	})
}

func TestProfileService_UpdateHeader(t *testing.T) {
	svc, profileStore, _, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := int32(1)
	files := []*domain.File{{Filename: "header.png"}}
	oldPath := "old.png"

	t.Run("Missing file", func(t *testing.T) {
		err := svc.UpdateHeader(ctx, userID, nil)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("GetHeader error", func(t *testing.T) {
		profileStore.EXPECT().GetHeaderByUserID(ctx, userID).Return(nil, errors.New("db"))
		err := svc.UpdateHeader(ctx, userID, files)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("File upload error", func(t *testing.T) {
		profileStore.EXPECT().GetHeaderByUserID(ctx, userID).Return(&oldPath, nil)
		err := svc.UpdateHeader(ctx, userID, files)
		assert.ErrorIs(t, err, domain.ErrService)
	})
}

func TestProfileService_GetProfileByUserID(t *testing.T) {
	svc, profileStore, friendStore, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	selfUserID := int32(1)
	targetUserID := int32(2)

	t.Run("Success", func(t *testing.T) {
		expectedProfile := domain.Profile{UserID: targetUserID}
		relationsCount := &domain.UserRelationsCounts{Friends: 10}
		status := domain.FriendshipStatusFriends

		profileStore.EXPECT().GetProfileByUserID(ctx, targetUserID).Return(expectedProfile, nil)
		friendStore.EXPECT().CountUserRelations(ctx, targetUserID).Return(relationsCount, nil)
		friendStore.EXPECT().GetFriendshipStatus(ctx, selfUserID, targetUserID).Return(status, nil)

		result, err := svc.GetProfileByUserID(ctx, selfUserID, targetUserID)

		assert.NoError(t, err)
		assert.Equal(t, targetUserID, result.UserID)
		assert.Equal(t, *relationsCount, result.RelationsCount)
		assert.Equal(t, status, result.RelationStatus)
	})

	t.Run("Profile not found", func(t *testing.T) {
		profileStore.EXPECT().GetProfileByUserID(ctx, targetUserID).Return(domain.Profile{}, domain.ErrNotFound)

		result, err := svc.GetProfileByUserID(ctx, selfUserID, targetUserID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Nil(t, result)
	})

	t.Run("Count relations error", func(t *testing.T) {
		expectedProfile := domain.Profile{UserID: targetUserID}

		profileStore.EXPECT().GetProfileByUserID(ctx, targetUserID).Return(expectedProfile, nil)
		friendStore.EXPECT().CountUserRelations(ctx, targetUserID).Return(nil, errors.New("db error"))

		result, err := svc.GetProfileByUserID(ctx, selfUserID, targetUserID)

		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, result)
	})
}
