package db

import (
	"context"
	"database/sql"
	"project/domain"
	"regexp"
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

	rows := sqlmock.NewRows([]string{"id", "author_id", "community_id", "text", "created_at", "community_name", "community_avatar", "likes_count", "liked_by_user"}).
		AddRow(1, 10, nil, "Post text here that is long enough", time.Now(), nil, nil, 5, false)
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT 
            p.id,
            p.author_id,
            p.community_id,
            p.text,
            p.created_at,
            c.name as community_name,
            c.avatar_path as community_avatar,
            COALESCE(likes.count, 0) AS likes_count,
            EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $3) AS liked_by_user
        FROM posts p
        LEFT JOIN communities c ON p.community_id = c.id
        LEFT JOIN (
            SELECT post_id, COUNT(*) AS count
            FROM post_likes
            GROUP BY post_id
        ) likes ON likes.post_id = p.id
        WHERE p.community_id IS NULL 
        OR p.community_id IN (SELECT community_id FROM community_subscriptions WHERE user_id = $3)
        ORDER BY p.created_at DESC
        LIMIT $1 OFFSET $2
    `)).
		WithArgs(10, 0, int32(1)).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT file_path FROM post_attachments`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(`SELECT file_path FROM post_photos`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	posts, err := store.PostsPaginatedList(context.Background(), int32(1), int32(10), int32(0))
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
}

func TestGetPostByID(t *testing.T) {
	_, mock, store := setupMockPostDB(t)
	now := time.Now()

	row := sqlmock.NewRows([]string{"id", "author_id", "community_id", "text", "created_at", "community_name", "community_avatar", "likes_count", "liked_by_user"}).
		AddRow(1, 10, nil, "Valid post text longer than 24 chars", now, nil, nil, 3, true)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT 
			p.id,
			p.author_id,
			p.community_id,
			p.text,
			p.created_at,
			c.name as community_name,
			c.avatar_path as community_avatar,
			COALESCE(likes.count, 0) AS likes_count,
			EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2) AS liked_by_user
		FROM posts p
		LEFT JOIN communities c ON p.community_id = c.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM post_likes
			GROUP BY post_id
		) likes ON likes.post_id = p.id
		WHERE p.id = $1
	`)).
		WithArgs(uint(1), int32(1)).WillReturnRows(row)
	mock.ExpectQuery(`SELECT file_path FROM post_attachments`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(`SELECT file_path FROM post_photos`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	post, err := store.GetPostByID(context.Background(), int32(1), uint(1))
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
		Photos:      []string{"photo1.jpg"},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO posts`).
		WithArgs(post.AuthorID, post.Text, nil).
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
		Photos:      []string{"photo2.jpg"},
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
		WithArgs(uint(1), uint(1)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := store.DeletePost(context.Background(), uint(1), uint(1))
	assert.NoError(t, err)
}

func TestGetPostsByUser(t *testing.T) {
	_, mock, store := setupMockPostDB(t)
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "author_id", "community_id", "text", "created_at", "community_name", "community_avatar", "likes_count", "liked_by_user"}).
		AddRow(1, 10, nil, "User post text longer than 24 chars", now, nil, nil, 2, false)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT 
			p.id,
			p.author_id,
			p.community_id,
			p.text,
			p.created_at,
			c.name as community_name,
			c.avatar_path as community_avatar,
			COALESCE(likes.count, 0) AS likes_count,
			EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2) AS liked_by_user
		FROM posts p
		LEFT JOIN communities c ON p.community_id = c.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM post_likes
			GROUP BY post_id
		) likes ON likes.post_id = p.id
		WHERE p.author_id = $1 AND p.community_id IS NULL
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4
	`)).
		WithArgs(uint(10), int32(1), int32(10), int32(0)).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT file_path FROM post_attachments`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(`SELECT file_path FROM post_photos`).WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	posts, err := store.GetPostsByUser(context.Background(), int32(1), uint(10), int32(10), int32(0))
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
}
