package service

import (
	"context"
	"errors"
	"project/domain"
	repo_mocks "project/internal/repository/mocks"
	grpc_mocks "project/internal/service/mocks"
	pb "project/shared/pb"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newCommentServiceMocks(t *testing.T) (*CommentService,
	*repo_mocks.MockCommentStore,
	*repo_mocks.MockPostStore,
	*grpc_mocks.MockProfileServiceClient,
	*gomock.Controller) {

	ctrl := gomock.NewController(t)
	commentStore := repo_mocks.NewMockCommentStore(ctrl)
	postStore := repo_mocks.NewMockPostStore(ctrl)
	profileService := grpc_mocks.NewMockProfileServiceClient(ctrl)

	svc := &CommentService{
		commentStore:   commentStore,
		postStore:      postStore,
		profileService: profileService,
	}
	return svc, commentStore, postStore, profileService, ctrl
}

func TestCommentService_CreateComment(t *testing.T) {
	svc, commentStore, postStore, profileService, ctrl := newCommentServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)
		text := "Test comment"

		req := domain.CommentCreateRequest{
			PostID: postID,
			Text:   text,
		}

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			&domain.PostDB{
				ID: uint(postID),
			},
			nil,
		)

		commentStore.EXPECT().CreateComment(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, comment *domain.Comment) error {
				assert.Equal(t, userID, comment.AuthorID)
				assert.Equal(t, postID, comment.PostID)
				assert.Equal(t, text, comment.Text)
				comment.ID = int32(50)
				return nil
			},
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					userID: {
						UserID:     userID,
						FullName:   "John Doe",
						AvatarPath: nil,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.CreateComment(ctx, userID, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, text, result.Text)
		assert.Equal(t, "John Doe", result.AuthorName)
	})

	t.Run("Post not found", func(t *testing.T) {
		userID := int32(1)
		postID := int32(999)
		text := "Test comment"

		req := domain.CommentCreateRequest{
			PostID: postID,
			Text:   text,
		}

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			nil,
			domain.ErrPostNotFound,
		)

		result, err := svc.CreateComment(ctx, userID, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("Post DB error", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)
		text := "Test comment"

		req := domain.CommentCreateRequest{
			PostID: postID,
			Text:   text,
		}

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.CreateComment(ctx, userID, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Validation failed - empty text", func(t *testing.T) {
		userID := int32(1)
		req := domain.CommentCreateRequest{
			PostID: 100,
			Text:   "",
		}

		result, err := svc.CreateComment(ctx, userID, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrInvalidInput, err)
	})

	t.Run("DB error on create", func(t *testing.T) {
		userID := int32(1)
		req := domain.CommentCreateRequest{
			PostID: 100,
			Text:   "Test comment",
		}

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(req.PostID)).Return(
			&domain.PostDB{ID: uint(req.PostID)},
			nil,
		)

		commentStore.EXPECT().CreateComment(gomock.Any(), gomock.Any()).Return(
			errors.New("db error"),
		)

		result, err := svc.CreateComment(ctx, userID, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Profile service error", func(t *testing.T) {
		userID := int32(1)
		req := domain.CommentCreateRequest{
			PostID: 100,
			Text:   "Test comment",
		}

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(req.PostID)).Return(
			&domain.PostDB{ID: uint(req.PostID)},
			nil,
		)

		commentStore.EXPECT().CreateComment(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, comment *domain.Comment) error {
				comment.ID = int32(50)
				return nil
			},
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.CreateComment(ctx, userID, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrService, err)
	})
}

func TestCommentService_GetComment(t *testing.T) {
	svc, commentStore, _, profileService, ctrl := newCommentServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(100)
		authorID := int32(2)

		comment := &domain.Comment{
			ID:       commentID,
			PostID:   50,
			AuthorID: authorID,
			Text:     "Test comment",
			CreatedAt: time.Now(),
		}

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			comment,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					authorID: {
						UserID:     authorID,
						FullName:   "Jane Smith",
						AvatarPath: nil,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.GetComment(ctx, userID, commentID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, commentID, result.ID)
		assert.Equal(t, "Jane Smith", result.AuthorName)
	})

	t.Run("Comment not found", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(999)

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			nil,
			domain.ErrNotFound,
		)

		result, err := svc.GetComment(ctx, userID, commentID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(100)

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetComment(ctx, userID, commentID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Profile service error", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(100)
		authorID := int32(2)

		comment := &domain.Comment{
			ID:       commentID,
			PostID:   50,
			AuthorID: authorID,
			Text:     "Test comment",
			CreatedAt: time.Now(),
		}

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			comment,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.GetComment(ctx, userID, commentID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrService, err)
	})
}

func TestCommentService_GetPostComments(t *testing.T) {
	svc, commentStore, postStore, profileService, ctrl := newCommentServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success with pagination", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)
		limit := int32(10)
		offset := int32(0)

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			&domain.PostDB{ID: uint(postID)},
			nil,
		)

		comments := []domain.Comment{
			{
				ID:       1,
				PostID:   postID,
				AuthorID: 2,
				Text:     "Comment 1",
			},
			{
				ID:       2,
				PostID:   postID,
				AuthorID: 3,
				Text:     "Comment 2",
			},
		}

		commentStore.EXPECT().GetCommentsByPost(gomock.Any(), postID, limit, offset).Return(
			comments,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					2: {
						UserID:     2,
						FullName:   "User 2",
						AvatarPath: nil,
						Dob:        timestamppb.Now(),
					},
					3: {
						UserID:     3,
						FullName:   "User 3",
						AvatarPath: nil,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.GetPostComments(ctx, userID, postID, domain.PaginateQueryParams{
			Limit: limit,
			Page:  1,
		})
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "User 2", result[0].AuthorName)
		assert.Equal(t, "User 3", result[1].AuthorName)
	})

	t.Run("Post not found", func(t *testing.T) {
		userID := int32(1)
		postID := int32(999)

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			nil,
			domain.ErrPostNotFound,
		)

		result, err := svc.GetPostComments(ctx, userID, postID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error on post check", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetPostComments(ctx, userID, postID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Empty comments", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			&domain.PostDB{ID: uint(postID)},
			nil,
		)

		commentStore.EXPECT().GetCommentsByPost(gomock.Any(), postID, gomock.Any(), gomock.Any()).Return(
			[]domain.Comment{},
			nil,
		)

		result, err := svc.GetPostComments(ctx, userID, postID, domain.PaginateQueryParams{})
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("DB error on get comments", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			&domain.PostDB{ID: uint(postID)},
			nil,
		)

		commentStore.EXPECT().GetCommentsByPost(gomock.Any(), postID, gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("db error"),
		)

		result, err := svc.GetPostComments(ctx, userID, postID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Profile service error", func(t *testing.T) {
		userID := int32(1)
		postID := int32(100)

		postStore.EXPECT().GetPostByID(gomock.Any(), userID, uint(postID)).Return(
			&domain.PostDB{ID: uint(postID)},
			nil,
		)

		comments := []domain.Comment{
			{
				ID:       1,
				PostID:   postID,
				AuthorID: 2,
				Text:     "Comment 1",
			},
		}

		commentStore.EXPECT().GetCommentsByPost(gomock.Any(), postID, gomock.Any(), gomock.Any()).Return(
			comments,
			nil,
		)

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.GetPostComments(ctx, userID, postID, domain.PaginateQueryParams{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrService, err)
	})
}



func TestCommentService_DeleteComment(t *testing.T) {
	svc, commentStore, _, _, ctrl := newCommentServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(100)

		existingComment := &domain.Comment{
			ID:       commentID,
			AuthorID: userID,
			Text:     "Comment text",
		}

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			existingComment,
			nil,
		)

		commentStore.EXPECT().DeleteComment(gomock.Any(), commentID, userID).Return(nil)

		err := svc.DeleteComment(ctx, commentID, userID)
		assert.NoError(t, err)
	})

	t.Run("Comment not found", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(999)

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			nil,
			domain.ErrNotFound,
		)

		err := svc.DeleteComment(ctx, commentID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error on get comment", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(100)

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			nil,
			errors.New("db error"),
		)

		err := svc.DeleteComment(ctx, commentID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})

	t.Run("Access denied - not author", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(100)
		authorID := int32(999)

		existingComment := &domain.Comment{
			ID:       commentID,
			AuthorID: authorID,
		}

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			existingComment,
			nil,
		)

		err := svc.DeleteComment(ctx, commentID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrAccessDenied, err)
	})

	t.Run("DB error on delete", func(t *testing.T) {
		userID := int32(1)
		commentID := int32(100)

		existingComment := &domain.Comment{
			ID:       commentID,
			AuthorID: userID,
		}

		commentStore.EXPECT().GetCommentByID(gomock.Any(), commentID).Return(
			existingComment,
			nil,
		)

		commentStore.EXPECT().DeleteComment(gomock.Any(), commentID, userID).Return(
			errors.New("db error"),
		)

		err := svc.DeleteComment(ctx, commentID, userID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestCommentService_GetPostCommentsCount(t *testing.T) {
	svc, commentStore, _, _, ctrl := newCommentServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		postID := int32(100)
		expectedCount := int32(42)

		commentStore.EXPECT().GetPostCommentsCount(gomock.Any(), postID).Return(
			expectedCount,
			nil,
		)

		count, err := svc.GetPostCommentsCount(ctx, postID)
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("DB error", func(t *testing.T) {
		postID := int32(100)

		commentStore.EXPECT().GetPostCommentsCount(gomock.Any(), postID).Return(
			int32(0),
			errors.New("db error"),
		)

		count, err := svc.GetPostCommentsCount(ctx, postID)
		assert.Error(t, err)
		assert.Equal(t, int32(0), count)
		assert.Equal(t, domain.ErrDB, err)
	})
}

func TestCommentService_EnrichCommentWithProfile(t *testing.T) {
	svc, _, _, profileService, ctrl := newCommentServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		comment := &domain.Comment{
			ID:       100,
			PostID:   50,
			AuthorID: 1,
			Text:     "Test comment",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					1: {
						UserID:     1,
						FullName:   "Test User",
						AvatarPath: nil,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.enrichCommentWithProfile(ctx, comment)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test User", result.AuthorName)
		assert.Equal(t, comment.Text, result.Text)
	})

	t.Run("Profile not found", func(t *testing.T) {
		comment := &domain.Comment{
			ID:       100,
			AuthorID: 999,
			Text:     "Test comment",
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{},
			},
			nil,
		)

		result, err := svc.enrichCommentWithProfile(ctx, comment)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Пользователь", result.AuthorName)
	})

	t.Run("gRPC error", func(t *testing.T) {
		comment := &domain.Comment{
			ID:       100,
			AuthorID: 1,
			Text:     "Test comment",
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.enrichCommentWithProfile(ctx, comment)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrService, err)
	})
}

func TestCommentService_EnrichCommentsWithProfiles(t *testing.T) {
	svc, _, _, profileService, ctrl := newCommentServiceMocks(t)
	defer ctrl.Finish()
	ctx := context.Background()

	t.Run("Success with multiple comments", func(t *testing.T) {
		comments := []domain.Comment{
			{
				ID:       1,
				AuthorID: 1,
				Text:     "Comment 1",
			},
			{
				ID:       2,
				AuthorID: 2,
				Text:     "Comment 2",
			},
			{
				ID:       3,
				AuthorID: 1,
				Text:     "Comment 3",
			},
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					1: {
						UserID:     1,
						FullName:   "User One",
						AvatarPath: nil,
						Dob:        timestamppb.Now(),
					},
					2: {
						UserID:     2,
						FullName:   "User Two",
						AvatarPath: nil,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.enrichCommentsWithProfiles(ctx, comments)
		assert.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, "User One", result[0].AuthorName)
		assert.Equal(t, "User Two", result[1].AuthorName)
		assert.Equal(t, "User One", result[2].AuthorName)
	})

	t.Run("Empty comments", func(t *testing.T) {
		result, err := svc.enrichCommentsWithProfiles(ctx, []domain.Comment{})
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Some profiles not found", func(t *testing.T) {
		comments := []domain.Comment{
			{
				ID:       1,
				AuthorID: 1,
				Text:     "Comment 1",
			},
			{
				ID:       2,
				AuthorID: 999,
				Text:     "Comment 2",
			},
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			&pb.GetShortProfileMapByUserIDsResponse{
				Profiles: map[int32]*pb.ShortProfile{
					1: {
						UserID:     1,
						FullName:   "Existing User",
						AvatarPath: nil,
						Dob:        timestamppb.Now(),
					},
				},
			},
			nil,
		)

		result, err := svc.enrichCommentsWithProfiles(ctx, comments)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Existing User", result[0].AuthorName)
		assert.Equal(t, "Пользователь", result[1].AuthorName)
	})

	t.Run("gRPC error", func(t *testing.T) {
		comments := []domain.Comment{
			{
				ID:       1,
				AuthorID: 1,
				Text:     "Comment 1",
			},
		}

		profileService.EXPECT().GetShortProfileMapByUserIDs(gomock.Any(), gomock.Any()).Return(
			nil,
			errors.New("grpc error"),
		)

		result, err := svc.enrichCommentsWithProfiles(ctx, comments)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrService, err)
	})
}