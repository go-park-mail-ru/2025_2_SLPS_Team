package db

import (
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
	"strings"
	"time"
)

type DBPostStore struct {
	db *sql.DB
}

func NewDBPostStore(db *sql.DB) domain.PostStore {
	return &DBPostStore{db: db}
}

// Возвращает пагинированный слайс постов
func (store *DBPostStore) PostsPaginatedList(page, limit int) ([]domain.Post, int, error) {
	if page < 1 || limit < 1 { //У нас нет отрицательных или нулевых страниц, также я не могу отрисовать на странице -7 постов
		return nil, 0, domain.ErrInvalidInput
	}

	offset := (page - 1) * limit //Смещенение для игнорирования первых offset постов

	query := `
        SELECT p.id, p.author_id, p.text, p.created_at, p.updated_at
        FROM posts p
        ORDER BY p.created_at DESC
        LIMIT $1 OFFSET $2
    `

	rows, err := store.db.Query(query, limit, offset) // Получаем посты с пагинацией
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var posts []domain.Post //Слайс типа domain.Post, сюда мы будем добавлять считанные из rows посты post. Наша функция возвращает этот слайс

	for rows.Next() { //Начинаем считывать строки из sql запроса ПОСТРОЧНО!

		var post domain.Post //Структура нашего поста

		err := rows.Scan( //Scan записывает столбцы из sql запроса rows в поля нашей структуры post ПО УКАЗАТЕЛЮ. Возвращает ошибку
			&post.ID,
			&post.AuthorID,
			&post.Text,
			&post.CreatedAt,
			&post.UpdatedAt,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan post: %w", err)
		}

		// Загружаем attachments и photos
		attachments, photos, err := store.getPostMedia(post.ID)
		if err != nil {
			return nil, 0, err
		}

		post.Attachments = attachments
		post.PhotosPath = photos
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	// Получаем общее количество для пагинации
	var total int
	countQuery := `SELECT COUNT(*) FROM posts`
	err = store.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count posts: %w", err)
	}

	totalPages := (total + limit - 1) / limit
	return posts, totalPages, nil
}

// Возвращает пост по ID поста
func (store *DBPostStore) GetPostByID(id uint) (*domain.Post, error) {
	query := `
        SELECT p.id, p.author_id, p.text, p.created_at, p.updated_at
        FROM posts p
        WHERE id = $1
    `

	var post domain.Post
	err := store.db.QueryRow(query, id).Scan(
		&post.ID,
		&post.AuthorID,
		&post.Text,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("post not found: %w", err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	// Загружаем attachments и photos
	attachments, photos, err := store.getPostMedia(id)
	if err != nil {
		return nil, err
	}

	post.Attachments = attachments
	post.PhotosPath = photos

	return &post, nil
}

// Создает новый пост с транзакцией
func (store *DBPostStore) CreatePost(post *domain.Post) error {
	//Валидируем входные данные структуры post в соответствии с CONSTRAINT в БД с помощью функции
	if err := store.validatePost(post); err != nil {
		return err
	}
	//Начинаем транзакцию
	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	//В случае ошибки откатываем транзакцию. Если не получим вложения или фото.
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	//Создание поста
	query := `
        INSERT INTO posts (author_id, text)
        VALUES ($1, $2)
        RETURNING id, created_at, updated_at
    `
	err = tx.QueryRow(query, post.AuthorID, post.Text).Scan(
		&post.ID,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	//Сохраняем вложения и фотографии в той же транзакции
	if err := store.savePostAttachmentsTx(tx, post.ID, post.Attachments); err != nil {
		return err
	}
	if err := store.savePostPhotosTx(tx, post.ID, post.PhotosPath); err != nil {
		return err
	}

	//Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Обновляет существующий пост
func (store *DBPostStore) UpdatePost(post *domain.Post) error {
	//Валидируем входные данные структуры post в соответствии с CONSTRAINT в БД с помощью функции
	if err := store.validatePost(post); err != nil {
		return err
	}
	//Начинаем транзакцию
	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	//В случае ошибки откатываем транзакцию. Если не получим вложения или фото.
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `
        UPDATE posts 
        SET text = $1, updated_at = $2
        WHERE id = $3 AND author_id = $4
        RETURNING updated_at
    `

	err = tx.QueryRow(query, post.Text, time.Now(), post.ID, post.AuthorID).Scan(
		&post.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrPostNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	//Обновляем вложения и фотографии
	if err := store.updatePostAttachmentsTx(tx, post.ID, post.Attachments); err != nil {
		return err
	}

	if err := store.updatePostPhotosTx(tx, post.ID, post.PhotosPath); err != nil {
		return err
	}

	//Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Удаляет существующий пост
func (store *DBPostStore) DeletePost(id uint, authorID uint) error {
	if id == 0 || authorID == 0 {
		return domain.ErrInvalidInput
	}

	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	//Удаляем пост
	query := `
	DELETE FROM posts 
	WHERE id = $1 AND author_id = $2
	`
	result, err := tx.Exec(query, id, authorID)

	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, err := result.RowsAffected() //Вернет количество обновленных строк
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 { //Если ни одной строки не обновлено, значит поста не было
		return domain.ErrPostNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Получение постов пользователя с пагинацией
func (store *DBPostStore) GetPostsByUser(userID uint, page, limit int) ([]domain.Post, int, error) {
	if userID == 0 || page < 1 || limit < 1 {
		return nil, 0, domain.ErrInvalidInput
	}

	offset := (page - 1) * limit

	query := `
        SELECT id, author_id, text, created_at, updated_at
        FROM posts 
        WHERE author_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := store.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user posts: %w", err)
	}
	defer rows.Close()

	var posts []domain.Post
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
			return nil, 0, fmt.Errorf("failed to scan post: %w", err)
		}

		attachments, photos, err := store.getPostMedia(post.ID)
		if err != nil {
			return nil, 0, err
		}

		post.Attachments = attachments
		post.PhotosPath = photos
		posts = append(posts, post)
	}

	// Count для пагинации
	var total int
	countQuery := `SELECT COUNT(*) FROM posts WHERE author_id = $1`
	err = store.db.QueryRow(countQuery, userID).Scan(
		&total,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user posts: %w", err)
	}

	totalPages := (total + limit - 1) / limit
	return posts, totalPages, nil
}

// НИЖЕ БУДУТ ПРИВЕДЕНЫ ВСПОМОГАТЕЛЬНЫЕ ФУКНЦИИ

// Валидация данных поста
func (store *DBPostStore) validatePost(post *domain.Post) error {
	if post.AuthorID == 0 {
		return domain.ErrPostInvalidAuthor
	}

	text := strings.TrimSpace(post.Text)
	if text == "" {
		return domain.ErrPostTextEmpty
	}
	if len(text) < 24 {
		return domain.ErrPostTextTooShort
	}
	if len(text) > 4096 {
		return domain.ErrPostTextTooLong
	}

	return nil
}

// Получение слайса путей ВЛОЖЕНИЙ и ФОТОГРАФИЙ
func (store *DBPostStore) getPostMedia(postID uint) ([]string, []string, error) {
	attachments, err := store.getPostAttachments(postID)
	if err != nil {
		return nil, nil, err
	}
	photos, err := store.getPostPhotos(postID)
	if err != nil {
		return nil, nil, err
	}

	return attachments, photos, nil
}

// Получение слайса путей вложений
func (store *DBPostStore) getPostAttachments(postID uint) ([]string, error) {
	query := `
        SELECT file_path,  
        FROM attachments 
        WHERE obj_id = $1 AND obj_type = 'post'
		ORDER BY id
    `

	rows, err := store.db.Query(query, postID)
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
func (store *DBPostStore) getPostPhotos(postID uint) ([]string, error) {
	query := `
        SELECT file_path,  
        FROM post_photos 
        WHERE post_id = $1
		ORDER BY id
    `

	rows, err := store.db.Query(query, postID)
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
func (store *DBPostStore) savePostAttachmentsTx(tx *sql.Tx, postID uint, attachments []string) error {
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
			INSERT INTO attachments (obj_id, obj_type, file_path) 
			VALUES ($1, 'post', $2)
		`
		_, err := tx.Exec(query, postID, attachment)
		if err != nil {
			return fmt.Errorf("failed to save attachment: %w", err)
		}
	}

	return nil
}

// Сохранение слайса путей ФОТОГРАФИЙ через транзакции.
func (store *DBPostStore) savePostPhotosTx(tx *sql.Tx, postID uint, photos []string) error {
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
			INSERT INTO photos (post_id, file_path)
			VALUES ($1, $2)
		`
		_, err := tx.Exec(query, postID, photo)
		if err != nil {
			return fmt.Errorf("failed to save photo: %w", err)
		}
	}

	return nil
}

// Обновление ВЛОЖЕНИЙ поста через транзакции
func (store *DBPostStore) updatePostAttachmentsTx(tx *sql.Tx, postID uint, attachments []string) error {
	//Удаляем старые ВЛОЖЕНИЯ (Потом можно будет запихнуть в отдельную функцию удаления)
	query := `
		DELETE FROM attachments
		WHERE obj_id = $1 AND obj_type = 'post'
	`
	_, err := tx.Exec(query, postID)
	if err != nil {
		return fmt.Errorf("failed to delete old attachments: %w", err)
	}
	// Вставляем новые ВЛОЖЕНИЯ
	return store.savePostAttachmentsTx(tx, postID, attachments)
}

// Обновление ФОТОГРАФИЙ поста через транзакции
func (store *DBPostStore) updatePostPhotosTx(tx *sql.Tx, postID uint, attachments []string) error {
	//Удаляем старые ФОТОГРАФИИ (Потом можно будет запихнуть в отдельную функцию удаления)
	query := `
		DELETE FROM post_photos
		WHERE post_id = $1
	`
	_, err := tx.Exec(query, postID)
	if err != nil {
		return fmt.Errorf("failed to delete old attachments: %w", err)
	}
	// Вставляем новые ВЛОЖЕНИЯ
	return store.savePostPhotosTx(tx, postID, attachments)
}
