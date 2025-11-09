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
func (store *DBPostStore) PostsPaginatedList(ctx context.Context, limit, offset int) ([]domain.PostWithShortUser, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start PostsPaginatedList", zap.Int("offset", offset), zap.Int("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
        SELECT 
            p.id, p.author_id, p.text, p.created_at, p.updated_at,
            u.user_id, u.first_name ||' '|| u.last_name, u.avatar_path
        FROM posts p
        JOIN profiles u ON p.author_id = u.user_id
        ORDER BY p.created_at DESC
        LIMIT $1 OFFSET $2
    `

	rows, err := store.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query posts with authors", zap.Error(err))
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var postsWithAuthors []domain.PostWithShortUser

	for rows.Next() {
		var (
			post   domain.Post
			author domain.ShortProfile
		)

		err := rows.Scan(
			&post.ID,
			&post.AuthorID,
			&post.Text,
			&post.CreatedAt,
			&post.UpdatedAt,
			&author.UserID,
			&author.FullName,
			&author.AvatarPath,
		)
		if err != nil {
			dblogger.Error("Failed to scan post with author", zap.Error(err))
			return nil, fmt.Errorf("failed to scan post with author: %w", err)
		}

		// Загружаем вложения и фото для поста
		attachments, photos, err := store.getPostMedia(ctx, post.ID)
		if err != nil {
			dblogger.Error("Failed to load post media", zap.Error(err))
			return nil, err
		}

		post.Attachments = attachments
		post.PhotosPath = photos

		postsWithAuthors = append(postsWithAuthors, domain.PostWithShortUser{
			Post:   post,
			Author: author,
		})
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Rows iteration error", zap.Error(err))
		return nil, err
	}

	return postsWithAuthors, nil
}

// Возвращает пост по ID поста
func (store *DBPostStore) GetPostByID(ctx context.Context, id uint) (*domain.Post, error) {
	start := time.Now()                           //засекаем время начала операции
	dblogger := domain.DBLogger(ctx, "postStore") //создаем специализированный логгер для БД с тегами layer="db" и repo="postStore"
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetPostByID", zap.Uint("postID", id))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
        SELECT p.id, p.author_id, p.text, p.created_at, p.updated_at
        FROM posts p
        WHERE p.id = $1
    `

	dblogger = dblogger.With(zap.String("query", query))
	var post domain.Post
	err := store.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID,
		&post.AuthorID,
		&post.Text,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Post not found")
		return nil, domain.ErrPostNotFound
	}

	if err != nil {
		dblogger.Error("Failed to get post", zap.Error(err))
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	// Загружаем attachments и photos
	attachments, photos, err := store.getPostMedia(ctx, id)
	if err != nil {
		return nil, err
	}

	post.Attachments = attachments
	post.PhotosPath = photos

	dblogger.Info("Post retrieved successfully")
	return &post, nil
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

	//Валидируем входные данные структуры post в соответствии с CONSTRAINT в БД с помощью функции
	if err := store.validatePost(post); err != nil {
		dblogger.Warn("Post validation failed", zap.Error(err))
		return err
	}

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
        INSERT INTO posts (author_id, text)
        VALUES ($1, $2)
        RETURNING id, created_at, updated_at
    `

	dblogger = dblogger.With(zap.String("query", query))
	err = tx.QueryRowContext(ctx, query, post.AuthorID, post.Text).Scan(
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
	if err := store.savePostPhotosTx(ctx, tx, post.ID, post.PhotosPath); err != nil {
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
	//Валидируем входные данные структуры post в соответствии с CONSTRAINT в БД с помощью функции
	if err := store.validatePost(post); err != nil {
		dblogger.Warn("Post validation failed", zap.Error(err))
		return err
	}
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

	if err := store.updatePostPhotosTx(ctx, tx, post.ID, post.PhotosPath); err != nil {
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
func (store *DBPostStore) GetPostsByUser(ctx context.Context, userID uint, limit, offset int) ([]domain.Post, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "postStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetPostsByUser", zap.Uint("userID", userID), zap.Int("offset", offset), zap.Int("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
        SELECT id, author_id, text, created_at, updated_at
        FROM posts 
        WHERE author_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `
	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)

	if err != nil {
		dblogger.Error("Failed to query user posts", zap.Error(err))
		return nil, fmt.Errorf("failed to query user posts: %w", err)
	}
	defer rows.Close()

	posts := []domain.Post{}
	for rows.Next() {
		var post domain.Post
		err := rows.Scan(
			&post.ID,
			&post.AuthorID,
			&post.Text,
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			dblogger.Error("Failed to scan post", zap.Error(err))
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		attachments, photos, err := store.getPostMedia(ctx, post.ID)
		if err != nil {
			return nil, err
		}

		post.Attachments = attachments
		post.PhotosPath = photos
		posts = append(posts, post)
	}

	dblogger.Info("User posts retrieved successfully", zap.Int("postsCount", len(posts)))
	return posts, nil
}

// НИЖЕ БУДУТ ПРИВЕДЕНЫ ВСПОМОГАТЕЛЬНЫЕ ФУКНЦИИ

// Валидация данных поста
func (store *DBPostStore) validatePost(post *domain.Post) error {
	if post.AuthorID == 0 {
		return domain.ErrPostInvalidAuthor
	}

	text := strings.TrimSpace(post.Text)
	if len(text) > 4096 {
		return domain.ErrPostTextTooLong
	}

	return nil
}

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
