package domain

import (
	"context"
	"mime/multipart"
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
	LikeCount   int      `json:"likeCount"`
	IsLiked     bool     `json:"isLiked"`
}
type PostWithShortUser struct {
	Post   Post         `json:"post"`
	Author ShortProfile `json:"author"`
}

// PostCreateRequest - запрос на создание поста для валидации
type PostCreateRequest struct {
	Text        string   `json:"text" valid:"optional,length(0|4096)"`
	Attachments []string `json:"attachments" valid:"optional"`
	Photos      []string `json:"photos" valid:"optional"`
}

// PostUpdateRequest - запрос на обновление поста для валидации
type PostUpdateRequest struct {
	Text        string   `json:"text" valid:"optional,length(0|4096)"`
	Attachments []string `json:"attachments" valid:"optional"`
	Photos      []string `json:"photos" valid:"optional"`
}

type PostService interface {
	PostsPaginate(ctx context.Context, userID int, params PaginateQueryParams) ([]PostWithShortUser, error)
	GetPost(ctx context.Context, userID int, postID uint) (*Post, error)
	CreatePost(ctx context.Context, userID int, text string, attachmentFiles []*multipart.FileHeader, photoFiles []*multipart.FileHeader) (*Post, error)
	UpdatePost(ctx context.Context, postID uint, userID int, text string, attachmentFiles []*multipart.FileHeader, photoFiles []*multipart.FileHeader) error
	DeletePost(ctx context.Context, postID uint, userID int) error
	GetUserPosts(ctx context.Context, selfUserID int, userID uint, params PaginateQueryParams) ([]Post, error)
	UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int) error
}

type PostStore interface {
	// Получение постов с пагинацией
	PostsPaginatedList(ctx context.Context, userID, limit, offset int) ([]PostWithShortUser, error)
	// Получение поста по ID
	GetPostByID(ctx context.Context, userID int, id uint) (*Post, error)
	// Создание поста
	CreatePost(ctx context.Context, post *Post) error
	// Обновление поста
	UpdatePost(ctx context.Context, post *Post) error
	// Удаление поста
	DeletePost(ctx context.Context, id uint, authorID uint) error
	// Получение постов пользователя
	GetPostsByUser(ctx context.Context, selfUserID int, userID uint, limit, offset int) ([]Post, error)

	UpdateLikeOnPostByUserID(ctx context.Context, userID, postID int) error
}
