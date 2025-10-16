package domain

import "time"

type Post struct {
	ID        uint      `json:"id"`         //в БД табличка posts называется
	AuthorID  uint      `json:"authorID"`   //в БД табличка posts называется
	Text      string    `json:"text"`       //в БД табличка posts называется
	CreatedAt time.Time `json:"created_at"` //в БД табличка posts называется
	UpdatedAt time.Time `json:"updated_at"` //в БД табличка posts называется

	Attachments []string `json:"attachments"` //в БД табличка post_attachments называется
	PhotosPath  []string `json:"photos"`      //в БД табличка post_photos называется

	// GroupName       string `json:"groupName"`        //Это с заделом на будущее
	// CommunityAvatar string `json:"communityAvatar"`

	// LikeCount    uint `json:"likes"`
	// RepostsCount uint `json:"reposts"`
	// CommentCount uint `json:"comments"`
}

type PostStore interface {
	// Получение постов с пагинацией
	PostsPaginatedList(page, limit int) ([]Post, int, error)

	// Получение поста по ID
	GetPostByID(id uint) (*Post, error)

	// Создание поста
	CreatePost(post *Post) error

	// Обновление поста
	UpdatePost(post *Post) error

	// Удаление поста
	DeletePost(id uint, authorID uint) error

	// Получение постов пользователя
	GetPostsByUser(userID uint, page, limit int) ([]Post, int, error)
}
