package domain

import "time"

type Post struct {
	ID        uint      `json:"id"`
	AuthorID  uint      `json:"authorID"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Attachments []string `json:"attachments"`
	PhotosPath  []string `json:"photos"`

	GroupName       string `json:"groupName"`
	CommunityAvatar string `json:"communityAvatar"`

	LikeCount    uint `json:"likes"`
	RepostsCount uint `json:"reposts"`
	CommentCount uint `json:"comments"`
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
