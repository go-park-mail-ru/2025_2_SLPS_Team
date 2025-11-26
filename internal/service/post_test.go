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

func newPostServiceMocks(t *testing.T) (*PostService, *mocks.MockPostStore, *mocks.MockUserStore, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	postStore := mocks.NewMockPostStore(ctrl)
	userStore := mocks.NewMockUserStore(ctrl)
	svc := &PostService{
		postStore: postStore,
		userStore: userStore,
	}
	return svc, postStore, userStore, ctrl
}

func TestPostService_PostsPaginate(t *testing.T) {
	svc, postStore, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		avatar := "avatar.png"
		posts := []domain.PostWithShortUser{
			{
				Post: domain.Post{
					ID:       1,
					AuthorID: 1,
					Text:     "Test post",
				},
				Author: domain.ShortProfile{
					UserID:     1,
					FullName:   "John Doe",
					AvatarPath: &avatar,
				},
			},

			{
				Post: domain.Post{
					ID:       1,
					AuthorID: 1,
					Text:     "Test post",
				},
				Author: domain.ShortProfile{
					UserID:     1,
					FullName:   "John Doe",
					AvatarPath: &avatar,
				},
			},
		}

		postStore.EXPECT().
			PostsPaginatedList(ctx, 20, 0).
			Return(posts, nil)

		result, err := svc.PostsPaginate(ctx, domain.PaginateQueryParams{
			Page:  1,
			Limit: 20,
		})

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("Default pagination", func(t *testing.T) {
		posts := []domain.PostWithShortUser{}

		postStore.EXPECT().
			PostsPaginatedList(ctx, 20, 0).
			Return(posts, nil)

		result, err := svc.PostsPaginate(ctx, domain.PaginateQueryParams{})

		assert.NoError(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("DB error", func(t *testing.T) {
		postStore.EXPECT().
			PostsPaginatedList(ctx, 20, 0).
			Return(nil, errors.New("dbconn error"))

		result, err := svc.PostsPaginate(ctx, domain.PaginateQueryParams{
			Page:  1,
			Limit: 20,
		})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestPostService_GetPost(t *testing.T) {
	svc, postStore, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	postID := uint(1)

	t.Run("Success", func(t *testing.T) {
		post := &domain.Post{ID: postID, Text: "Test post"}

		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(post, nil)

		result, err := svc.GetPost(ctx, postID)

		assert.NoError(t, err)
		assert.Equal(t, post, result)
	})

	t.Run("Post not found", func(t *testing.T) {
		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(nil, domain.ErrPostNotFound)

		result, err := svc.GetPost(ctx, postID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrPostNotFound)
	})

	t.Run("DB error", func(t *testing.T) {
		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(nil, errors.New("dbconn error"))

		result, err := svc.GetPost(ctx, postID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestPostService_CreatePost(t *testing.T) {
	svc, postStore, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1
	text := "This is a valid post text with enough length"

	t.Run("Success without files", func(t *testing.T) {
		postStore.EXPECT().
			CreatePost(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, post *domain.Post) error {
				post.ID = 1
				return nil
			})

		result, err := svc.CreatePost(ctx, userID, text, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, uint(userID), result.AuthorID)
		assert.Equal(t, text, result.Text)
	})

	t.Run("Text too short", func(t *testing.T) {
		_, err := svc.CreatePost(ctx, userID, "short", nil, nil)

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("DB error on create", func(t *testing.T) {
		postStore.EXPECT().
			CreatePost(ctx, gomock.Any()).
			Return(errors.New("dbconn error"))

		_, err := svc.CreatePost(ctx, userID, text, nil, nil)

		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestPostService_UpdatePost(t *testing.T) {
	svc, postStore, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	postID := uint(1)
	userID := 1
	text := "Updated post text with sufficient length"

	t.Run("Success", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:       postID,
			AuthorID: uint(userID),
			Text:     "Original text",
		}

		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(existingPost, nil)

		postStore.EXPECT().
			UpdatePost(ctx, gomock.Any()).
			Return(nil)

		err := svc.UpdatePost(ctx, postID, userID, text, nil, nil)

		assert.NoError(t, err)
	})

	t.Run("Not author", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:       postID,
			AuthorID: 999, // Different author
			Text:     "Original text",
		}

		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(existingPost, nil)

		err := svc.UpdatePost(ctx, postID, userID, text, nil, nil)

		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})

	t.Run("Post not found", func(t *testing.T) {
		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(nil, domain.ErrPostNotFound)

		err := svc.UpdatePost(ctx, postID, userID, text, nil, nil)

		assert.ErrorIs(t, err, domain.ErrPostNotFound)
	})

	t.Run("Text too short", func(t *testing.T) {
		err := svc.UpdatePost(ctx, postID, userID, "short", nil, nil)

		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}

func TestPostService_DeletePost(t *testing.T) {
	svc, postStore, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	postID := uint(1)
	userID := 1

	t.Run("Success", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:       postID,
			AuthorID: uint(userID),
			Text:     "Post to delete",
		}

		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(existingPost, nil)

		postStore.EXPECT().
			DeletePost(ctx, postID, uint(userID)).
			Return(nil)

		err := svc.DeletePost(ctx, postID, userID)

		assert.NoError(t, err)
	})

	t.Run("Post not found", func(t *testing.T) {
		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(nil, domain.ErrPostNotFound)

		err := svc.DeletePost(ctx, postID, userID)

		assert.ErrorIs(t, err, domain.ErrPostNotFound)
	})

	t.Run("Not author", func(t *testing.T) {
		existingPost := &domain.Post{
			ID:       postID,
			AuthorID: 999, // Different author
			Text:     "Post to delete",
		}

		postStore.EXPECT().
			GetPostByID(ctx, postID).
			Return(existingPost, nil)

		err := svc.DeletePost(ctx, postID, userID)

		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})
}

func TestPostService_GetUserPosts(t *testing.T) {
	svc, postStore, userStore, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uint(1)

	t.Run("Success", func(t *testing.T) {
		userStore.EXPECT().
			GetUserByID(ctx, int32(userID)).
			Return(domain.User{ID: int32(userID)}, nil)

		posts := []domain.Post{
			{ID: 1, AuthorID: userID, Text: "User post 1"},
		}

		postStore.EXPECT().
			GetPostsByUser(ctx, userID, 20, 0).
			Return(posts, nil)

		result, err := svc.GetUserPosts(ctx, userID, domain.PaginateQueryParams{
			Page:  1,
			Limit: 20,
		})

		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("User not found", func(t *testing.T) {
		userStore.EXPECT().
			GetUserByID(ctx, int32(userID)).
			Return(domain.User{}, domain.ErrUserNotFound)

		result, err := svc.GetUserPosts(ctx, userID, domain.PaginateQueryParams{
			Page:  1,
			Limit: 20,
		})

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, result)
	})

	t.Run("DB error on user check", func(t *testing.T) {
		userStore.EXPECT().
			GetUserByID(ctx, int32(userID)).
			Return(domain.User{}, errors.New("dbconn error"))

		result, err := svc.GetUserPosts(ctx, userID, domain.PaginateQueryParams{
			Page:  1,
			Limit: 20,
		})

		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, result)
	})

	t.Run("DB error on posts get", func(t *testing.T) {
		userStore.EXPECT().
			GetUserByID(ctx, int32(userID)).
			Return(domain.User{ID: int32(userID)}, nil)

		postStore.EXPECT().
			GetPostsByUser(ctx, userID, 20, 0).
			Return(nil, errors.New("dbconn error"))

		result, err := svc.GetUserPosts(ctx, userID, domain.PaginateQueryParams{
			Page:  1,
			Limit: 20,
		})

		assert.ErrorIs(t, err, domain.ErrDB)
		assert.Nil(t, result)
	})
}
