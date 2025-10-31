package domain

import (
	"context"
	"time"
)

type Post struct {
	ID        uint      `json:"id"`         //в БД табличка posts называется
	AuthorID  uint      `json:"authorID"`   //в БД табличка posts называется
	Text      string    `json:"text"`       //в БД табличка posts называется
	CreatedAt time.Time `json:"created_at"` //в БД табличка posts называется
	UpdatedAt time.Time `json:"updated_at"` //в БД табличка posts называется

	Attachments []string `json:"attachments"` //в БД табличка post_attachments называется
	PhotosPath  []string `json:"photos"`      //в БД табличка post_photos называется

}

type PostStore interface {
	// Получение постов с пагинацией
	PostsPaginatedList(ctx context.Context, page, limit int) ([]Post, int, error)
	// Получение поста по ID
	GetPostByID(ctx context.Context, id uint) (*Post, error)
	// Создание поста
	CreatePost(ctx context.Context, post *Post) error
	// Обновление поста
	UpdatePost(ctx context.Context, post *Post) error
	// Удаление поста
	DeletePost(ctx context.Context, id uint, authorID uint) error
	// Получение постов пользователя
	GetPostsByUser(ctx context.Context, userID uint, page, limit int) ([]Post, int, error)
}
