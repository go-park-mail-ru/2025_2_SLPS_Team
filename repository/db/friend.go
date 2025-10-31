package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
	"project/internal/service"
	"time"

	"go.uber.org/zap"
)

type DBFriendStore struct {
	db *sql.DB
}

func NewDBFriendStore(db *sql.DB) domain.FriendStore {
	return &DBFriendStore{db: db}
}

// ensureUserOrder гарантирует правильный порядок пользователей
func ensureUserOrder(userID1, userID2 int) (int, int) {
	if userID1 < userID2 {
		return userID1, userID2
	}
	return userID2, userID1
}

// CreateFriendship создает запрос в друзья
func (store *DBFriendStore) CreateFriendship(ctx context.Context, userID1, userID2 int) error {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CreateFriendship", zap.Int("userID1", userID1), zap.Int("userID2", userID2))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	firstUserID, secondUserID := ensureUserOrder(userID1, userID2)

	query := `
		INSERT INTO friend_relationships (first_user_id, second_user_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (first_user_id, second_user_id) 
		DO UPDATE SET status = $3, updated_at = CURRENT_TIMESTAMP
	`

	dblogger = dblogger.With(zap.String("query", query))
	_, err := store.db.ExecContext(ctx, query, firstUserID, secondUserID, domain.FriendshipPending)
	if err != nil {
		dblogger.Error("Failed to create friendship", zap.Error(err))
		return fmt.Errorf("failed to create friendship: %w", err)
	}

	dblogger.Info("Friendship created/updated successfully")
	return nil
}

// GetFriendship получает информацию о дружбе
func (store *DBFriendStore) GetFriendship(ctx context.Context, userID1, userID2 int) (*domain.Friendship, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetFriendship", zap.Int("userID1", userID1), zap.Int("userID2", userID2))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	firstUserID, secondUserID := ensureUserOrder(userID1, userID2)

	query := `
		SELECT first_user_id, second_user_id, status, created_at, updated_at
		FROM friend_relationships 
		WHERE first_user_id = $1 AND second_user_id = $2
	`

	dblogger = dblogger.With(zap.String("query", query))
	var friendship domain.Friendship
	err := store.db.QueryRowContext(ctx, query, firstUserID, secondUserID).Scan(
		&friendship.FirstUserID,
		&friendship.SecondUserID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Friendship not found")
		return nil, domain.ErrNotFound
	}

	if err != nil {
		dblogger.Error("Failed to get friendship", zap.Error(err))
		return nil, fmt.Errorf("failed to get friendship: %w", err)
	}

	dblogger.Info("Friendship retrieved successfully")
	return &friendship, nil
}
