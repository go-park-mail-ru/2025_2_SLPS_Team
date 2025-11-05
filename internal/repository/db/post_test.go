package db

import (
	"context"
	"database/sql"
	"project/domain"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockPostDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *DBPostStore) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	store := NewDBPostStore(mockDB).(*DBPostStore)
	return mockDB, mock, store
}

func TestPostsPaginatedList(t *testing.T) {
	_, mock, store := setupMockPostDB(t)
	rows := sqlmock.NewRows([]string{"id", "author_id", "text", "created_at", "updated_at"}).
		AddRow(1, 10, "Post text here that is long enough", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT p.id, p.author_id, p.text, p.created_at, p.updated_at FROM posts`).
		WithArgs(10, 0).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT file_path FROM post_attachments`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(`SELECT file_path FROM post_photos`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	posts, err := store.PostsPaginatedList(context.Background(), 10, 0)
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
}

func TestGetPostByID(t *testing.T) {
	_, mock, store := setupMockPostDB(t)
	now := time.Now()
	row := sqlmock.NewRows([]string{"id", "author_id", "text", "created_at", "updated_at"}).
		AddRow(1, 10, "Valid post text longer than 24 chars", now, now)
	mock.ExpectQuery(`SELECT p.id, p.author_id, p.text, p.created_at, p.updated_at FROM posts`).
		WithArgs(1).WillReturnRows(row)
	mock.ExpectQuery(`SELECT file_path FROM post_attachments`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(`SELECT file_path FROM post_photos`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	post, err := store.GetPostByID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), post.ID)
}

func TestCreatePost(t *testing.T) {
	_, mock, store := setupMockPostDB(t)
	now := time.Now()
	post := &domain.Post{
		AuthorID:    1,
		Text:        "Valid post text longer than 24 characters",
		Attachments: []string{"file1.pdf"},
		PhotosPath:  []string{"photo1.jpg"},
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO posts`).
		WithArgs(post.AuthorID, post.Text).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, now, now))
	mock.ExpectExec(`INSERT INTO post_attachments`).WithArgs(1, "file1.pdf").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO post_photos`).WithArgs(1, "photo1.jpg").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := store.CreatePost(context.Background(), post)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), post.ID)
}

func TestUpdatePost(t *testing.T) {
	_, mock, store := setupMockPostDB(t)
	post := &domain.Post{
		ID:          1,
		AuthorID:    1,
		Text:        "Updated post text longer than 24 characters",
		Attachments: []string{"file2.pdf"},
		PhotosPath:  []string{"photo2.jpg"},
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE posts`).
		WithArgs(post.Text, sqlmock.AnyArg(), post.ID, post.AuthorID).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(time.Now()))
	mock.ExpectExec(`DELETE FROM post_attachments`).WithArgs(post.ID).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO post_attachments`).WithArgs(post.ID, "file2.pdf").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM post_photos`).WithArgs(post.ID).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO post_photos`).WithArgs(post.ID, "photo2.jpg").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := store.UpdatePost(context.Background(), post)
	assert.NoError(t, err)
}

func TestDeletePost(t *testing.T) {
	_, mock, store := setupMockPostDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM posts`).
		WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err := store.DeletePost(context.Background(), 1, 1)
	assert.NoError(t, err)
}

func TestGetPostsByUser(t *testing.T) {
	_, mock, store := setupMockPostDB(t)
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "author_id", "text", "created_at", "updated_at"}).
		AddRow(1, 10, "User post text longer than 24 chars", now, now)
	mock.ExpectQuery(`SELECT id, author_id, text, created_at, updated_at FROM posts`).
		WithArgs(10, 10, 0).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT file_path FROM post_attachments`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(`SELECT file_path FROM post_photos`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	posts, err := store.GetPostsByUser(context.Background(), 10, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
}
