package service

import (
	"context"
	"errors"
	"project/domain"
	repo_mocks "project/internal/repository/mocks"
	grpc_mocks "project/internal/service/mocks" // gRPC моки
	pb "project/shared/pb"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newPostServiceMocks(t *testing.T) (*PostService,
	*repo_mocks.MockPostStore,
	*repo_mocks.MockCommunityStore,
	*grpc_mocks.MockAuthServiceClient,
	*grpc_mocks.MockProfileServiceClient,
	*gomock.Controller) {

	ctrl := gomock.NewController(t)
	postStore := repo_mocks.NewMockPostStore(ctrl)
	communityStore := repo_mocks.NewMockCommunityStore(ctrl)
	authService := grpc_mocks.NewMockAuthServiceClient(ctrl)
	profileService := grpc_mocks.NewMockProfileServiceClient(ctrl)

	svc := &PostService{
		postStore:      postStore,
		authService:    authService,
		communityStore: communityStore,
		profileService: profileService,
	}
	return svc, postStore, communityStore, authService, profileService, ctrl
}

var avatar = "123"

func TestPostService_CreatePost(t *testing.T) {
	svc, postStore, communityStore, _, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success personal post", func(t *testing.T) {
		userID := int32(1)
		text := "Hello world"

		postStore.EXPECT().CreatePost(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, post *domain.Post) error {
				assert.Equal(t, uint(userID), post.AuthorID)
				assert.Equal(t, text, post.Text)
				assert.Nil(t, post.CommunityID)
				return nil
			},
		)

		result, err := svc.CreatePost(ctx, userID, text, nil, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, text, result.Text)
	})

	t.Run("Success community post", func(t *testing.T) {
		userID := int32(1)
		communityID := int32(10)
		text := "Community post"

		communityStore.EXPECT().GetCommunityByID(ctx, communityID).Return(
			&domain.Community{
				ID:        communityID,
				CreatorID: userID, // Пользователь создатель
			},
			nil,
		)

		postStore.EXPECT().CreatePost(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, post *domain.Post) error {
				assert.Equal(t, uint(userID), post.AuthorID)
				assert.Equal(t, text, post.Text)
				assert.Equal(t, &communityID, post.CommunityID)
				return nil
			},
		)

		result, err := svc.CreatePost(ctx, userID, text, &communityID, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Community not found", func(t *testing.T) {
		userID := int32(1)
		communityID := int32(999)
		text := "Test post"

		communityStore.EXPECT().GetCommunityByID(ctx, communityID).Return(
			nil,
			domain.ErrNotFound,
		)

		result, err := svc.CreatePost(ctx, userID, text, &communityID, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("Not community creator", func(t *testing.T) {
		userID := int32(1)
		communityID := int32(10)
		text := "Test post"

		communityStore.EXPECT().GetCommunityByID(ctx, communityID).Return(
			&domain.Community{
				ID:        communityID,
				CreatorID: int32(999), // Другой создатель
			},
			nil,
		)

		result, err := svc.CreatePost(ctx, userID, text, &communityID, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrAccessDenied, err)
	})

	t.Run("DB error on create", func(t *testing.T) {
		userID := int32(1)
		text := "Hello world"

		postStore.EXPECT().CreatePost(ctx, gomock.Any()).Return(
			errors.New("db error"),
		)

		result, err := svc.CreatePost(ctx, userID, text, nil, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestPostService_GetPost(t *testing.T) {
	svc, postStore, _, _, profileService, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)
		authorID := uint(2)

		postDB := &domain.PostDB{
			ID:       postID,
			AuthorID: authorID,
			Text:     "Test post",
		}

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			postDB,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
			UserIDs: []int32{int32(authorID)},
		}).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(authorID): {
						UserID:     int32(authorID),
						FullName:   "John Doe",
						AvatarPath: &avatar,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.GetPost(ctx, userID, postID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, postID, result.ID)
		assert.Equal(t, "John Doe", result.AuthorName)
	})

	t.Run("Post not found", func(t *testing.T) {
		userID := int32(1)
		postID := uint(999)

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			nil,
			domain.ErrPostNotFound,
		)

		result, err := svc.GetPost(ctx, userID, postID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrPostNotFound, err)
	})

	t.Run("DB error", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetPost(ctx, userID, postID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Profile service error", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)
		authorID := uint(2)

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			&domain.PostDB{
				ID:       postID,
				AuthorID: authorID,
				Text:     "Test post",
			},
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.GetPost(ctx, userID, postID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrService, err)
	})
}

func TestPostService_UpdatePost(t *testing.T) {
	svc, postStore, _, _, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success text update", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)
		newText := "Updated text"

		existingPost := &domain.PostDB{
			ID:          postID,
			AuthorID:    uint(userID),
			Text:        "Old text",
			Attachments: []string{},
			Photos:      []string{},
		}

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			existingPost,
			nil,
		)

		postStore.EXPECT().UpdatePost(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, post *domain.Post) error {
				assert.Equal(t, postID, post.ID)
				assert.Equal(t, newText, post.Text)
				return nil
			},
		)

		err := svc.UpdatePost(ctx, postID, userID, newText, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("Post not found", func(t *testing.T) {
		userID := int32(1)
		postID := uint(999)

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			nil,
			domain.ErrPostNotFound,
		)

		err := svc.UpdatePost(ctx, postID, userID, "text", nil, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrPostNotFound, err)
	})

	t.Run("Access denied - not author", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)

		existingPost := &domain.PostDB{
			ID:       postID,
			AuthorID: uint(999), // Другой автор
		}

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			existingPost,
			nil,
		)

		err := svc.UpdatePost(ctx, postID, userID, "text", nil, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrAccessDenied, err)
	})

	//t.Run("Validation failed", func(t *testing.T) {
	//    userID := int32(1)
	//    postID := uint(100)
	//
	//    existingPost := &domain.PostDB{
	//        ID:       postID,
	//        AuthorID: uint(userID),
	//        Text:     "Old text",
	//    }
	//
	//    postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
	//        existingPost,
	//        nil,
	//    )
	//
	//    // Пустой текст - валидация не пройдет
	//    err := svc.UpdatePost(ctx, postID, userID, "", nil, nil)
	//    assert.Error(t, err)
	//    assert.Equal(t, domain.ErrInvalidInput, err)
	//})

	t.Run("DB error on update", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)

		existingPost := &domain.PostDB{
			ID:          postID,
			AuthorID:    uint(userID),
			Text:        "Old text",
			Attachments: []string{},
			Photos:      []string{},
		}

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			existingPost,
			nil,
		)

		postStore.EXPECT().UpdatePost(ctx, gomock.Any()).Return(
			errors.New("db error"),
		)

		err := svc.UpdatePost(ctx, postID, userID, "new text", nil, nil)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestPostService_DeletePost(t *testing.T) {
	svc, postStore, _, _, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)

		existingPost := &domain.PostDB{
			ID:          postID,
			AuthorID:    uint(userID),
			Attachments: []string{"/file1.jpg"},
			Photos:      []string{"/photo1.jpg"},
		}

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			existingPost,
			nil,
		)

		postStore.EXPECT().DeletePost(ctx, postID, uint(userID)).Return(nil)

		err := svc.DeletePost(ctx, postID, userID)
		assert.NoError(t, err)
	})

	t.Run("Post not found", func(t *testing.T) {
		userID := int32(1)
		postID := uint(999)

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			nil,
			domain.ErrPostNotFound,
		)

		err := svc.DeletePost(ctx, postID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrPostNotFound, err)
	})

	t.Run("Access denied - not author", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)

		existingPost := &domain.PostDB{
			ID:       postID,
			AuthorID: uint(999), // Другой автор
		}

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			existingPost,
			nil,
		)

		err := svc.DeletePost(ctx, postID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrAccessDenied, err)
	})

	t.Run("DB error on delete", func(t *testing.T) {
		userID := int32(1)
		postID := uint(100)

		existingPost := &domain.PostDB{
			ID:       postID,
			AuthorID: uint(userID),
		}

		postStore.EXPECT().GetPostByID(ctx, userID, postID).Return(
			existingPost,
			nil,
		)

		postStore.EXPECT().DeletePost(ctx, postID, uint(userID)).Return(
			errors.New("db error"),
		)

		err := svc.DeletePost(ctx, postID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestPostService_GetUserPosts(t *testing.T) {
	svc, postStore, _, authService, profileService, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := uint(2)
		limit := int32(10)
		offset := int32(0)

		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(targetUserID)}).Return(
			&pb.UserExistsResponse{Exists: true},
			nil,
		)

		postsDB := []domain.PostDB{
			{
				ID:       uint(100),
				AuthorID: targetUserID,
				Text:     "Post 1",
			},
		}

		postStore.EXPECT().GetPostsByUser(ctx, selfUserID, targetUserID, limit, offset).Return(
			postsDB,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
			UserIDs: []int32{int32(targetUserID)},
		}).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(targetUserID): {
						UserID:     int32(targetUserID),
						FullName:   "John Doe",
						AvatarPath: &avatar,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.GetUserPosts(ctx, selfUserID, targetUserID, domain.PaginateQueryParams{
			Limit: limit,
			Page:  1,
		})
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "John Doe", result[0].AuthorName)
	})

	t.Run("User not exists", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := uint(999)

		authService.EXPECT().IsUserExists(ctx, &pb.UserIDRequest{UserId: int32(targetUserID)}).Return(
			&pb.UserExistsResponse{Exists: false},
			nil,
		)

		result, err := svc.GetUserPosts(ctx, selfUserID, targetUserID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotExist, err)
	})

	t.Run("DB error on auth check", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := uint(2)

		authService.EXPECT().IsUserExists(ctx, gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.GetUserPosts(ctx, selfUserID, targetUserID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("DB error on get posts", func(t *testing.T) {
		selfUserID := int32(1)
		targetUserID := uint(2)

		authService.EXPECT().IsUserExists(ctx, gomock.Any()).Return(
			&pb.UserExistsResponse{Exists: true},
			nil,
		)

		postStore.EXPECT().GetPostsByUser(ctx, selfUserID, targetUserID, gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetUserPosts(ctx, selfUserID, targetUserID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestPostService_GetCommunityPosts(t *testing.T) {
	svc, postStore, communityStore, _, profileService, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		communityID := int32(10)
		limit := int32(10)
		offset := int32(0)

		communityStore.EXPECT().GetCommunityByID(ctx, communityID).Return(
			&domain.Community{
				ID: communityID,
			},
			nil,
		)

		postsDB := []domain.PostDB{
			{
				ID:          uint(100),
				AuthorID:    uint(2),
				CommunityID: &communityID,
				Text:        "Community post",
			},
		}

		postStore.EXPECT().GetCommunityPosts(ctx, userID, communityID, limit, offset).Return(
			postsDB,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
			UserIDs: []int32{int32(2)},
		}).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(2): {
						UserID:   int32(2),
						FullName: "John Doe",
					},
				},
			},
			nil,
		)

		result, err := svc.GetCommunityPosts(ctx, userID, communityID, domain.PaginateQueryParams{
			Limit: limit,
			Page:  1,
		})
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("Community not found", func(t *testing.T) {
		userID := int32(1)
		communityID := int32(999)

		communityStore.EXPECT().GetCommunityByID(ctx, communityID).Return(
			nil,
			domain.ErrNotFound,
		)

		result, err := svc.GetCommunityPosts(ctx, userID, communityID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error on get posts", func(t *testing.T) {
		userID := int32(1)
		communityID := int32(10)

		communityStore.EXPECT().GetCommunityByID(ctx, communityID).Return(
			&domain.Community{ID: communityID},
			nil,
		)

		postStore.EXPECT().GetCommunityPosts(ctx, userID, communityID, gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetCommunityPosts(ctx, userID, communityID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestPostService_PostsPaginate(t *testing.T) {
	svc, postStore, _, _, profileService, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		limit := int32(10)
		offset := int32(0)

		postsDB := []domain.PostDB{
			{
				ID:       uint(100),
				AuthorID: uint(2),
				Text:     "Post 1",
			},
			{
				ID:       uint(101),
				AuthorID: uint(3),
				Text:     "Post 2",
			},
		}

		postStore.EXPECT().PostsPaginatedList(ctx, userID, limit, offset).Return(
			postsDB,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
			UserIDs: []int32{int32(2), int32(3)},
		}).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(2): {
						UserID:   int32(2),
						FullName: "John Doe",
					},
					int32(3): {
						UserID:   int32(3),
						FullName: "Jane Doe",
					},
				},
			},
			nil,
		)

		result, err := svc.PostsPaginate(ctx, userID, domain.PaginateQueryParams{
			Limit: limit,
			Page:  1,
		})
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("Empty result", func(t *testing.T) {
		userID := int32(1)

		postStore.EXPECT().PostsPaginatedList(ctx, userID, gomock.Any(), gomock.Any()).Return(
			[]domain.PostDB{},
			nil,
		)

		result, err := svc.PostsPaginate(ctx, userID, domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("DB error", func(t *testing.T) {
		userID := int32(1)

		postStore.EXPECT().PostsPaginatedList(ctx, userID, gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.PostsPaginate(ctx, userID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Profile service error", func(t *testing.T) {
		userID := int32(1)

		postsDB := []domain.PostDB{
			{
				ID:       uint(100),
				AuthorID: uint(2),
				Text:     "Post 1",
			},
		}

		postStore.EXPECT().PostsPaginatedList(ctx, userID, gomock.Any(), gomock.Any()).Return(
			postsDB,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.PostsPaginate(ctx, userID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrService, err)
	})
}

func TestPostService_UpdateLikeOnPostByUserID(t *testing.T) {
	svc, postStore, _, _, _, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)

		postStore.EXPECT().UpdateLikeOnPostByUserID(ctx, userID, postID).Return(nil)

		err := svc.UpdateLikeOnPostByUserID(ctx, userID, postID)
		assert.NoError(t, err)
	})

	t.Run("DB error", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)

		postStore.EXPECT().UpdateLikeOnPostByUserID(ctx, userID, postID).Return(
			errors.New("db error"),
		)

		err := svc.UpdateLikeOnPostByUserID(ctx, userID, postID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestPostService_EnrichPostsWithProfiles(t *testing.T) {
	svc, _, _, _, profileService, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success with profiles", func(t *testing.T) {
		postsDB := []domain.PostDB{
			{
				ID:       uint(1),
				AuthorID: uint(2),
				Text:     "Post 1",
			},
			{
				ID:       uint(2),
				AuthorID: uint(3),
				Text:     "Post 2",
			},
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
			UserIDs: []int32{int32(2), int32(3)},
		}).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(2): {
						UserID:     int32(2),
						FullName:   "John Doe",
						AvatarPath: &avatar,
						Dob:        timestamppb.New(time.Now()),
					},
					int32(3): {
						UserID:     int32(3),
						FullName:   "Jane Doe",
						AvatarPath: &avatar,
						Dob:        timestamppb.New(time.Now()),
					},
				},
			},
			nil,
		)

		result, err := svc.enrichPostsWithProfiles(ctx, postsDB)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "John Doe", result[0].AuthorName)
		assert.Equal(t, "Jane Doe", result[1].AuthorName)
	})

	t.Run("Empty posts", func(t *testing.T) {
		result, err := svc.enrichPostsWithProfiles(ctx, []domain.PostDB{})
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Profile not found", func(t *testing.T) {
		postsDB := []domain.PostDB{
			{
				ID:       uint(1),
				AuthorID: uint(999), // Несуществующий пользователь
				Text:     "Post 1",
			},
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{}, // Пустой мап
			},
			nil,
		)

		result, err := svc.enrichPostsWithProfiles(ctx, postsDB)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Пользователь", result[0].AuthorName) // Дефолтное значение
	})
}

func TestPostService_EnrichPostWithProfile(t *testing.T) {
	svc, _, _, _, profileService, ctrl := newPostServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		postDB := &domain.PostDB{
			ID:       uint(100),
			AuthorID: uint(2),
			Text:     "Test post",
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
			UserIDs: []int32{int32(2)},
		}).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					int32(2): {
						UserID:     int32(2),
						FullName:   "John Doe",
						AvatarPath: &avatar,
						Dob:        timestamppb.New(time.Now()),
					},
				},
			},
			nil,
		)

		result, err := svc.enrichPostWithProfile(ctx, postDB)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "John Doe", result.AuthorName)
		assert.Equal(t, "123", *result.AuthorAvatar)
	})

	t.Run("Profile not found", func(t *testing.T) {
		postDB := &domain.PostDB{
			ID:       uint(100),
			AuthorID: uint(999),
			Text:     "Test post",
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{},
			},
			nil,
		)

		result, err := svc.enrichPostWithProfile(ctx, postDB)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Пользователь", result.AuthorName)
		assert.Nil(t, result.AuthorAvatar)
	})

	t.Run("gRPC error", func(t *testing.T) {
		postDB := &domain.PostDB{
			ID:       uint(100),
			AuthorID: uint(2),
			Text:     "Test post",
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(ctx, gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.enrichPostWithProfile(ctx, postDB)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrService, err)
	})
}

func TestPostService_ConvertToPointerSlice(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		result := convertToPointerSlice([]string{})
		assert.Empty(t, result)
	})

	t.Run("With elements", func(t *testing.T) {
		input := []string{"file1.jpg", "file2.jpg"}
		result := convertToPointerSlice(input)

		assert.Len(t, result, 2)
		assert.Equal(t, "file1.jpg", *result[0])
		assert.Equal(t, "file2.jpg", *result[1])
	})
}
