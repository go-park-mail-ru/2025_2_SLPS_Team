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
	FriendShipBlocked  FriendshipStatus = "blocked"
)

type Friendship struct {
	FirstUserID   int              `json:"firstUserID"`
	SecondUseerID int              `json:"secondUserID"`
	Status        FriendshipStatus `json:"status"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

// FriendshipWithProfile добавляет информацию о профиле друга
type FriendshipWithProfile struct {
	Friendship
	Friend ShortProfile `json:"friend"`
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

type FriendStore interface {
	//Основные методы CRUD
	CreateFriendship(ctx context.Context, firstUserID, secondUserID int) error
	GetFriendship(ctx context.Context, firstUserID, secondUserID int) (*Friendship, error)
	UpdateFriendshipStatus(ctx context.Context, firstUserID, secondUserID int, status FriendshipStatus) error
	DeleteFriendship(ctx context.Context, firstUserID, secondUserId int) error

	//Получение списков
	GetUserFriends(ctx context.Context, firstUserID, secondUserID int) ([]ShortProfile, error)
	GetFriendshipByStatus(ctx context.Context, userID int, status FriendshipStatus) ([]FriendshipWithProfile, error)
	GetFriendshipRequests(ctx context.Context, userID int) ([]FriendshipWithProfile, error)

	//Вспомогательные методы
	AreFriends(ctx context.Context, userID1, userID2 int) (bool, error)
	GetFriendshipStatus(ctx context.Context, userID1, userID2 int) (FriendshipStatus, error)
}
