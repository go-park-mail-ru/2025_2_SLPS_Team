package service

import (
	"context"
	"errors"
	"project/domain"
	"project/shared/pb"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type CommentService struct {
	commentStore   domain.CommentStore
	authService    pb.AuthServiceClient
	profileService pb.ProfileServiceClient
	postStore      domain.PostStore
}

func NewCommentService(commentStore domain.CommentStore, authService pb.AuthServiceClient, profileService pb.ProfileServiceClient, postStore domain.PostStore) domain.CommentService {
	return &CommentService{
		commentStore:   commentStore,
		authService:    authService,
		profileService: profileService,
		postStore:      postStore,
	}
}

// CreateComment создает новый комментарий
func (s *CommentService) CreateComment(ctx context.Context, userID int32, req domain.CommentCreateRequest) (*domain.CommentView, error) {
	// Валидация
	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		domain.Warn(ctx, "Comment validation failed", zap.Error(err))
		return nil, domain.ErrInvalidInput
	}

	domain.Info(ctx, "Creating comment", zap.Int32("postID", req.PostID), zap.Int32("userID", userID))

	// Проверяем существование поста
	post, err := s.postStore.GetPostByID(ctx, userID, uint(req.PostID))
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			domain.Warn(ctx, "Post not found", zap.Int32("postID", req.PostID))
			return nil, domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get post", err)
		return nil, domain.ErrDB
	}

	// Проверяем, что пост существует
	if post == nil {
		domain.Warn(ctx, "Post not found", zap.Int32("postID", req.PostID))
		return nil, domain.ErrNotFound
	}

	// Создаем объект комментария
	comment := &domain.Comment{
		PostID:   req.PostID,
		AuthorID: userID,
		Text:     req.Text,
	}

	// Сохраняем комментарий в БД
	if err := s.commentStore.CreateComment(ctx, comment); err != nil {
		domain.Error(ctx, "Failed to create comment", err)
		return nil, domain.ErrDB
	}

	// Получаем созданный комментарий для ответа
	commentView, err := s.enrichCommentWithProfile(ctx, comment)
	if err != nil {
		domain.Error(ctx, "Failed to enrich comment", err)
		return nil, err
	}

	domain.Info(ctx, "Comment created successfully",
		zap.Int32("commentID", comment.ID),
		zap.Int32("postID", req.PostID))

	return commentView, nil
}

// GetComment возвращает комментарий по ID
func (s *CommentService) GetComment(ctx context.Context, userID int32, commentID int32) (*domain.CommentView, error) {
	domain.Info(ctx, "Getting comment", zap.Int32("commentID", commentID))

	// Получаем комментарий из БД
	comment, err := s.commentStore.GetCommentByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Comment not found", zap.Int32("commentID", commentID))
			return nil, domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get comment", err)
		return nil, domain.ErrDB
	}

	// Обогащаем данными профиля
	commentView, err := s.enrichCommentWithProfile(ctx, comment)
	if err != nil {
		return nil, err
	}

	return commentView, nil
}

// GetPostComments возвращает комментарии поста с пагинацией
func (s *CommentService) GetPostComments(ctx context.Context, userID int32, postID int32, params domain.PaginateQueryParams) ([]domain.CommentView, error) {
	offset, limit := domain.ValidatePaginationParams(params)

	domain.Info(ctx, "Getting post comments", zap.Int32("postID", postID))

	// Проверяем существование поста
	_, err := s.postStore.GetPostByID(ctx, userID, uint(postID))
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			domain.Warn(ctx, "Post not found", zap.Int32("postID", postID))
			return nil, domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get post", err)
		return nil, domain.ErrDB
	}

	// Получаем комментарии из БД
	comments, err := s.commentStore.GetCommentsByPost(ctx, postID, limit, offset)
	if err != nil {
		domain.Error(ctx, "Failed to get comments", err)
		return nil, domain.ErrDB
	}

	// Обогащаем данные профилями
	commentsView, err := s.enrichCommentsWithProfiles(ctx, comments)
	if err != nil {
		return nil, err
	}

	return commentsView, nil
}

// UpdateComment обновляет комментарий
func (s *CommentService) UpdateComment(ctx context.Context, commentID int32, userID int32, text string) error {
	// Валидация
	updateRequest := domain.CommentUpdateRequest{Text: text}
	ok, err := govalidator.ValidateStruct(updateRequest)
	if !ok || err != nil {
		domain.Warn(ctx, "Comment validation failed", zap.Error(err))
		return domain.ErrInvalidInput
	}

	domain.Info(ctx, "Updating comment", zap.Int32("commentID", commentID), zap.Int32("userID", userID))

	// Получаем текущий комментарий
	existingComment, err := s.commentStore.GetCommentByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Comment not found for update", zap.Int32("commentID", commentID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get comment for update", err)
		return domain.ErrDB
	}

	// Проверяем права доступа
	if existingComment.AuthorID != userID {
		domain.Warn(ctx, "Access denied: user is not comment author",
			zap.Int32("commentID", commentID),
			zap.Int32("userID", userID),
			zap.Int32("authorID", existingComment.AuthorID))
		return domain.ErrAccessDenied
	}

	// Обновляем текст комментария
	updatedComment := &domain.Comment{
		ID:       commentID,
		AuthorID: userID,
		Text:     text,
	}

	if err := s.commentStore.UpdateComment(ctx, updatedComment); err != nil {
		domain.Error(ctx, "Failed to update comment", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Comment updated successfully")
	return nil
}

// DeleteComment удаляет комментарий
func (s *CommentService) DeleteComment(ctx context.Context, commentID int32, userID int32) error {
	domain.Info(ctx, "Deleting comment", zap.Int32("commentID", commentID), zap.Int32("userID", userID))

	// Получаем комментарий для проверки прав
	existingComment, err := s.commentStore.GetCommentByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.Warn(ctx, "Comment not found for deletion", zap.Int32("commentID", commentID))
			return domain.ErrNotFound
		}
		domain.Error(ctx, "Failed to get comment for deletion", err)
		return domain.ErrDB
	}

	// Проверяем права доступа
	if existingComment.AuthorID != userID {
		domain.Warn(ctx, "Access denied: user is not comment author",
			zap.Int32("commentID", commentID),
			zap.Int32("userID", userID),
			zap.Int32("authorID", existingComment.AuthorID))
		return domain.ErrAccessDenied
	}

	// Удаляем комментарий
	if err := s.commentStore.DeleteComment(ctx, commentID, userID); err != nil {
		domain.Error(ctx, "Failed to delete comment", err)
		return domain.ErrDB
	}

	domain.Info(ctx, "Comment deleted successfully")
	return nil
}

// GetPostCommentsCount возвращает количество комментариев поста
func (s *CommentService) GetPostCommentsCount(ctx context.Context, postID int32) (int32, error) {
	domain.Info(ctx, "Getting post comments count", zap.Int32("postID", postID))

	count, err := s.commentStore.GetPostCommentsCount(ctx, postID)
	if err != nil {
		domain.Error(ctx, "Failed to get comments count", err)
		return 0, domain.ErrDB
	}

	return count, nil
}

// Вспомогательные методы

// enrichCommentWithProfile обогащает комментарий данными профиля
func (s *CommentService) enrichCommentWithProfile(ctx context.Context, comment *domain.Comment) (*domain.CommentView, error) {
	// Получаем профиль автора через gRPC
	profilesResp, err := s.profileService.GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
		UserIDs: []int32{comment.AuthorID},
	})
	if err != nil {
		domain.Error(ctx, "Failed to get profile via gRPC", err)
		return nil, domain.ErrService
	}

	commentView := &domain.CommentView{
		ID:        comment.ID,
		PostID:    comment.PostID,
		AuthorID:  comment.AuthorID,
		ParentID:  comment.ParentID,
		Text:      comment.Text,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}

	// Заполняем данные автора из профиля
	if profile, exists := profilesResp.Profiles[comment.AuthorID]; exists {
		commentView.AuthorName = profile.FullName
		commentView.AuthorAvatar = profile.AvatarPath
	} else {
		domain.Warn(ctx, "Profile not found for user", zap.Int32("authorID", comment.AuthorID))
		commentView.AuthorName = "Пользователь"
	}

	return commentView, nil
}

// enrichCommentsWithProfiles обогащает список комментариев данными профилей
func (s *CommentService) enrichCommentsWithProfiles(ctx context.Context, comments []domain.Comment) ([]domain.CommentView, error) {
	if len(comments) == 0 {
		return []domain.CommentView{}, nil
	}

	// Собираем ID авторов
	authorIDs := make([]int32, 0, len(comments))
	for _, comment := range comments {
		authorIDs = append(authorIDs, comment.AuthorID)
	}

	// Получаем профили через gRPC
	profilesResp, err := s.profileService.GetShortProfileMapByUserIDs(ctx, &pb.GetShortProfileMapByUserIDsRequest{
		UserIDs: authorIDs,
	})
	if err != nil {
		domain.Error(ctx, "Failed to get profiles via gRPC", err)
		return nil, domain.ErrService
	}

	commentsView := make([]domain.CommentView, 0, len(comments))
	for _, comment := range comments {
		commentView := domain.CommentView{
			ID:        comment.ID,
			PostID:    comment.PostID,
			AuthorID:  comment.AuthorID,
			ParentID:  comment.ParentID,
			Text:      comment.Text,
			CreatedAt: comment.CreatedAt,
			UpdatedAt: comment.UpdatedAt,
		}

		// Заполняем данные автора
		if profile, exists := profilesResp.Profiles[comment.AuthorID]; exists {
			commentView.AuthorName = profile.FullName
			commentView.AuthorAvatar = profile.AvatarPath
		} else {
			commentView.AuthorName = "Пользователь"
		}

		commentsView = append(commentsView, commentView)
	}

	return commentsView, nil
}
