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
	// Основные операции CRUD
	CreateFriendship(ctx context.Context, actionUserID, targetUserID int) error
	GetFriendship(ctx context.Context, userID1, userID2 int) (*Friendship, error)
	UpdateFriendshipStatus(ctx context.Context, userID1, userID2 int, status FriendshipStatus) error
	DeleteFriendship(ctx context.Context, userID1, userID2 int) error

	// Получение списков с пагинацией
	GetUserFriends(ctx context.Context, userID int, page, limit int) ([]ShortProfile, int, error)
	GetFriendshipRequests(ctx context.Context, userID int, page, limit int) ([]FriendshipWithProfile, int, error)
	GetSentRequests(ctx context.Context, userID int, page, limit int) ([]FriendshipWithProfile, int, error)

	// Вспомогательные методы
	AreFriends(ctx context.Context, userID1, userID2 int) (bool, error)
	GetFriendshipStatus(ctx context.Context, userID1, userID2 int) (FriendshipStatus, error)
}
