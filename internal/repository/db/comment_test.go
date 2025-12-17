package db

import (
	"context"
	"database/sql"
	"errors"
	"project/domain"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBCommentStore_CreateComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewDBCommentStore(db).(*DBCommentStore)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		comment := &domain.Comment{
			PostID:   100,
			AuthorID: 1,
			Text:     "Test comment",
		}

		expectedID := int32(50)
		expectedTime := time.Now()

		mock.ExpectQuery(`INSERT INTO comments`).
			WithArgs(comment.PostID, comment.AuthorID, comment.Text).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(expectedID, expectedTime, expectedTime))

		err := store.CreateComment(ctx, comment)
		assert.NoError(t, err)
		assert.Equal(t, expectedID, comment.ID)
		assert.Equal(t, expectedTime, comment.CreatedAt)
		assert.Equal(t, expectedTime, comment.UpdatedAt)
	})

	t.Run("DB error", func(t *testing.T) {
		comment := &domain.Comment{
			PostID:   100,
			AuthorID: 1,
			Text:     "Test comment",
		}

		mock.ExpectQuery(`INSERT INTO comments`).
			WithArgs(comment.PostID, comment.AuthorID, comment.Text).
			WillReturnError(errors.New("db error"))

		err := store.CreateComment(ctx, comment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create comment")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBCommentStore_GetCommentByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewDBCommentStore(db).(*DBCommentStore)
	ctx := context.Background()

	t.Run("Success with parent_id", func(t *testing.T) {
		commentID := int32(100)
		parentID := int32(50)
		expectedTime := time.Now()

		mock.ExpectQuery(`SELECT.*FROM comments WHERE id = \$1`).
			WithArgs(commentID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
				AddRow(commentID, 200, 1, parentID, "Test comment", expectedTime, expectedTime))

		comment, err := store.GetCommentByID(ctx, commentID)
		assert.NoError(t, err)
		assert.NotNil(t, comment)
		assert.Equal(t, commentID, comment.ID)
		assert.Equal(t, &parentID, comment.ParentID)
		assert.Equal(t, "Test comment", comment.Text)
	})

	t.Run("Success without parent_id", func(t *testing.T) {
		commentID := int32(101)
		expectedTime := time.Now()

		mock.ExpectQuery(`SELECT.*FROM comments WHERE id = \$1`).
			WithArgs(commentID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
				AddRow(commentID, 200, 1, nil, "Test comment", expectedTime, expectedTime))

		comment, err := store.GetCommentByID(ctx, commentID)
		assert.NoError(t, err)
		assert.NotNil(t, comment)
		assert.Nil(t, comment.ParentID)
	})

	t.Run("Comment not found", func(t *testing.T) {
		commentID := int32(999)

		mock.ExpectQuery(`SELECT.*FROM comments WHERE id = \$1`).
			WithArgs(commentID).
			WillReturnError(sql.ErrNoRows)

		comment, err := store.GetCommentByID(ctx, commentID)
		assert.Error(t, err)
		assert.Nil(t, comment)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error", func(t *testing.T) {
		commentID := int32(100)

		mock.ExpectQuery(`SELECT.*FROM comments WHERE id = \$1`).
			WithArgs(commentID).
			WillReturnError(errors.New("db error"))

		comment, err := store.GetCommentByID(ctx, commentID)
		assert.Error(t, err)
		assert.Nil(t, comment)
		assert.Contains(t, err.Error(), "failed to get comment")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBCommentStore_GetCommentsByPost(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewDBCommentStore(db).(*DBCommentStore)
	ctx := context.Background()

	t.Run("Success with comments", func(t *testing.T) {
		postID := int32(100)
		limit := int32(10)
		offset := int32(0)
		expectedTime := time.Now()

		rows := sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
			AddRow(1, postID, 1, nil, "Comment 1", expectedTime, expectedTime).
			AddRow(2, postID, 2, 1, "Comment 2", expectedTime, expectedTime)

		mock.ExpectQuery(`SELECT.*FROM comments WHERE post_id = \$1.*LIMIT \$2 OFFSET \$3`).
			WithArgs(postID, limit, offset).
			WillReturnRows(rows)

		comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
		assert.NoError(t, err)
		assert.Len(t, comments, 2)
		assert.Equal(t, postID, comments[0].PostID)
		assert.Equal(t, postID, comments[1].PostID)
		assert.Nil(t, comments[0].ParentID)
		assert.Equal(t, int32(1), *comments[1].ParentID)
	})

	t.Run("Empty result", func(t *testing.T) {
		postID := int32(999)
		limit := int32(10)
		offset := int32(0)

		rows := sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"})

		mock.ExpectQuery(`SELECT.*FROM comments WHERE post_id = \$1.*LIMIT \$2 OFFSET \$3`).
			WithArgs(postID, limit, offset).
			WillReturnRows(rows)

		comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
		assert.NoError(t, err)
		assert.Empty(t, comments)
	})

	t.Run("DB query error", func(t *testing.T) {
		postID := int32(100)
		limit := int32(10)
		offset := int32(0)

		mock.ExpectQuery(`SELECT.*FROM comments WHERE post_id = \$1.*LIMIT \$2 OFFSET \$3`).
			WithArgs(postID, limit, offset).
			WillReturnError(errors.New("db error"))

		comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
		assert.Error(t, err)
		assert.Nil(t, comments)
		assert.Contains(t, err.Error(), "failed to query comments")
	})

	t.Run("Scan error", func(t *testing.T) {
		postID := int32(100)
		limit := int32(10)
		offset := int32(0)

		rows := sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
			AddRow("invalid", postID, 1, nil, "Comment", time.Now(), time.Now()) // Неправильный тип для id

		mock.ExpectQuery(`SELECT.*FROM comments WHERE post_id = \$1.*LIMIT \$2 OFFSET \$3`).
			WithArgs(postID, limit, offset).
			WillReturnRows(rows)

		comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
		assert.Error(t, err)
		assert.Nil(t, comments)
		assert.Contains(t, err.Error(), "failed to scan comment")
	})

	t.Run("Rows iteration error", func(t *testing.T) {
		postID := int32(100)
		limit := int32(10)
		offset := int32(0)

		rows := sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
			AddRow(1, postID, 1, nil, "Comment", time.Now(), time.Now()).
			RowError(0, errors.New("row error"))

		mock.ExpectQuery(`SELECT.*FROM comments WHERE post_id = \$1.*LIMIT \$2 OFFSET \$3`).
			WithArgs(postID, limit, offset).
			WillReturnRows(rows)

		comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
		assert.Error(t, err)
		assert.Nil(t, comments)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBCommentStore_UpdateComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewDBCommentStore(db).(*DBCommentStore)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		comment := &domain.Comment{
			ID:       100,
			AuthorID: 1,
			Text:     "Updated text",
		}
		expectedTime := time.Now()

		mock.ExpectQuery(`UPDATE comments SET text = \$1.*WHERE id = \$2 AND author_id = \$3`).
			WithArgs(comment.Text, comment.ID, comment.AuthorID).
			WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(expectedTime))

		err := store.UpdateComment(ctx, comment)
		assert.NoError(t, err)
		assert.Equal(t, expectedTime, comment.UpdatedAt)
	})

	t.Run("Comment not found", func(t *testing.T) {
		comment := &domain.Comment{
			ID:       999,
			AuthorID: 1,
			Text:     "Updated text",
		}

		mock.ExpectQuery(`UPDATE comments SET text = \$1.*WHERE id = \$2 AND author_id = \$3`).
			WithArgs(comment.Text, comment.ID, comment.AuthorID).
			WillReturnError(sql.ErrNoRows)

		err := store.UpdateComment(ctx, comment)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error", func(t *testing.T) {
		comment := &domain.Comment{
			ID:       100,
			AuthorID: 1,
			Text:     "Updated text",
		}

		mock.ExpectQuery(`UPDATE comments SET text = \$1.*WHERE id = \$2 AND author_id = \$3`).
			WithArgs(comment.Text, comment.ID, comment.AuthorID).
			WillReturnError(errors.New("db error"))

		err := store.UpdateComment(ctx, comment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update comment")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBCommentStore_DeleteComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewDBCommentStore(db).(*DBCommentStore)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		commentID := int32(100)
		authorID := int32(1)

		mock.ExpectExec(`DELETE FROM comments WHERE id = \$1 AND author_id = \$2`).
			WithArgs(commentID, authorID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := store.DeleteComment(ctx, commentID, authorID)
		assert.NoError(t, err)
	})

	t.Run("Comment not found", func(t *testing.T) {
		commentID := int32(999)
		authorID := int32(1)

		mock.ExpectExec(`DELETE FROM comments WHERE id = \$1 AND author_id = \$2`).
			WithArgs(commentID, authorID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := store.DeleteComment(ctx, commentID, authorID)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})

	t.Run("DB error on exec", func(t *testing.T) {
		commentID := int32(100)
		authorID := int32(1)

		mock.ExpectExec(`DELETE FROM comments WHERE id = \$1 AND author_id = \$2`).
			WithArgs(commentID, authorID).
			WillReturnError(errors.New("db error"))

		err := store.DeleteComment(ctx, commentID, authorID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete comment")
	})

	t.Run("DB error on rows affected", func(t *testing.T) {
		commentID := int32(100)
		authorID := int32(1)

		result := sqlmock.NewErrorResult(errors.New("rows affected error"))
		mock.ExpectExec(`DELETE FROM comments WHERE id = \$1 AND author_id = \$2`).
			WithArgs(commentID, authorID).
			WillReturnResult(result)

		err := store.DeleteComment(ctx, commentID, authorID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBCommentStore_GetPostCommentsCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewDBCommentStore(db).(*DBCommentStore)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		postID := int32(100)
		expectedCount := int32(42)

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM comments WHERE post_id = \$1`).
			WithArgs(postID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedCount))

		count, err := store.GetPostCommentsCount(ctx, postID)
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
	})

	t.Run("DB error", func(t *testing.T) {
		postID := int32(100)

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM comments WHERE post_id = \$1`).
			WithArgs(postID).
			WillReturnError(errors.New("db error"))

		count, err := store.GetPostCommentsCount(ctx, postID)
		assert.Error(t, err)
		assert.Equal(t, int32(0), count)
		assert.Contains(t, err.Error(), "failed to get comments count")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBCommentStore_NewDBCommentStore(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewDBCommentStore(db)
	assert.NotNil(t, store)

	// Проверяем, что возвращается правильный тип
	_, ok := store.(*DBCommentStore)
	assert.True(t, ok)
}