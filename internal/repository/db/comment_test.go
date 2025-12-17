package db

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"project/domain"
)

func newCommentStoreMock(t *testing.T) (*DBCommentStore, sqlmock.Sqlmock, *sql.DB) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err, "failed to create sqlmock")
	store := NewDBCommentStore(dbConn).(*DBCommentStore)
	return store, mock, dbConn
}

func TestCreateComment_Success(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	comment := &domain.Comment{
		PostID:   100,
		AuthorID: 1,
		Text:     "Test comment",
	}
	createdAt := time.Now()
	updatedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO comments (post_id, author_id, text)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`)).
		WithArgs(comment.PostID, comment.AuthorID, comment.Text).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int32(42), createdAt, updatedAt))

	err := store.CreateComment(ctx, comment)
	assert.NoError(t, err)
	assert.Equal(t, int32(42), comment.ID)
	assert.Equal(t, createdAt, comment.CreatedAt)
	assert.Equal(t, updatedAt, comment.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateComment_Error(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	comment := &domain.Comment{
		PostID:   100,
		AuthorID: 1,
		Text:     "Test comment",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO comments`)).
		WithArgs(comment.PostID, comment.AuthorID, comment.Text).
		WillReturnError(errors.New("database error"))

	err := store.CreateComment(ctx, comment)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create comment")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentByID_Success(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	commentID := int32(100)
	parentID := int32(50)
	createdAt := time.Now()
	updatedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, post_id, author_id, parent_id, text, created_at, updated_at
		FROM comments
		WHERE id = $1
	`)).
		WithArgs(commentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
			AddRow(commentID, 200, 1, parentID, "Test comment", createdAt, updatedAt))

	comment, err := store.GetCommentByID(ctx, commentID)
	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, commentID, comment.ID)
	assert.Equal(t, &parentID, comment.ParentID)
	assert.Equal(t, "Test comment", comment.Text)
	assert.Equal(t, createdAt, comment.CreatedAt)
	assert.Equal(t, updatedAt, comment.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentByID_SuccessWithoutParentID(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	commentID := int32(101)
	createdAt := time.Now()
	updatedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, post_id, author_id, parent_id, text, created_at, updated_at
		FROM comments
		WHERE id = $1
	`)).
		WithArgs(commentID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
			AddRow(commentID, 200, 1, nil, "Test comment", createdAt, updatedAt))

	comment, err := store.GetCommentByID(ctx, commentID)
	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Nil(t, comment.ParentID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentByID_NotFound(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	commentID := int32(999)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, post_id, author_id, parent_id, text, created_at, updated_at FROM comments WHERE id = $1`)).
		WithArgs(commentID).
		WillReturnError(sql.ErrNoRows)

	comment, err := store.GetCommentByID(ctx, commentID)
	assert.Error(t, err)
	assert.Nil(t, comment)
	assert.Equal(t, domain.ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentByID_Error(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	commentID := int32(100)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, post_id, author_id, parent_id, text, created_at, updated_at FROM comments WHERE id = $1`)).
		WithArgs(commentID).
		WillReturnError(errors.New("database error"))

	comment, err := store.GetCommentByID(ctx, commentID)
	assert.Error(t, err)
	assert.Nil(t, comment)
	assert.Contains(t, err.Error(), "failed to get comment")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentsByPost_Success(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	postID := int32(100)
	limit := int32(10)
	offset := int32(0)
	createdAt := time.Now()
	updatedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, post_id, author_id, parent_id, text, created_at, updated_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`)).
		WithArgs(postID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
			AddRow(1, postID, 1, nil, "Comment 1", createdAt, updatedAt).
			AddRow(2, postID, 2, 1, "Comment 2", createdAt, updatedAt))

	comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, comments, 2)
	assert.Equal(t, int32(1), comments[0].ID)
	assert.Equal(t, postID, comments[0].PostID)
	assert.Nil(t, comments[0].ParentID)
	assert.Equal(t, int32(2), comments[1].ID)
	assert.Equal(t, int32(1), *comments[1].ParentID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentsByPost_EmptyResult(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	postID := int32(999)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, post_id, author_id, parent_id, text, created_at, updated_at FROM comments WHERE post_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs(postID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}))

	comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
	assert.NoError(t, err)
	assert.Empty(t, comments)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentsByPost_QueryError(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	postID := int32(100)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, post_id, author_id, parent_id, text, created_at, updated_at FROM comments WHERE post_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs(postID, limit, offset).
		WillReturnError(errors.New("database error"))

	comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, comments)
	assert.Contains(t, err.Error(), "failed to query comments")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentsByPost_ScanError(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	postID := int32(100)
	limit := int32(10)
	offset := int32(0)
	createdAt := time.Now()
	updatedAt := time.Now()

	// Первая строка нормальная, вторая - с неправильным типом данных для id
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, post_id, author_id, parent_id, text, created_at, updated_at FROM comments WHERE post_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs(postID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
			AddRow(1, postID, 1, nil, "Comment 1", createdAt, updatedAt).
			AddRow("invalid", postID, 2, nil, "Comment 2", createdAt, updatedAt)) // Неправильный тип для id

	comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, comments)
	assert.Contains(t, err.Error(), "failed to scan comment")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommentsByPost_RowsErr(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	postID := int32(100)
	limit := int32(10)
	offset := int32(0)
	createdAt := time.Now()
	updatedAt := time.Now()

	rows := sqlmock.NewRows([]string{"id", "post_id", "author_id", "parent_id", "text", "created_at", "updated_at"}).
		AddRow(1, postID, 1, nil, "Comment 1", createdAt, updatedAt).
		AddRow(2, postID, 2, 1, "Comment 2", createdAt, updatedAt).
		RowError(1, errors.New("row iteration error"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, post_id, author_id, parent_id, text, created_at, updated_at FROM comments WHERE post_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs(postID, limit, offset).
		WillReturnRows(rows)

	comments, err := store.GetCommentsByPost(ctx, postID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, comments)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateComment_Success(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	comment := &domain.Comment{
		ID:       100,
		AuthorID: 1,
		Text:     "Updated text",
	}
	updatedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE comments
		SET text = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND author_id = $3
		RETURNING updated_at
	`)).
		WithArgs(comment.Text, comment.ID, comment.AuthorID).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(updatedAt))

	err := store.UpdateComment(ctx, comment)
	assert.NoError(t, err)
	assert.Equal(t, updatedAt, comment.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateComment_NotFound(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	comment := &domain.Comment{
		ID:       999,
		AuthorID: 1,
		Text:     "Updated text",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE comments SET text = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND author_id = $3 RETURNING updated_at`)).
		WithArgs(comment.Text, comment.ID, comment.AuthorID).
		WillReturnError(sql.ErrNoRows)

	err := store.UpdateComment(ctx, comment)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateComment_Error(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	comment := &domain.Comment{
		ID:       100,
		AuthorID: 1,
		Text:     "Updated text",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE comments SET text = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND author_id = $3 RETURNING updated_at`)).
		WithArgs(comment.Text, comment.ID, comment.AuthorID).
		WillReturnError(errors.New("database error"))

	err := store.UpdateComment(ctx, comment)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update comment")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteComment_Success(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	commentID := int32(100)
	authorID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM comments WHERE id = $1 AND author_id = $2`)).
		WithArgs(commentID, authorID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := store.DeleteComment(ctx, commentID, authorID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteComment_NotFound(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	commentID := int32(999)
	authorID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM comments WHERE id = $1 AND author_id = $2`)).
		WithArgs(commentID, authorID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.DeleteComment(ctx, commentID, authorID)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteComment_ExecError(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	commentID := int32(100)
	authorID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM comments WHERE id = $1 AND author_id = $2`)).
		WithArgs(commentID, authorID).
		WillReturnError(errors.New("database error"))

	err := store.DeleteComment(ctx, commentID, authorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete comment")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteComment_RowsAffectedError(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	commentID := int32(100)
	authorID := int32(1)

	// Создаем результат с ошибкой для RowsAffected
	result := sqlmock.NewErrorResult(errors.New("rows affected error"))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM comments WHERE id = $1 AND author_id = $2`)).
		WithArgs(commentID, authorID).
		WillReturnResult(result)

	err := store.DeleteComment(ctx, commentID, authorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPostCommentsCount_Success(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	postID := int32(100)
	expectedCount := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM comments WHERE post_id = $1`)).
		WithArgs(postID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedCount))

	count, err := store.GetPostCommentsCount(ctx, postID)
	assert.NoError(t, err)
	assert.Equal(t, expectedCount, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPostCommentsCount_Error(t *testing.T) {
	store, mock, dbConn := newCommentStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	postID := int32(100)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM comments WHERE post_id = $1`)).
		WithArgs(postID).
		WillReturnError(errors.New("database error"))

	count, err := store.GetPostCommentsCount(ctx, postID)
	assert.Error(t, err)
	assert.Equal(t, int32(0), count)
	assert.Contains(t, err.Error(), "failed to get comments count")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNewDBCommentStore(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err, "failed to create sqlmock")
	defer dbConn.Close()

	store := NewDBCommentStore(dbConn)
	assert.NotNil(t, store)

	// Проверяем, что возвращается правильный тип
	_, ok := store.(*DBCommentStore)
	assert.True(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}