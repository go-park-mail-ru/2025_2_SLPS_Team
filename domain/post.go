package domain

import (
	"context"
	"time"
)

//easyjson:json
type Post struct {
	ID          uint      `json:"id"`       //в БД табличка posts называется
	AuthorID    uint      `json:"authorID"` //в БД табличка posts называется
	CommunityID *int32    `json:"communityID,omitempty"`
	Text        string    `json:"text"`       //в БД табличка posts называется
	CreatedAt   time.Time `json:"created_at"` //в БД табличка posts называется
	UpdatedAt   time.Time `json:"updated_at"` //в БД табличка posts называется

	Attachments []string `json:"attachments"` //в БД табличка post_attachments называется
	Photos      []string `json:"photos"`      //в БД табличка post_photos называется
	LikeCount   int32    `json:"likeCount"`
	IsLiked     bool     `json:"isLiked"`
}

//easyjson:json
type PostView struct {
	ID              uint      `json:"id"`
	AuthorID        uint      `json:"authorID"` // ID пользователя-создателя
	AuthorName      string    `json:"authorName"`
	AuthorAvatar    *string   `json:"authorAvatar"`
	CommunityID     *int32    `json:"communityID,omitempty"`
	CommunityName   *string   `json:"communityName,omitempty"`
	CommunityAvatar *string   `json:"communityAvatar,omitempty"`
	Text            string    `json:"text"`
	Attachments     []string  `json:"attachments"`
	Photos          []string  `json:"photos"`
	LikeCount       int32     `json:"likeCount"`
	CommentsCount   int32     `json:"commentsCount"`
	IsLiked         bool      `json:"isLiked"`
	CreatedAt       time.Time `json:"createdAt"`
	IsCommunityPost bool      `json:"isCommunityPost"`
}

//easyjson:json
type PostDB struct {
	ID              uint      `json:"id"`
	AuthorID        uint      `json:"authorID"`
	CommunityID     *int32    `json:"communityID,omitempty"`
	Text            string    `json:"text"`
	CreatedAt       time.Time `json:"createdAt"`
	CommunityName   *string   `json:"communityName,omitempty"`
	CommunityAvatar *string   `json:"communityAvatar,omitempty"`
	LikeCount       int32     `json:"likeCount"`
	CommentsCount   int32     `json:"commentsCount"`
	IsLiked         bool      `json:"isLiked"`
	Attachments     []string  `json:"attachments"`
	Photos          []string  `json:"photos"`
}

//easyjson:json
type PostWithAuthor struct {
	Post      Post         `json:"post"`
	Author    ShortProfile `json:"author"`
	Community *Community   `json:"community,omitempty"` // Информация о сообществе для постов в сообществах
}

//easyjson:json
type PostCreateRequest struct {
	Text        string   `json:"text" valid:"optional,length(0|4096)"`
	CommunityID *int32   `json:"communityID" valid:"optional"` // Новое поле
	Attachments []string `json:"attachments" valid:"optional"`
	Photos      []string `json:"photos" valid:"optional"`
}

//easyjson:json
type PostUpdateRequest struct {
	Text        string   `json:"text" valid:"optional,length(0|4096)"`
	Attachments []string `json:"attachments" valid:"optional"`
	Photos      []string `json:"photos" valid:"optional"`
}

//easyjson:json
type PostFeedItem struct {
	Post        Post          `json:"post"`
	Author      *ShortProfile `json:"author,omitempty"`    // Для постов пользователей
	Community   *Community    `json:"community,omitempty"` // Для постов сообществ
	IsCommunity bool          `json:"isCommunity"`         // Флаг, указывающий тип поста
}

type PostService interface {
	// Получение постов с пагинацией (включая посты из сообществ)
	PostsPaginate(ctx context.Context, userID int32, params PaginateQueryParams) ([]PostView, error)

	// Получение поста по ID
	GetPost(ctx context.Context, userID int32, postID uint) (*PostView, error)

	// Создание поста
	CreatePost(ctx context.Context, userID int32, text string, communityID *int32, attachmentFiles []*File, photoFiles []*File) (*Post, error)

	// Обновление поста
	UpdatePost(ctx context.Context, postID uint, userID int32, text string, attachmentFiles []*File, photoFiles []*File) error

	// Удаление поста
	DeletePost(ctx context.Context, postID uint, userID int32) error

	// Получение постов пользователя
	GetUserPosts(ctx context.Context, selfUserID int32, userID uint, params PaginateQueryParams) ([]PostView, error)

	// Получение постов сообщества
	GetCommunityPosts(ctx context.Context, userID int32, communityID int32, params PaginateQueryParams) ([]PostView, error)

	// Лайк/дизлайк поста
	UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int32) error
}

type PostStore interface {
	// Получение постов с пагинацией
	PostsPaginatedList(ctx context.Context, userID, limit, offset int32) ([]PostDB, error)

	// Получение поста по ID
	GetPostByID(ctx context.Context, userID int32, id uint) (*PostDB, error)

	// Создание поста
	CreatePost(ctx context.Context, post *Post) error

	// Обновление поста
	UpdatePost(ctx context.Context, post *Post) error

	// Удаление поста
	DeletePost(ctx context.Context, id uint, authorID uint) error

	// Получение постов пользователя
	GetPostsByUser(ctx context.Context, selfUserID int32, userID uint, limit, offset int32) ([]PostDB, error)

	// Получение постов сообщества
	GetCommunityPosts(ctx context.Context, userID int32, communityID int32, limit, offset int32) ([]PostDB, error)

	// Лайк/дизлайк поста
	UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int32) error
}
