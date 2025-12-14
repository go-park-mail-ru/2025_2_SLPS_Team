package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
	"time"

	"go.uber.org/zap"
)

type DBCommentStore struct {
	db *sql.DB
}

func NewDBCommentStore(db *sql.DB) domain.CommentStore {
	return &DBCommentStore{db: db}
}

// CreateComment создает новый комментарий
func (store *DBCommentStore) CreateComment(ctx context.Context, comment *domain.Comment) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "commentStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CreateComment", zap.Int32("postID", comment.PostID), zap.Int32("authorID", comment.AuthorID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		INSERT INTO comments (post_id, author_id, text)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	dblogger = dblogger.With(zap.String("query", query))
	err := store.db.QueryRowContext(ctx, query,
		comment.PostID,
		comment.AuthorID,
		comment.Text,
	).Scan(
		&comment.ID,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)

	if err != nil {
		dblogger.Error("Failed to create comment", zap.Error(err))
		return fmt.Errorf("failed to create comment: %w", err)
	}

	dblogger.Info("Comment created successfully", zap.Int32("commentID", comment.ID))
	return nil
}

// GetCommentByID возвращает комментарий по ID
func (store *DBCommentStore) GetCommentByID(ctx context.Context, id int32) (*domain.Comment, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "commentStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetCommentByID", zap.Int32("commentID", id))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT id, post_id, author_id, parent_id, text, created_at, updated_at
		FROM comments
		WHERE id = $1
	`

	dblogger = dblogger.With(zap.String("query", query))
	var comment domain.Comment
	var parentID sql.NullInt32

	err := store.db.QueryRowContext(ctx, query, id).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.AuthorID,
		&parentID,
		&comment.Text,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Comment not found")
		return nil, domain.ErrNotFound
	}

	if err != nil {
		dblogger.Error("Failed to get comment", zap.Error(err))
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	if parentID.Valid {
		comment.ParentID = &parentID.Int32
	}

	dblogger.Info("Comment retrieved successfully")
	return &comment, nil
}

// GetCommentsByPost возвращает комментарии поста с пагинацией
func (store *DBCommentStore) GetCommentsByPost(ctx context.Context, postID int32, limit, offset int32) ([]domain.Comment, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "commentStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetCommentsByPost", zap.Int32("postID", postID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT id, post_id, author_id, parent_id, text, created_at, updated_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, postID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query comments", zap.Error(err))
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	comments := []domain.Comment{}
	for rows.Next() {
		var comment domain.Comment
		var parentID sql.NullInt32

		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.AuthorID,
			&parentID,
			&comment.Text,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			dblogger.Error("Failed to scan comment", zap.Error(err))
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		if parentID.Valid {
			comment.ParentID = &parentID.Int32
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Rows iteration error", zap.Error(err))
		return nil, err
	}

	dblogger.Info("Comments retrieved successfully", zap.Int("count", len(comments)))
	return comments, nil
}

// UpdateComment обновляет комментарий
func (store *DBCommentStore) UpdateComment(ctx context.Context, comment *domain.Comment) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "commentStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateComment", zap.Int32("commentID", comment.ID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		UPDATE comments
		SET text = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND author_id = $3
		RETURNING updated_at
	`

	dblogger = dblogger.With(zap.String("query", query))
	err := store.db.QueryRowContext(ctx, query,
		comment.Text,
		comment.ID,
		comment.AuthorID,
	).Scan(&comment.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Comment not found for update")
		return domain.ErrNotFound
	}

	if err != nil {
		dblogger.Error("Failed to update comment", zap.Error(err))
		return fmt.Errorf("failed to update comment: %w", err)
	}

	dblogger.Info("Comment updated successfully")
	return nil
}

// DeleteComment удаляет комментарий
func (store *DBCommentStore) DeleteComment(ctx context.Context, id int32, authorID int32) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "commentStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start DeleteComment", zap.Int32("commentID", id), zap.Int32("authorID", authorID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `DELETE FROM comments WHERE id = $1 AND author_id = $2`
	dblogger = dblogger.With(zap.String("query", query))
	result, err := store.db.ExecContext(ctx, query, id, authorID)
	if err != nil {
		dblogger.Error("Failed to delete comment", zap.Error(err))
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		dblogger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		dblogger.Warn("Comment not found for deletion")
		return domain.ErrNotFound
	}

	dblogger.Info("Comment deleted successfully")
	return nil
}

// GetPostCommentsCount возвращает количество комментариев поста
func (store *DBCommentStore) GetPostCommentsCount(ctx context.Context, postID int32) (int32, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "commentStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetPostCommentsCount", zap.Int32("postID", postID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `SELECT COUNT(*) FROM comments WHERE post_id = $1`
	dblogger = dblogger.With(zap.String("query", query))
	var count int32
	err := store.db.QueryRowContext(ctx, query, postID).Scan(&count)
	if err != nil {
		dblogger.Error("Failed to get comments count", zap.Error(err))
		return 0, fmt.Errorf("failed to get comments count: %w", err)
	}

	dblogger.Info("Comments count retrieved successfully", zap.Int32("count", count))
	return count, nil
}