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

type Friendship struct {
	ID           int              `json:"id"`
	FirstUserID  int              `json:"firstUserID"`
	SecondUserID int              `json:"secondUserID"`
	ActionUserID int              `json:"actionUserID"` // Кто отправил запрос
	Status       FriendshipStatus `json:"status"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

type ShortProfileWithStatusAndDOB struct {
	ShortProfile
	Status *FriendshipStatus `json:"status"` // Указатель
	Dob    time.Time         `json:"dob"`
}

// FriendshipWithProfile добавляет информацию о профиле друга
type FriendshipWithProfile struct {
	Friendship
	Friend ShortProfile `json:"friend"`
}

type ShortProfileAndDOB struct {
	ShortProfile
	Dob time.Time `json:"dob"`
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
	GetFriends(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortProfileAndDOB, error)
	GetAllUsers(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortProfileWithStatusAndDOB, error)

	SearchShortProfilesByFullNameAndRelationType(ctx context.Context, userID int, params PaginateQueryParams, fullName string, fType FriendshipCountType) ([]ShortProfile, error)
	GetFriendRequests(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortProfileAndDOB, error)
	GetSentRequests(ctx context.Context, userID int, params PaginateQueryParams) ([]ShortProfileAndDOB, error)

	GetFriendshipStatus(ctx context.Context, userID, friendID int) (FriendshipStatus, error)
	CountUserRelations(ctx context.Context, userID int, countType FriendshipCountType) (int, error)
}

type FriendStore interface {
	// Основные операции CRUD
	CreateFriendship(ctx context.Context, actionUserID, targetUserID int) error
	GetFriendship(ctx context.Context, userID1, userID2 int) (*Friendship, error)
	UpdateFriendshipStatus(ctx context.Context, actionUserID, targetUserID int, status FriendshipStatus) error
	DeleteFriendship(ctx context.Context, userID1, userID2 int) error

	// Получение списков с пагинацией
	GetUserFriends(ctx context.Context, userID, limit, offset int) ([]ShortProfileAndDOB, error)
	GetAllUsers(ctx context.Context, userID int, limit, offset int) ([]ShortProfileWithStatusAndDOB, error)

	GetFriendshipRequests(ctx context.Context, userID, limit, offset int) ([]ShortProfileAndDOB, error)
	GetSentRequests(ctx context.Context, userID, limit, offset int) ([]ShortProfileAndDOB, error)
	GetShortProfilesBySearchIDSAndFriendType(ctx context.Context, userID int, fType FriendshipCountType, targetIDs []int, limit, offset int) ([]ShortProfile, error)
	// Вспомогательные методы
	AreFriends(ctx context.Context, userID1, userID2 int) (bool, error)
	GetFriendshipStatus(ctx context.Context, userID1, userID2 int) (FriendshipStatus, error)
	CountUserRelations(ctx context.Context, userID int, countType FriendshipCountType) (int, error)
}
