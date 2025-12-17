package domain

import (
	"context"
	"time"
)

//easyjson:json
type Comment struct {
	ID        int32     `json:"id"`
	PostID    int32     `json:"postID"`
	AuthorID  int32     `json:"authorID"`
	ParentID  *int32    `json:"parentID,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

//easyjson:json
type CommentView struct {
	ID           int32     `json:"id"`
	PostID       int32     `json:"postID"`
	AuthorID     int32     `json:"authorID"`
	AuthorName   string    `json:"authorName"`
	AuthorAvatar *string   `json:"authorAvatar,omitempty"`
	ParentID     *int32    `json:"parentID,omitempty"`
	Text         string    `json:"text"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

//easyjson:json
type CommentCreateRequest struct {
	PostID int32  `json:"postID" valid:"required"`
	Text   string `json:"text" valid:"required,length(1|4096)"`
}

//easyjson:json
type CommentResponse struct {
	Message string       `json:"message"`
	Comment *CommentView `json:"comment"`
}

//easyjson:json
type CommentCountResponse struct {
	Count int32 `json:"count"`
}

//easyjson:json
type CommentViews []CommentView

//easyjson:json
type CommentUpdateRequest struct {
	Text string `json:"text" valid:"required,length(1|4096)"`
}

// CommentService - интерфейс сервиса комментариев
type CommentService interface {
	CreateComment(ctx context.Context, userID int32, req CommentCreateRequest) (*CommentView, error)
	GetComment(ctx context.Context, userID int32, commentID int32) (*CommentView, error)
	GetPostComments(ctx context.Context, userID int32, postID int32, params PaginateQueryParams) ([]CommentView, error)
	UpdateComment(ctx context.Context, commentID int32, userID int32, text string) error
	DeleteComment(ctx context.Context, commentID int32, userID int32) error
	GetPostCommentsCount(ctx context.Context, postID int32) (int32, error)
}

// CommentStore - интерфейс хранилища комментариев
type CommentStore interface {
	CreateComment(ctx context.Context, comment *Comment) error
	GetCommentByID(ctx context.Context, id int32) (*Comment, error)
	GetCommentsByPost(ctx context.Context, postID int32, limit, offset int32) ([]Comment, error)
	UpdateComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, id int32, authorID int32) error
	GetPostCommentsCount(ctx context.Context, postID int32) (int32, error)
}
