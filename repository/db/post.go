package db

import (
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
	"time"
)

type DBPostStore struct {
	db *sql.DB
}

func NewDBPostStore(db *sql.DB) domain.PostStore {
	return &DBPostStore{db: db}
}

func (store *DBPostStore) PostsPaginatedList(page, limit int) ([]domain.Post, int, error) {
	offset := (page - 1) * limit

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
		attachments, err := store.getPostAttachments(post.ID)
		if err != nil {
			return nil, 0, err
		}

		photos, err := store.getPostPhotos(post.ID)
		if err != nil {
			return nil, 0, err
		}

		post.Attachments = attachments
		post.PhotosPath = photos

		posts = append(posts, post)
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
	attachments, err := store.getPostAttachments(id)
	if err != nil {
		return nil, err
	}

	photos, err := store.getPostPhotos(id)
	if err != nil {
		return nil, err
	}

	post.Attachments = attachments
	post.PhotosPath = photos

	return &post, nil
}

func (store *DBPostStore) CreatePost(post *domain.Post) error {
	query := `
        INSERT INTO posts (author_id, text)
        VALUES ($1, $2)
        RETURNING id, created_at, updated_at
    `

	err := store.db.QueryRow(query, post.AuthorID, post.Text).Scan(
		&post.ID,
		&post.CreatedAt,
		&post.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	// Сохраняем attachments и photos
	if err := store.savePostAttachments(post.ID, post.Attachments, post.PhotosPath); err != nil {
		return err
	}

	return nil
}

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

func (store *DBPostStore) savePostAttachments(postID uint, attachments []string) error {
	if len(attachments) == 0 {
		return nil
	}

	for _, attachment := range attachments {

		query := `
			INSERT INTO attachments (obj_id, obj_type, file_path) 
			VALUES ($1, 'post', $2)
		`

		_, err := store.db.Exec(query, postID, attachment)
		if err != nil {
			return fmt.Errorf("failed to save attachment: %w", err)
		}
	}

	return nil
}

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

// Остальные методы (UpdatePost, DeletePost, GetPostsByUser) реализуются аналогично
func (store *DBPostStore) UpdatePost(post *domain.Post) error {
	query := `
        UPDATE posts 
        SET text = $1, updated_at = $2, like_count = $3, repost_count = $4, 
            comment_count = $5, group_name = $6, community_avatar = $7
        WHERE id = $8 AND author_id = $9
        RETURNING updated_at
    `

	err := store.db.QueryRow(
		query,
		post.Text, time.Now(), post.ID, post.AuthorID,
	).Scan(&post.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("post not found or access denied")
	}
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	// Обновляем attachments (в реальном приложении - удалить старые, добавить новые)
	return nil
}

func (store *DBPostStore) DeletePost(id uint, authorID uint) error {
	query := `DELETE FROM posts WHERE id = $1 AND author_id = $2`
	result, err := store.db.Exec(query, id, authorID)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("post not found or access denied")
	}

	return nil
}

func (store *DBPostStore) GetPostsByUser(userID uint, page, limit int) ([]domain.Post, int, error) {
	// Аналогично PostsPaginatedList с дополнительным WHERE author_id = $1
	offset := (page - 1) * limit

	query := `
        SELECT id, author_id, text, created_at, updated_at,
            like_count, repost_count, comment_count,
            group_name, community_avatar
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
			&post.ID, &post.AuthorID, &post.Text, &post.CreatedAt, &post.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan post: %w", err)
		}

		attachments, photos, err := store.getPostAttachments(post.ID)
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
	err = store.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user posts: %w", err)
	}

	totalPages := (total + limit - 1) / limit
	return posts, totalPages, nil
}
