package service

import (
	"context"
	"errors"
	"mime/multipart"
	"project/domain"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type ProfileService struct {
	profileStore        domain.ProfileStore
	userStore           domain.UserStore
	elasticProfileStore domain.ElasticProfileStore
}

func NewProfileService(profileStore domain.ProfileStore, userStore domain.UserStore, elasticProfileStore domain.ElasticProfileStore) domain.ProfileService {
	return &ProfileService{
		profileStore: profileStore,
		userStore:    userStore,
	}
}

func (api *ProfileService) UpdateProfile(ctx context.Context, profile domain.Profile, userID int, files []*multipart.FileHeader) error {

	ok, err := govalidator.ValidateStruct(profile)
	if !ok || err != nil {
		domain.FromContext(ctx).Warn("Profile validation failed")
		return domain.ErrInvalidInput
	}

	if len(files) == 1 {
		avatarOldPath, err := api.profileStore.GetAvatarByUserID(ctx, userID)
		if err != nil {
			domain.FromContext(ctx).Error("Failed to get old avatar path", zap.Error(err))
			return domain.ErrDB
		}

		newfilePath, err := HandleFileUpload(files, []*string{avatarOldPath})
		if err != nil {
			domain.FromContext(ctx).Error("Failed to upload new avatar", zap.Error(err))
			return domain.ErrService
		}

		err = api.profileStore.UpdateAvatar(ctx, newfilePath[0], userID)
		if err != nil {
			domain.FromContext(ctx).Error("Failed to update avatar", zap.Error(err))
			return domain.ErrDB
		}

	} else {
		domain.FromContext(ctx).Warn("Missing avatar field")
	}

	err = api.profileStore.UpdateProfile(ctx, profile, userID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to update profile", zap.Error(err))
		return domain.ErrDB
	}

	fullName := profile.FirstName + " " + profile.LastName
	err = api.elasticProfileStore.UpdateProfile(ctx, fullName, userID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to update profile index in es", zap.Error(err))
		return domain.ErrDB
	}

	domain.FromContext(ctx).Info("Profile updated successfully")
	return nil
}

func (api *ProfileService) UpdateAvatar(ctx context.Context, userID int, files []*multipart.FileHeader) error {

	if len(files) == 1 {
		avatarOldPath, err := api.profileStore.GetAvatarByUserID(ctx, userID)
		if err != nil {
			domain.FromContext(ctx).Error("Failed to get old avatar path", zap.Error(err))
			return domain.ErrDB
		}

		newfilePath, err := HandleFileUpload(files, []*string{avatarOldPath})
		if err != nil {
			domain.FromContext(ctx).Error("Failed to upload avatar", zap.Error(err))
			return domain.ErrService
		}

		err = api.profileStore.UpdateAvatar(ctx, newfilePath[0], userID)
		if err != nil {
			domain.FromContext(ctx).Error("Failed to update avatar", zap.Error(err))
			return domain.ErrDB
		}

	} else {
		domain.FromContext(ctx).Warn("Missing avatar field in request")
		return domain.ErrInvalidInput
	}

	domain.FromContext(ctx).Info("Avatar updated successfully")
	return nil
}

func (api *ProfileService) UpdateHeader(ctx context.Context, userID int, files []*multipart.FileHeader) error {
	if len(files) == 1 {
		headerOldPath, err := api.profileStore.GetHeaderByUserID(ctx, userID)
		if err != nil {
			domain.FromContext(ctx).Error("Failed to get old header path", zap.Error(err))
			return domain.ErrDB
		}

		newfilePath, err := HandleFileUpload(files, []*string{headerOldPath})
		if err != nil {
			domain.FromContext(ctx).Error("Failed to upload header", zap.Error(err))
			return domain.ErrService
		}

		err = api.profileStore.UpdateHeader(ctx, newfilePath[0], userID)
		if err != nil {
			domain.FromContext(ctx).Error("Failed to update header", zap.Error(err))
			return domain.ErrService
		}

	} else {
		domain.FromContext(ctx).Warn("Missing header field in request")
		return domain.ErrInvalidInput
	}

	domain.FromContext(ctx).Info("Header updated successfully")
	return nil
}

func (api *ProfileService) GetProfileByUserID(ctx context.Context, userID int) (*domain.Profile, error) {

	profile, err := api.profileStore.GetProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.FromContext(ctx).Warn("User not found", zap.Int("userID", userID))
			return nil, domain.ErrNotFound
		}
		domain.FromContext(ctx).Error("Failed to get profile", zap.Error(err), zap.Int("userID", userID))
		return nil, domain.ErrDB
	}

	domain.FromContext(ctx).Info("return profile successfully")
	return &profile, nil
}

func (api *ProfileService) DeleteAvatarByUserID(ctx context.Context, userID int) error {
	avatar_path, err := api.profileStore.DeleteAvatarByUserID(ctx, userID)
	if err != nil {
		domain.FromContext(ctx).Error("Fail to delete avatar path", zap.Error(err))
		return domain.ErrDB
	}

	err = DeleteFile(*avatar_path)
	if err != nil {
		domain.FromContext(ctx).Error("Fail to delete avatar file", zap.Error(err))
		return domain.ErrDB
	}

	return nil
}
