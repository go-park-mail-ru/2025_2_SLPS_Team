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
	Accepted int `json:"countAccepted"`
	Pending  int `json:"countPending"`
	Sent     int `json:"countSent"`
	Blocked  int `json:"CountBlocked"`
}

type Friendship struct {
	ID           int              `json:"id"`
	FirstUserID  int              `json:"firstUserID"`
	SecondUserID int              `json:"secondUserID"`
	ActionUserID int              `json:"actionUserID"` // Кто отправил запрос
	Status       FriendshipStatus `json:"status"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// FriendResponse - ответ для API с информацией о друге
type FriendResponse struct {
	UserID     int              `json:"userID"`
	FirstName  string           `json:"firstName"`
	LastName   string           `json:"lastName"`
	AvatarPath *string          `json:"avatarPath"`
	Status     FriendshipStatus `json:"status,omitempty"`
	CreatedAt  time.Time        `json:"createdAt,omitempty"`
}

type FriendsCountResponse struct {
	UserID    int                 `json:"userID"`
	Count     int                 `json:"count"`
	CountType FriendshipCountType `json:"countType,omitempty"`
}

type FriendService interface {
	SendFriendRequest(ctx context.Context, actionUserID, targetUserID int) error
	AcceptFriendRequest(ctx context.Context, userID, friendID int) error
	RejectFriendRequest(ctx context.Context, userID, friendID int) error
	RemoveFriend(ctx context.Context, userID, friendID int) error
	GetFriends(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortProfile, error)
	GetAllUsers(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortProfile, error)

	SearchShortProfilesByFullNameAndRelationType(ctx context.Context, userID int, params PaginateQueryParams, fullName string, fType FriendshipCountType) ([]ShortProfile, error)
	GetFriendRequests(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortProfile, error)
	GetSentRequests(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortProfile, error)

	GetFriendshipStatus(ctx context.Context, userID, friendID int) (FriendshipStatus, error)
	CountUserRelations(ctx context.Context, userID int) (*UserRelationsCounts, error)
}

type FriendStore interface {
	// Основные операции CRUD
	CreateFriendship(ctx context.Context, actionUserID, targetUserID int) error
	GetFriendship(ctx context.Context, userID1, userID2 int) (*Friendship, error)
	UpdateFriendshipStatus(ctx context.Context, actionUserID, targetUserID int, status FriendshipStatus) error
	DeleteFriendship(ctx context.Context, userID1, userID2 int) error

	// Получение списков с пагинацией
	GetUserFriends(ctx context.Context, userID, limit, offset int) ([]ShortProfile, error)
	GetAllUsers(ctx context.Context, userID int, limit, offset int) ([]ShortProfile, error)

	GetFriendshipRequests(ctx context.Context, userID, limit, offset int) ([]ShortProfile, error)
	GetSentRequests(ctx context.Context, userID, limit, offset int) ([]ShortProfile, error)
	GetShortProfilesBySearchIDSAndFriendType(ctx context.Context, userID int, fType FriendshipCountType, targetIDs []int, limit, offset int) ([]ShortProfile, error)
	// Вспомогательные методы
	AreFriends(ctx context.Context, userID1, userID2 int) (bool, error)
	GetFriendshipStatus(ctx context.Context, userID1, userID2 int) (FriendshipStatus, error)
	CountUserRelations(ctx context.Context, userID int) (*UserRelationsCounts, error)
}
