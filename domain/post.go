package domain

import (
	"context"
	"mime/multipart"
	"time"
)

type Post struct {
	ID          uint      `json:"id"`       //в БД табличка posts называется
	AuthorID    uint      `json:"authorID"` //в БД табличка posts называется
	CommunityID *int      `json:"communityID,omitempty"`
	Text        string    `json:"text"`       //в БД табличка posts называется
	CreatedAt   time.Time `json:"created_at"` //в БД табличка posts называется
	UpdatedAt   time.Time `json:"updated_at"` //в БД табличка posts называется

	Attachments []string `json:"attachments"` //в БД табличка post_attachments называется
	PhotosPath  []string `json:"photos"`      //в БД табличка post_photos называется
	LikeCount   int      `json:"likeCount"`
	IsLiked     bool     `json:"isLiked"`
}

// Унифицированная структура для отображения поста
type PostView struct {
	ID              uint      `json:"id"`
	AuthorID        uint      `json:"authorID"` // ID пользователя-создателя
	AuthorName      string    `json:"authorName"`
	AuthorAvatar    *string   `json:"authorAvatar"`
	CommunityID     *int      `json:"communityID,omitempty"`
	CommunityName   *string   `json:"communityName,omitempty"`
	CommunityAvatar *string   `json:"communityAvatar,omitempty"`
	Text            string    `json:"text"`
	Attachments     []string  `json:"attachments"`
	Photos          []string  `json:"photos"`
	LikeCount       int       `json:"likeCount"`
	IsLiked         bool      `json:"isLiked"`
	CreatedAt       time.Time `json:"createdAt"`
	IsCommunityPost bool      `json:"isCommunityPost"`
}

type PostWithAuthor struct {
	Post      Post         `json:"post"`
	Author    ShortProfile `json:"author"`
	Community *Community   `json:"community,omitempty"` // Информация о сообществе для постов в сообществах
}

type PostWithShortUser struct {
	Post   Post         `json:"post"`
	Author ShortProfile `json:"author"`
}

// PostCreateRequest - запрос на создание поста для валидации
type PostCreateRequest struct {
	Text        string   `json:"text" valid:"optional,length(0|4096)"`
	CommunityID *int     `json:"communityID" valid:"optional"` // Новое поле
	Attachments []string `json:"attachments" valid:"optional"`
	Photos      []string `json:"photos" valid:"optional"`
}

// PostUpdateRequest - запрос на обновление поста для валидации
type PostUpdateRequest struct {
	Text        string   `json:"text" valid:"optional,length(0|4096)"`
	Attachments []string `json:"attachments" valid:"optional"`
	Photos      []string `json:"photos" valid:"optional"`
}

// PostFeedItem - элемент ленты, который может быть от пользователя или сообщества
type PostFeedItem struct {
	Post        Post          `json:"post"`
	Author      *ShortProfile `json:"author,omitempty"`    // Для постов пользователей
	Community   *Community    `json:"community,omitempty"` // Для постов сообществ
	IsCommunity bool          `json:"isCommunity"`         // Флаг, указывающий тип поста
}

type PostService interface {
	// Получение постов с пагинацией (включая посты из сообществ)
	PostsPaginate(ctx context.Context, userID int, params PaginateQueryParams) ([]PostView, error)

	// Получение поста по ID
	GetPost(ctx context.Context, userID int, postID uint) (*PostView, error)

	// Создание поста
	CreatePost(ctx context.Context, userID int, text string, communityID *int, attachmentFiles []*multipart.FileHeader, photoFiles []*multipart.FileHeader) (*Post, error)

	// Обновление поста
	UpdatePost(ctx context.Context, postID uint, userID int, text string, attachmentFiles []*multipart.FileHeader, photoFiles []*multipart.FileHeader) error

	// Удаление поста
	DeletePost(ctx context.Context, postID uint, userID int) error

	// Получение постов пользователя
	GetUserPosts(ctx context.Context, selfUserID int, userID uint, params PaginateQueryParams) ([]PostView, error)

	// Получение постов сообщества
	GetCommunityPosts(ctx context.Context, userID int, communityID int, params PaginateQueryParams) ([]PostView, error)

	// Лайк/дизлайк поста
	UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int) error
}

type PostStore interface {
	// Получение постов с пагинацией
	PostsPaginatedList(ctx context.Context, userID, limit, offset int) ([]PostView, error)

	// Получение поста по ID
	GetPostByID(ctx context.Context, userID int, id uint) (*PostView, error)

	// Создание поста
	CreatePost(ctx context.Context, post *Post) error

	// Обновление поста
	UpdatePost(ctx context.Context, post *Post) error

	// Удаление поста
	DeletePost(ctx context.Context, id uint, authorID uint) error

	// Получение постов пользователя
	GetPostsByUser(ctx context.Context, selfUserID int, userID uint, limit, offset int) ([]PostView, error)

	// Получение постов сообщества
	GetCommunityPosts(ctx context.Context, userID int, communityID int, limit, offset int) ([]PostView, error)

	// Лайк/дизлайк поста
	UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int) error
}
