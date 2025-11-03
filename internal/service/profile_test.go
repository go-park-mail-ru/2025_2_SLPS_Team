package service

import (
	"context"
	"errors"
	"mime/multipart"
	"project/domain"
	"project/internal/repository/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newProfileServiceMocks(t *testing.T) (*ProfileService, *mocks.MockProfileStore, *mocks.MockUserStore, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	profileStore := mocks.NewMockProfileStore(ctrl)
	userStore := mocks.NewMockUserStore(ctrl)
	svc := &ProfileService{profileStore: profileStore, userStore: userStore}
	return svc, profileStore, userStore, ctrl
}

func TestProfileService_UpdateProfile(t *testing.T) {
	svc, profileStore, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	profile := domain.Profile{FirstName: "Test"}

	t.Run("Success without files", func(t *testing.T) {
		profileStore.EXPECT().UpdateProfile(ctx, profile, userID).Return(nil)
		err := svc.UpdateProfile(ctx, profile, userID, nil)
		assert.NoError(t, err)
	})

	t.Run("GetAvatar error", func(t *testing.T) {
		files := []*multipart.FileHeader{{Filename: "avatar.png"}}
		profileStore.EXPECT().GetAvatarByUserID(ctx, userID).Return(nil, errors.New("db"))
		err := svc.UpdateProfile(ctx, profile, userID, files)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("UpdateAvatar error", func(t *testing.T) {
		files := []*multipart.FileHeader{{Filename: "avatar.png"}}
		oldPath := "old.png"
		profileStore.EXPECT().GetAvatarByUserID(ctx, userID).Return(&oldPath, nil)
		err := svc.UpdateProfile(ctx, profile, userID, files)
		assert.ErrorIs(t, err, domain.ErrService)
	})

	t.Run("UpdateProfile error", func(t *testing.T) {
		profileStore.EXPECT().UpdateProfile(ctx, profile, userID).Return(errors.New("db"))
		err := svc.UpdateProfile(ctx, profile, userID, nil)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestProfileService_UpdateAvatar(t *testing.T) {
	svc, profileStore, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	files := []*multipart.FileHeader{{Filename: "avatar.png"}}
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

	t.Run("Update error", func(t *testing.T) {
		profileStore.EXPECT().GetAvatarByUserID(ctx, userID).Return(&oldPath, nil)
		err := svc.UpdateAvatar(ctx, userID, files)
		assert.ErrorIs(t, err, domain.ErrService)
	})
}

func TestProfileService_UpdateHeader(t *testing.T) {
	svc, profileStore, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	files := []*multipart.FileHeader{{Filename: "header.png"}}
	oldPath := "old.png"

	t.Run("Success", func(t *testing.T) {
		profileStore.EXPECT().GetHeaderByUserID(ctx, userID).Return(&oldPath, nil)
		err := svc.UpdateHeader(ctx, userID, files)
		assert.ErrorIs(t, err, domain.ErrService)
	})

	t.Run("Missing file", func(t *testing.T) {
		err := svc.UpdateHeader(ctx, userID, nil)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("GetHeader error", func(t *testing.T) {
		profileStore.EXPECT().GetHeaderByUserID(ctx, userID).Return(nil, errors.New("db"))
		err := svc.UpdateHeader(ctx, userID, files)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Update error", func(t *testing.T) {
		profileStore.EXPECT().GetHeaderByUserID(ctx, userID).Return(&oldPath, nil)
		err := svc.UpdateHeader(ctx, userID, files)
		assert.ErrorIs(t, err, domain.ErrService)
	})
}

func TestProfileService_GetProfileByUserID(t *testing.T) {
	svc, profileStore, _, ctrl := newProfileServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	profile := domain.Profile{FirstName: "User"}

	t.Run("Success", func(t *testing.T) {
		profileStore.EXPECT().GetProfileByUserID(ctx, userID).Return(profile, nil)
		res, err := svc.GetProfileByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, &profile, res)
	})

	t.Run("Not found", func(t *testing.T) {
		profileStore.EXPECT().GetProfileByUserID(ctx, userID).Return(domain.Profile{}, domain.ErrNotFound)
		res, err := svc.GetProfileByUserID(ctx, userID)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error", func(t *testing.T) {
		profileStore.EXPECT().GetProfileByUserID(ctx, userID).Return(domain.Profile{}, errors.New("db"))
		res, err := svc.GetProfileByUserID(ctx, userID)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}
