package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
	"strings"
	"time"

	"go.uber.org/zap"
)

type DBPostStore struct {
	db *sql.DB
}

func NewDBPostStore(db *sql.DB) domain.PostStore {
	return &DBPostStore{db: db}
}

// Возвращает пагинированный слайс постов
func (store *DBPostStore) PostsPaginatedList(ctx context.Context, userID, limit, offset int32) ([]domain.PostView, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start PostsPaginatedList", zap.Int32("offset", offset), zap.Int32("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
        SELECT 
            p.id,
            p.author_id,
            p.community_id,
            p.text,
            p.created_at,
            u.first_name || ' ' || u.last_name as user_name,
            u.avatar_path as user_avatar,
            c.name as community_name,
            c.avatar_path as community_avatar,
            COALESCE(likes.count, 0) AS likes_count,
            EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $3) AS liked_by_user
        FROM posts p
        LEFT JOIN profiles u ON p.author_id = u.user_id
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
    `

	rows, err := store.db.QueryContext(ctx, query, limit, offset, userID)
	if err != nil {
		dblogger.Error("Failed to query posts", zap.Error(err))
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	posts := []domain.PostView{}

	for rows.Next() {
		var (
			postView   domain.PostView
			commID     sql.NullInt64
			commName   sql.NullString
			commAvatar sql.NullString
		)

		err := rows.Scan(
			&postView.ID,
			&postView.AuthorID,
			&commID,
			&postView.Text,
			&postView.CreatedAt,
			&postView.AuthorName,
			&postView.AuthorAvatar,
			&commName,
			&commAvatar,
			&postView.LikeCount,
			&postView.IsLiked,
		)
		if err != nil {
			dblogger.Error("Failed to scan post", zap.Error(err))
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		// Обрабатываем community_id
		if commID.Valid {
			communityID := int32(commID.Int64)
			postView.CommunityID = &communityID
			postView.IsCommunityPost = true

			if commName.Valid {
				postView.CommunityName = &commName.String
			}
			if commAvatar.Valid {
				postView.CommunityAvatar = &commAvatar.String
			}
		}

		// Загружаем вложения и фото
		attachments, photos, err := store.getPostMedia(ctx, postView.ID)
		if err != nil {
			dblogger.Error("Failed to load post media", zap.Error(err))
			return nil, err
		}

		postView.Attachments = attachments
		postView.Photos = photos

		posts = append(posts, postView)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Rows iteration error", zap.Error(err))
		return nil, err
	}

	return posts, nil
}

// Возвращает пост по ID поста
func (store *DBPostStore) GetPostByID(ctx context.Context, userID int32, id uint) (*domain.PostView, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetPostByID", zap.Uint("postID", id))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT 
			p.id,
			p.author_id,
			p.community_id,
			p.text,
			p.created_at,
			u.first_name || ' ' || u.last_name as user_name,
			u.avatar_path as user_avatar,
			c.name as community_name,
			c.avatar_path as community_avatar,
			COALESCE(likes.count, 0) AS likes_count,
			EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2) AS liked_by_user
		FROM posts p
		LEFT JOIN profiles u ON p.author_id = u.user_id
		LEFT JOIN communities c ON p.community_id = c.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM post_likes
			GROUP BY post_id
		) likes ON likes.post_id = p.id
		WHERE p.id = $1
	`

	dblogger = dblogger.With(zap.String("query", query))
	var postView domain.PostView
	var commID sql.NullInt64
	var commName sql.NullString
	var commAvatar sql.NullString

	err := store.db.QueryRowContext(ctx, query, id, userID).Scan(
		&postView.ID,
		&postView.AuthorID,
		&commID,
		&postView.Text,
		&postView.CreatedAt,
		&postView.AuthorName,
		&postView.AuthorAvatar,
		&commName,
		&commAvatar,
		&postView.LikeCount,
		&postView.IsLiked,
	)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Post not found")
		return nil, domain.ErrPostNotFound
	}

	if err != nil {
		dblogger.Error("Failed to get post", zap.Error(err))
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	// Обрабатываем community_id
	if commID.Valid {
		communityID := int32(commID.Int64)
		postView.CommunityID = &communityID
		postView.IsCommunityPost = true

		if commName.Valid {
			postView.CommunityName = &commName.String
		}
		if commAvatar.Valid {
			postView.CommunityAvatar = &commAvatar.String
		}
	}

	// Загружаем attachments и photos
	attachments, photos, err := store.getPostMedia(ctx, id)
	if err != nil {
		return nil, err
	}

	postView.Attachments = attachments
	postView.Photos = photos

	dblogger.Info("Post retrieved successfully")
	return &postView, nil
}

// Создает новый пост с транзакцией
func (store *DBPostStore) CreatePost(ctx context.Context, post *domain.Post) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CreatePost", zap.Uint("authorID", post.AuthorID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	//Начинаем транзакцию
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		dblogger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	//В случае ошибки откатываем транзакцию. Если не получим вложения или фото.
	defer func() {
		if err != nil {
			tx.Rollback()
			dblogger.Error("Transaction rolled back", zap.Error(err))
		}
	}()
	//Создание поста
	query := `
        INSERT INTO posts (author_id, text, community_id)
        VALUES ($1, $2, $3)
        RETURNING id, created_at, updated_at
    `

	dblogger = dblogger.With(zap.String("query", query))
	err = tx.QueryRowContext(ctx, query, post.AuthorID, post.Text, post.CommunityID).Scan(
		&post.ID,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		dblogger.Error("Failed to create post", zap.Error(err))
		return fmt.Errorf("failed to create post: %w", err)
	}

	//Сохраняем вложения и фотографии в той же транзакции
	if err := store.savePostAttachmentsTx(ctx, tx, post.ID, post.Attachments); err != nil {
		return err
	}
	if err := store.savePostPhotosTx(ctx, tx, post.ID, post.Photos); err != nil {
		return err
	}

	//Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		dblogger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dblogger.Info("Post created successfully", zap.Uint("postID", post.ID))
	return nil
}

// Обновляет существующий пост
func (store *DBPostStore) UpdatePost(ctx context.Context, post *domain.Post) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdatePost", zap.Uint("postID", post.ID), zap.Uint("authorID", post.AuthorID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	//Начинаем транзакцию
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		dblogger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	//В случае ошибки откатываем транзакцию. Если не получим вложения или фото.
	defer func() {
		if err != nil {
			tx.Rollback()
			dblogger.Error("Transaction rolled back", zap.Error(err))
		}
	}()

	query := `
        UPDATE posts 
        SET text = $1, updated_at = $2
        WHERE id = $3 AND author_id = $4
        RETURNING updated_at
    `
	dblogger = dblogger.With(zap.String("query", query))
	err = tx.QueryRowContext(ctx, query, post.Text, time.Now(), post.ID, post.AuthorID).Scan(
		&post.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Post not found for update")
		return domain.ErrPostNotFound
	}
	if err != nil {
		dblogger.Error("Failed to update post", zap.Error(err))
		return fmt.Errorf("failed to update post: %w", err)
	}

	//Обновляем вложения и фотографии
	if err := store.updatePostAttachmentsTx(ctx, tx, post.ID, post.Attachments); err != nil {
		return err
	}

	if err := store.updatePostPhotosTx(ctx, tx, post.ID, post.Photos); err != nil {
		return err
	}

	//Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		dblogger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dblogger.Info("Post updated successfully")
	return nil
}

// Получает посты сообщества
func (store *DBPostStore) GetCommunityPosts(ctx context.Context, userID int32, communityID int32, limit, offset int32) ([]domain.PostView, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetCommunityPosts", zap.Int32("communityID", communityID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT 
			p.id,
			p.author_id,
			p.community_id,
			p.text,
			p.created_at,
			u.first_name || ' ' || u.last_name as user_name,
			u.avatar_path as user_avatar,
			c.name as community_name,
			c.avatar_path as community_avatar,
			COALESCE(likes.count, 0) AS likes_count,
			EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2) AS liked_by_user
		FROM posts p
		LEFT JOIN profiles u ON p.author_id = u.user_id
		LEFT JOIN communities c ON p.community_id = c.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM post_likes
			GROUP BY post_id
		) likes ON likes.post_id = p.id
		WHERE p.community_id = $1
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, communityID, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query community posts", zap.Error(err))
		return nil, fmt.Errorf("failed to query community posts: %w", err)
	}
	defer rows.Close()

	posts := []domain.PostView{}
	for rows.Next() {
		var postView domain.PostView
		var commID sql.NullInt64
		var commName sql.NullString
		var commAvatar sql.NullString

		err := rows.Scan(
			&postView.ID,
			&postView.AuthorID,
			&commID,
			&postView.Text,
			&postView.CreatedAt,
			&postView.AuthorName,
			&postView.AuthorAvatar,
			&commName,
			&commAvatar,
			&postView.LikeCount,
			&postView.IsLiked,
		)
		if err != nil {
			dblogger.Error("Failed to scan post", zap.Error(err))
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		// Обрабатываем community данные
		postView.IsCommunityPost = true
		if commName.Valid {
			postView.CommunityName = &commName.String
		}
		if commAvatar.Valid {
			postView.CommunityAvatar = &commAvatar.String
		}

		// Загружаем вложения и фото
		attachments, photos, err := store.getPostMedia(ctx, postView.ID)
		if err != nil {
			return nil, err
		}

		postView.Attachments = attachments
		postView.Photos = photos
		posts = append(posts, postView)
	}

	dblogger.Info("Community posts retrieved successfully", zap.Int("postsCount", len(posts)))
	return posts, nil
}

// Удаляет существующий пост
func (store *DBPostStore) DeletePost(ctx context.Context, id uint, authorID uint) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start DeletePost", zap.Uint("postID", id), zap.Uint("authorID", authorID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	if id == 0 || authorID == 0 {
		dblogger.Warn("Invalid input parameters")
		return domain.ErrInvalidInput
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		dblogger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			dblogger.Error("Transaction rolled back", zap.Error(err))
		}
	}()
	//Удаляем пост
	query := `
	DELETE FROM posts 
	WHERE id = $1 AND author_id = $2
	`
	dblogger = dblogger.With(zap.String("query", query))
	result, err := tx.ExecContext(ctx, query, id, authorID)

	if err != nil {
		dblogger.Error("Failed to delete post", zap.Error(err))
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, err := result.RowsAffected() //Вернет количество обновленных строк
	if err != nil {
		dblogger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 { //Если ни одной строки не обновлено, значит поста не было
		dblogger.Warn("Post not found for deletion")
		return domain.ErrPostNotFound
	}

	if err := tx.Commit(); err != nil {
		dblogger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dblogger.Info("Post deleted successfully")
	return nil
}

// Получение постов пользователя с пагинацией
func (store *DBPostStore) GetPostsByUser(ctx context.Context, selfUserID int32, userID uint, limit, offset int32) ([]domain.PostView, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetPostsByUser", zap.Uint("userID", userID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT 
			p.id,
			p.author_id,
			p.community_id,
			p.text,
			p.created_at,
			u.first_name || ' ' || u.last_name as user_name,
			u.avatar_path as user_avatar,
			c.name as community_name,
			c.avatar_path as community_avatar,
			COALESCE(likes.count, 0) AS likes_count,
			EXISTS (SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = $2) AS liked_by_user
		FROM posts p
		LEFT JOIN profiles u ON p.author_id = u.user_id
		LEFT JOIN communities c ON p.community_id = c.id
		LEFT JOIN (
			SELECT post_id, COUNT(*) AS count
			FROM post_likes
			GROUP BY post_id
		) likes ON likes.post_id = p.id
		WHERE p.author_id = $1 AND p.community_id IS NULL
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, selfUserID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query user posts", zap.Error(err))
		return nil, fmt.Errorf("failed to query user posts: %w", err)
	}
	defer rows.Close()

	posts := []domain.PostView{}
	for rows.Next() {
		var postView domain.PostView
		var commID sql.NullInt64
		var commName sql.NullString
		var commAvatar sql.NullString

		err := rows.Scan(
			&postView.ID,
			&postView.AuthorID,
			&commID,
			&postView.Text,
			&postView.CreatedAt,
			&postView.AuthorName,
			&postView.AuthorAvatar,
			&commName,
			&commAvatar,
			&postView.LikeCount,
			&postView.IsLiked,
		)
		if err != nil {
			dblogger.Error("Failed to scan post", zap.Error(err))
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		// В постах пользователя community_id должен быть NULL
		postView.IsCommunityPost = false

		// Загружаем вложения и фото
		attachments, photos, err := store.getPostMedia(ctx, postView.ID)
		if err != nil {
			return nil, err
		}

		postView.Attachments = attachments
		postView.Photos = photos
		posts = append(posts, postView)
	}

	dblogger.Info("User posts retrieved successfully", zap.Int("postsCount", len(posts)))
	return posts, nil
}

// НИЖЕ БУДУТ ПРИВЕДЕНЫ ВСПОМОГАТЕЛЬНЫЕ ФУКНЦИИ

// Получение слайса путей ВЛОЖЕНИЙ и ФОТОГРАФИЙ
func (store *DBPostStore) getPostMedia(ctx context.Context, postID uint) ([]string, []string, error) {
	attachments, err := store.getPostAttachments(ctx, postID)
	if err != nil {
		return nil, nil, err
	}
	photos, err := store.getPostPhotos(ctx, postID)
	if err != nil {
		return nil, nil, err
	}

	return attachments, photos, nil
}

// Получение слайса путей вложений
func (store *DBPostStore) getPostAttachments(ctx context.Context, postID uint) ([]string, error) {
	query := `
        SELECT file_path
        FROM post_attachments 
        WHERE post_id = $1
		ORDER BY id
    `

	rows, err := store.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to query attachments: %w", err)
	}
	defer rows.Close()

	var attachments []string
	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, filePath)
	}

	return attachments, nil
}

// Получение слайса путей фотографий
func (store *DBPostStore) getPostPhotos(ctx context.Context, postID uint) ([]string, error) {
	query := `
        SELECT file_path
        FROM post_photos 
        WHERE post_id = $1
		ORDER BY id
    `

	rows, err := store.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to query post photos: %w", err)
	}
	defer rows.Close()

	var photos []string

	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			return nil, fmt.Errorf("failed to scan photo: %w", err)
		}
		photos = append(photos, filePath)
	}

	return photos, nil
}

// Сохранение слайса путей ВЛОЖЕНИЙ через транзакции.
func (store *DBPostStore) savePostAttachmentsTx(ctx context.Context, tx *sql.Tx, postID uint, attachments []string) error {
	if len(attachments) == 0 {
		return nil
	}

	for _, attachment := range attachments {
		if strings.TrimSpace(attachment) == "" {
			return fmt.Errorf("attachment path cannot be empty")
		}
		if len(attachment) < 1 || len(attachment) > 512 {
			return fmt.Errorf("attachment path length must be between 1 and 512 characters")
		}
		query := `
			INSERT INTO post_attachments (post_id, file_path) 
			VALUES ($1, $2)
		`
		_, err := tx.ExecContext(ctx, query, postID, attachment)
		if err != nil {
			return fmt.Errorf("failed to save attachment: %w", err)
		}
	}

	return nil
}

// Сохранение слайса путей ФОТОГРАФИЙ через транзакции.
func (store *DBPostStore) savePostPhotosTx(ctx context.Context, tx *sql.Tx, postID uint, photos []string) error {
	if len(photos) == 0 {
		return nil
	}
	for _, photo := range photos {
		if strings.TrimSpace(photo) == "" {
			return fmt.Errorf("photo path cannot be empty")
		}
		if len(photo) < 1 || len(photo) > 512 {
			return fmt.Errorf("photo path length must be between 1 and 512 characters")
		}

		query := `
			INSERT INTO post_photos (post_id, file_path)
			VALUES ($1, $2)
		`
		_, err := tx.ExecContext(ctx, query, postID, photo)
		if err != nil {
			return fmt.Errorf("failed to save photo: %w", err)
		}
	}

	return nil
}

// Обновление ВЛОЖЕНИЙ поста через транзакции
func (store *DBPostStore) updatePostAttachmentsTx(ctx context.Context, tx *sql.Tx, postID uint, attachments []string) error {
	//Удаляем старые ВЛОЖЕНИЯ (Потом можно будет запихнуть в отдельную функцию удаления)
	query := `
		DELETE FROM post_attachments
		WHERE post_id = $1
	`
	_, err := tx.ExecContext(ctx, query, postID)
	if err != nil {
		return fmt.Errorf("failed to delete old attachments: %w", err)
	}
	// Вставляем новые ВЛОЖЕНИЯ
	return store.savePostAttachmentsTx(ctx, tx, postID, attachments)
}

// Обновление ФОТОГРАФИЙ поста через транзакции
func (store *DBPostStore) updatePostPhotosTx(ctx context.Context, tx *sql.Tx, postID uint, photos []string) error {
	//Удаляем старые ФОТОГРАФИИ (Потом можно будет запихнуть в отдельную функцию удаления)
	query := `
		DELETE FROM post_photos
		WHERE post_id = $1
	`
	_, err := tx.ExecContext(ctx, query, postID)
	if err != nil {
		return fmt.Errorf("failed to delete old photos: %w", err)
	}
	// Вставляем новые ВЛОЖЕНИЯ
	return store.savePostPhotosTx(ctx, tx, postID, photos)
}

func (store *DBPostStore) UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int32) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateLikeOnPostByUserID")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
WITH toggled AS (
    DELETE FROM post_likes
    WHERE post_id = $1 AND user_id = $2
    RETURNING *
)
INSERT INTO post_likes (post_id, user_id)
SELECT $1, $2
WHERE NOT EXISTS (SELECT 1 FROM toggled)
	`
	_, err := store.db.ExecContext(ctx, query, postID, userID)
	dblogger = dblogger.With(zap.String("query", query), zap.Int32("userID", userID), zap.Int32("postID", postID))
	if err != nil {
		dblogger.Error("Failed update like on post")
		return fmt.Errorf("exec failed: %w", err)
	}

	dblogger.Info("like on post updated")
	return nil

}
