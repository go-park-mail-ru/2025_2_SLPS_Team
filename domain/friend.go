package domain

import (
	"context"

	"time"
)

type FriendshipStatus string

const (
	FriendshipPending  FriendshipStatus = "pending"
	FriendshipAccepted FriendshipStatus = "accepted"
	FriendshipRejected FriendshipStatus = "rejected"
	FriendshipBlocked  FriendshipStatus = "blocked"
	FriendshipNone     FriendshipStatus = "none"
)

type FriendshipCountType string

const (
	CountPending    FriendshipCountType = "pending"
	CountAccepted   FriendshipCountType = "accepted"
	CountRejected   FriendshipCountType = "rejected"
	CountBlocked    FriendshipCountType = "blocked"
	CountSent       FriendshipCountType = "sent"
	CountNotFriends FriendshipCountType = "notFriends"
)

type UserRelationsCounts struct {
	Accepted int32 `json:"countAccepted"`
	Pending  int32 `json:"countPending"`
	Sent     int32 `json:"countSent"`
	Blocked  int32 `json:"CountBlocked"`
}

//easyjson:json
type Friendship struct {
	ID           int32            `json:"id"`
	FirstUserID  int32            `json:"firstUserID"`
	SecondUserID int32            `json:"secondUserID"`
	ActionUserID int32            `json:"actionUserID"` // Кто отправил запрос
	Status       FriendshipStatus `json:"status"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

//easyjson:json
type FriendResponse struct {
	UserID     int32            `json:"userID"`
	FirstName  string           `json:"firstName"`
	LastName   string           `json:"lastName"`
	AvatarPath *string          `json:"avatarPath"`
	Status     FriendshipStatus `json:"status,omitempty"`
	CreatedAt  time.Time        `json:"createdAt,omitempty"`
}

//easyjson:json
type FriendsCountResponse struct {
	UserID    int32               `json:"userID"`
	Count     int32               `json:"count"`
	CountType FriendshipCountType `json:"countType,omitempty"`
}

//easyjson:json
type FriendshipStatusResponse struct {
	Status FriendshipStatus `json:"status" example:"pending" enums:"pending,accepted,rejected,blocked"` // Статус дружбы
}
type FriendService interface {
	SendFriendRequest(ctx context.Context, actionUserID, targetUserID int32) error
	AcceptFriendRequest(ctx context.Context, userID, friendID int32) error
	RejectFriendRequest(ctx context.Context, userID, friendID int32) error
	RemoveFriend(ctx context.Context, userID, friendID int32) error
	GetFriends(ctx context.Context, userID int32, params PaginateQueryParams) ([]ShortProfile, error)
	GetAllUsers(ctx context.Context, userID int32, params PaginateQueryParams) ([]ShortProfile, error)

	SearchShortProfilesByFullNameAndRelationType(ctx context.Context, userID int32, params PaginateQueryParams, fullName string, fType FriendshipCountType) ([]ShortProfile, error)
	GetFriendRequests(ctx context.Context, userID int32, params PaginateQueryParams) ([]ShortProfile, error)
	GetSentRequests(ctx context.Context, userID int32, params PaginateQueryParams) ([]ShortProfile, error)

	GetFriendshipStatus(ctx context.Context, userID, friendID int32) (FriendshipStatus, error)
	CountUserRelations(ctx context.Context, userID int32) (*UserRelationsCounts, error)
}

type FriendStore interface {
	// Основные операции CRUD
	CreateFriendship(ctx context.Context, actionUserID, targetUserID int32) error
	GetFriendship(ctx context.Context, userID1, userID2 int32) (*Friendship, error)
	UpdateFriendshipStatus(ctx context.Context, actionUserID, targetUserID int32, status FriendshipStatus) error
	DeleteFriendship(ctx context.Context, userID1, userID2 int32) error

	// Получение списков с пагинацией
	GetUserFriends(ctx context.Context, userID, limit, offset int32) ([]int32, error)
	GetAllUsers(ctx context.Context, userID int32) ([]int32, error)

	GetFriendshipRequests(ctx context.Context, userID, limit, offset int32) ([]int32, error)
	GetSentRequests(ctx context.Context, userID, limit, offset int32) ([]int32, error)
	GetUserIDsByFriendType(ctx context.Context, userID int32, fType FriendshipCountType) ([]int32, error)
	// Вспомогательные методы
	AreFriends(ctx context.Context, userID1, userID2 int32) (bool, error)
	GetFriendshipStatus(ctx context.Context, userID1, userID2 int32) (FriendshipStatus, error)
	CountUserRelations(ctx context.Context, userID int32) (*UserRelationsCounts, error)
}
