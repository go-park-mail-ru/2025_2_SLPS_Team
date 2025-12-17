package db

import (
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

func newPostStoreMock(t *testing.T) (*DBPostStore, sqlmock.Sqlmock, *sql.DB) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err, "failed to create sqlmock")
	store := NewDBPostStore(dbConn).(*DBPostStore)
	return store, mock, dbConn
}

func TestPostsPaginatedList_Success(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)
	now := time.Now()

	// Mock основного запроса
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
		COALESCE(comments.count, 0) AS comments_count, -- <-- ДОБАВИЛИ
		EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $3) AS liked_by_user
	FROM posts p
	LEFT JOIN communities c ON p.community_id = c.id
	LEFT JOIN (
		SELECT post_id, COUNT(*) AS count
		FROM post_likes
		GROUP BY post_id
	) likes ON likes.post_id = p.id
	LEFT JOIN ( -- <-- ДОБАВИЛИ
		SELECT post_id, COUNT(*) AS count
		FROM comments
		GROUP BY post_id
	) comments ON comments.post_id = p.id
	WHERE p.community_id IS NULL 
	OR p.community_id IN (SELECT community_id FROM community_subscriptions WHERE user_id = $3)
	ORDER BY p.created_at DESC
	LIMIT $1 OFFSET $2
`)).
		WithArgs(limit, offset, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "author_id", "community_id", "text", "created_at",
			"community_name", "community_avatar", "likes_count", "comments_count", "liked_by_user",
		}).
			AddRow(uint(1), uint(100), nil, "Post 1", now, nil, nil, int32(5), int32(3), true).
			AddRow(uint(2), uint(101), int64(50), "Post 2", now.Add(-time.Hour), "Community", "avatar.jpg", int32(10), int32(7), false))

	// Mock для вложений первого поста
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_attachments WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).
			AddRow("attachments/file1.pdf"))

	// Mock для фотографий первого поста
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_photos WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).
			AddRow("photos/photo1.jpg").
			AddRow("photos/photo2.jpg"))

	// Mock для вложений второго поста
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_attachments WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(2)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	// Mock для фотографий второго поста
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_photos WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(2)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	posts, err := store.PostsPaginatedList(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, posts, 2)

	// Проверяем первый пост (без сообщества)
	assert.Equal(t, uint(1), posts[0].ID)
	assert.Equal(t, uint(100), posts[0].AuthorID)
	assert.Nil(t, posts[0].CommunityID)
	assert.Equal(t, "Post 1", posts[0].Text)
	assert.Equal(t, int32(5), posts[0].LikeCount)
	assert.Equal(t, int32(3), posts[0].CommentsCount)
	assert.True(t, posts[0].IsLiked)
	assert.Equal(t, []string{"attachments/file1.pdf"}, posts[0].Attachments)
	assert.Equal(t, []string{"photos/photo1.jpg", "photos/photo2.jpg"}, posts[0].Photos)

	// Проверяем второй пост (с сообществом)
	assert.Equal(t, uint(2), posts[1].ID)
	assert.Equal(t, uint(101), posts[1].AuthorID)
	assert.Equal(t, int32(50), *posts[1].CommunityID)
	assert.Equal(t, "Community", *posts[1].CommunityName)
	assert.Equal(t, "avatar.jpg", *posts[1].CommunityAvatar)
	assert.Equal(t, int32(10), posts[1].LikeCount)
	assert.Equal(t, int32(7), posts[1].CommentsCount)
	assert.False(t, posts[1].IsLiked)
	assert.Empty(t, posts[1].Attachments)
	assert.Empty(t, posts[1].Photos)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostsPaginatedList_Empty(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(limit, offset, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "author_id", "community_id", "text", "created_at",
			"community_name", "community_avatar", "likes_count", "comments_count", "liked_by_user",
		}))

	posts, err := store.PostsPaginatedList(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Empty(t, posts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostsPaginatedList_QueryError(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(limit, offset, userID).
		WillReturnError(errors.New("database error"))

	posts, err := store.PostsPaginatedList(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, posts)
	assert.Contains(t, err.Error(), "failed to query posts")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostsPaginatedList_ScanError(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	// Неправильное количество колонок
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(limit, offset, userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id"}).
			AddRow(uint(1), uint(100)))

	posts, err := store.PostsPaginatedList(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, posts)
	assert.Contains(t, err.Error(), "failed to scan post")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostsPaginatedList_MediaQueryError(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(limit, offset, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "author_id", "community_id", "text", "created_at",
			"community_name", "community_avatar", "likes_count", "comments_count", "liked_by_user",
		}).AddRow(uint(1), uint(100), nil, "Post 1", now, nil, nil, int32(5), int32(3), true))

	// Ошибка при запросе вложений
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_attachments WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(1)).
		WillReturnError(errors.New("attachments error"))

	posts, err := store.PostsPaginatedList(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, posts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPostByID_Success(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	postID := uint(42)
	now := time.Now()

	// Mock основного запроса
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
			COALESCE(comments.count, 0) AS comments_count,
			EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2) AS liked_by_user
		FROM posts p
		LEFT JOIN communities c ON p.community_id = c.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM post_likes
			GROUP BY post_id
		) likes ON likes.post_id = p.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM comments
			GROUP BY post_id
		) comments ON comments.post_id = p.id
		WHERE p.id = $1
	`)).
		WithArgs(postID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "author_id", "community_id", "text", "created_at",
			"community_name", "community_avatar", "likes_count", "comments_count", "liked_by_user",
		}).AddRow(postID, uint(100), int64(50), "Test Post", now, "Community", "avatar.jpg", int32(25), int32(10), true))

	// Mock для вложений
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_attachments WHERE post_id = $1 ORDER BY id`)).
		WithArgs(postID).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).
			AddRow("attachments/doc.pdf"))

	// Mock для фотографий
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_photos WHERE post_id = $1 ORDER BY id`)).
		WithArgs(postID).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).
			AddRow("photos/image1.jpg").
			AddRow("photos/image2.jpg"))

	post, err := store.GetPostByID(ctx, userID, postID)
	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Equal(t, postID, post.ID)
	assert.Equal(t, uint(100), post.AuthorID)
	assert.Equal(t, int32(50), *post.CommunityID)
	assert.Equal(t, "Test Post", post.Text)
	assert.Equal(t, int32(25), post.LikeCount)
	assert.Equal(t, int32(10), post.CommentsCount)
	assert.True(t, post.IsLiked)
	assert.Equal(t, []string{"attachments/doc.pdf"}, post.Attachments)
	assert.Equal(t, []string{"photos/image1.jpg", "photos/image2.jpg"}, post.Photos)
	assert.Equal(t, "Community", *post.CommunityName)
	assert.Equal(t, "avatar.jpg", *post.CommunityAvatar)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPostByID_NotFound(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	postID := uint(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(postID, userID).
		WillReturnError(sql.ErrNoRows)

	post, err := store.GetPostByID(ctx, userID, postID)
	assert.Error(t, err)
	assert.Nil(t, post)
	assert.True(t, errors.Is(err, domain.ErrPostNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPostByID_WithoutCommunity(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	postID := uint(42)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(postID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "author_id", "community_id", "text", "created_at",
			"community_name", "community_avatar", "likes_count", "comments_count", "liked_by_user",
		}).AddRow(postID, uint(100), nil, "Personal Post", now, nil, nil, int32(15), int32(5), false))

	// Mock для медиа
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_attachments WHERE post_id = $1 ORDER BY id`)).
		WithArgs(postID).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_photos WHERE post_id = $1 ORDER BY id`)).
		WithArgs(postID).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	post, err := store.GetPostByID(ctx, userID, postID)
	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Nil(t, post.CommunityID)
	assert.Nil(t, post.CommunityName)
	assert.Nil(t, post.CommunityAvatar)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePost_Success(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	post := &domain.Post{
		AuthorID:    uint(100),
		Text:        "New Post",
		Attachments: []string{"attachments/file.pdf"},
		Photos:      []string{"photos/photo.jpg"},
	}
	postID := uint(42)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO posts (author_id, text, community_id) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`)).
		WithArgs(post.AuthorID, post.Text, (*uint)(nil)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(postID, now, now))

	// Mock для сохранения вложений
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO post_attachments (post_id, file_path) VALUES ($1, $2)`)).
		WithArgs(postID, "attachments/file.pdf").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock для сохранения фотографий
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO post_photos (post_id, file_path) VALUES ($1, $2)`)).
		WithArgs(postID, "photos/photo.jpg").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err := store.CreatePost(ctx, post)
	assert.NoError(t, err)
	assert.Equal(t, postID, post.ID)
	assert.Equal(t, now, post.CreatedAt)
	assert.Equal(t, now, post.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePost_WithCommunity(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(50)
	post := &domain.Post{
		AuthorID:    uint(100),
		Text:        "Community Post",
		CommunityID: &communityID,
	}
	postID := uint(42)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO posts (author_id, text, community_id) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`)).
		WithArgs(post.AuthorID, post.Text, communityID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(postID, now, now))
	mock.ExpectCommit()

	err := store.CreatePost(ctx, post)
	assert.NoError(t, err)
	assert.Equal(t, postID, post.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePost_BeginTxError(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	post := &domain.Post{AuthorID: uint(100), Text: "New Post"}

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	err := store.CreatePost(ctx, post)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePost_InsertPostError(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	post := &domain.Post{AuthorID: uint(100), Text: "New Post"}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO posts`)).
		WithArgs(post.AuthorID, post.Text, (*uint)(nil)).
		WillReturnError(errors.New("insert error"))

	err := store.CreatePost(ctx, post)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create post")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePost_SaveAttachmentsError(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	post := &domain.Post{
		AuthorID:    uint(100),
		Text:        "Post with attachments",
		Attachments: []string{""}, // Пустой путь
	}
	postID := uint(42)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO posts (author_id, text, community_id) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`)).
		WithArgs(post.AuthorID, post.Text, (*uint)(nil)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(postID, now, now))

	err := store.CreatePost(ctx, post)
	assert.Error(t, err)
	assert.Equal(t, "attachment path cannot be empty", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePost_SavePhotosError(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	post := &domain.Post{
		AuthorID: uint(100),
		Text:     "Post with photos",
		Photos:   []string{""}, // Пустой путь
	}
	postID := uint(42)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO posts (author_id, text, community_id) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`)).
		WithArgs(post.AuthorID, post.Text, (*uint)(nil)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(postID, now, now))

	err := store.CreatePost(ctx, post)
	assert.Error(t, err)
	assert.Equal(t, "photo path cannot be empty", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePost_Success(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	post := &domain.Post{
		ID:          uint(42),
		AuthorID:    uint(100),
		Text:        "Updated post text",
		Attachments: []string{"new/attachment.pdf"},
		Photos:      []string{"new/photo.jpg"},
	}
	updatedAt := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE posts SET text = $1, updated_at = $2 WHERE id = $3 AND author_id = $4 RETURNING updated_at`)).
		WithArgs(post.Text, sqlmock.AnyArg(), post.ID, post.AuthorID).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(updatedAt))

	// Mock для удаления старых вложений
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM post_attachments WHERE post_id = $1`)).
		WithArgs(post.ID).
		WillReturnResult(sqlmock.NewResult(0, 2))

	// Mock для вставки новых вложений
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO post_attachments (post_id, file_path) VALUES ($1, $2)`)).
		WithArgs(post.ID, "new/attachment.pdf").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock для удаления старых фотографий
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM post_photos WHERE post_id = $1`)).
		WithArgs(post.ID).
		WillReturnResult(sqlmock.NewResult(0, 3))

	// Mock для вставки новых фотографий
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO post_photos (post_id, file_path) VALUES ($1, $2)`)).
		WithArgs(post.ID, "new/photo.jpg").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err := store.UpdatePost(ctx, post)
	assert.NoError(t, err)
	assert.Equal(t, updatedAt, post.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePost_NotFound(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	post := &domain.Post{
		ID:       uint(42),
		AuthorID: uint(100),
		Text:     "Updated post",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE posts`)).
		WithArgs(post.Text, sqlmock.AnyArg(), post.ID, post.AuthorID).
		WillReturnError(sql.ErrNoRows)

	err := store.UpdatePost(ctx, post)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrPostNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePost_NoMedia(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	post := &domain.Post{
		ID:       uint(42),
		AuthorID: uint(100),
		Text:     "Updated post without media",
	}
	updatedAt := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE posts SET text = $1, updated_at = $2 WHERE id = $3 AND author_id = $4 RETURNING updated_at`)).
		WithArgs(post.Text, sqlmock.AnyArg(), post.ID, post.AuthorID).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(updatedAt))

	// Ожидаем удаление медиа даже если их нет
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM post_attachments WHERE post_id = $1`)).
		WithArgs(post.ID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM post_photos WHERE post_id = $1`)).
		WithArgs(post.ID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectCommit()

	err := store.UpdatePost(ctx, post)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommunityPosts_Success(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	communityID := int32(50)
	limit := int32(10)
	offset := int32(0)
	now := time.Now()

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
			COALESCE(comments.count, 0) AS comments_count,
			EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2) AS liked_by_user
		FROM posts p
		LEFT JOIN communities c ON p.community_id = c.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM post_likes
			GROUP BY post_id
		) likes ON likes.post_id = p.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM comments
			GROUP BY post_id
		) comments ON comments.post_id = p.id
		WHERE p.community_id = $1
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4
	`)).
		WithArgs(communityID, userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "author_id", "community_id", "text", "created_at",
			"community_name", "community_avatar", "likes_count", "comments_count", "liked_by_user",
		}).AddRow(uint(1), uint(100), int64(communityID), "Community Post", now, "Community", "avatar.jpg", int32(10), int32(5), true))

	// Mock для медиа
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_attachments WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_photos WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	posts, err := store.GetCommunityPosts(ctx, userID, communityID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
	assert.Equal(t, communityID, int32(*posts[0].CommunityID))
	assert.Equal(t, "Community", *posts[0].CommunityName)
	assert.True(t, posts[0].IsLiked)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeletePost_Success(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	postID := uint(42)
	authorID := uint(100)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM posts WHERE id = $1 AND author_id = $2`)).
		WithArgs(postID, authorID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := store.DeletePost(ctx, postID, authorID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeletePost_NotFound(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	postID := uint(42)
	authorID := uint(100)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM posts WHERE id = $1 AND author_id = $2`)).
		WithArgs(postID, authorID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.DeletePost(ctx, postID, authorID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrPostNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeletePost_InvalidInput(t *testing.T) {
	store, _, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()

	tests := []struct {
		name     string
		postID   uint
		authorID uint
	}{
		{"Zero postID", 0, 100},
		{"Zero authorID", 42, 0},
		{"Both zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.DeletePost(ctx, tt.postID, tt.authorID)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, domain.ErrInvalidInput))
		})
	}
}

func TestGetPostsByUser_Success(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	selfUserID := int32(1)
	userID := uint(100)
	limit := int32(10)
	offset := int32(0)
	now := time.Now()

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
			COALESCE(comments.count, 0) AS comments_count,
			EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2) AS liked_by_user
		FROM posts p
		LEFT JOIN communities c ON p.community_id = c.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM post_likes
			GROUP BY post_id
		) likes ON likes.post_id = p.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM comments
			GROUP BY post_id
		) comments ON comments.post_id = p.id
		WHERE p.author_id = $1 AND p.community_id IS NULL
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4
	`)).
		WithArgs(userID, selfUserID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "author_id", "community_id", "text", "created_at",
			"community_name", "community_avatar", "likes_count", "comments_count", "liked_by_user",
		}).AddRow(uint(1), uint(100), nil, "Personal Post", now, nil, nil, int32(15), int32(8), false))

	// Mock для медиа
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_attachments WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM post_photos WHERE post_id = $1 ORDER BY id`)).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	posts, err := store.GetPostsByUser(ctx, selfUserID, userID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, posts, 1)
	assert.Equal(t, userID, uint(posts[0].AuthorID))
	assert.Nil(t, posts[0].CommunityID)
	assert.False(t, posts[0].IsLiked)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateLikeOnPostByUserID_Success(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(100)
	postID := int32(42)

	mock.ExpectExec(regexp.QuoteMeta(`
WITH toggled AS (
    DELETE FROM post_likes
    WHERE post_id = $1 AND user_id = $2
    RETURNING *
)
INSERT INTO post_likes (post_id, user_id)
SELECT $1, $2
WHERE NOT EXISTS (SELECT 1 FROM toggled)
	`)).
		WithArgs(postID, userID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.UpdateLikeOnPostByUserID(ctx, userID, postID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateLikeOnPostByUserID_ExecError(t *testing.T) {
	store, mock, dbConn := newPostStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(100)
	postID := int32(42)

	mock.ExpectExec(regexp.QuoteMeta(`WITH toggled AS`)).
		WithArgs(postID, userID).
		WillReturnError(errors.New("exec error"))

	err := store.UpdateLikeOnPostByUserID(ctx, userID, postID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exec failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}
