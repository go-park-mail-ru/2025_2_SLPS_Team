package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
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
func (store *DBFriendStore) CreateFriendship(ctx context.Context, actionUserID, targetUserID int) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CreateFriendship",
		zap.Int("actionUserID", actionUserID),
		zap.Int("targetUserID", targetUserID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	firstUserID, secondUserID := ensureUserOrder(actionUserID, targetUserID)

	query := `
		INSERT INTO friend_relationships (first_user_id, second_user_id, action_user_id, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (first_user_id, second_user_id) 
		DO UPDATE SET status = $4, action_user_id = $3, updated_at = CURRENT_TIMESTAMP
	`

	dblogger = dblogger.With(zap.String("query", query))
	_, err := store.db.ExecContext(ctx, query, firstUserID, secondUserID, actionUserID, domain.FriendshipPending)
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
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetFriendship", zap.Int("userID1", userID1), zap.Int("userID2", userID2))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	firstUserID, secondUserID := ensureUserOrder(userID1, userID2)

	query := `
		SELECT first_user_id, second_user_id, action_user_id, status, created_at, updated_at
		FROM friend_relationships 
		WHERE first_user_id = $1 AND second_user_id = $2
	`

	dblogger = dblogger.With(zap.String("query", query))
	var friendship domain.Friendship
	err := store.db.QueryRowContext(ctx, query, firstUserID, secondUserID).Scan(
		&friendship.FirstUserID,
		&friendship.SecondUserID,
		&friendship.ActionUserID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Friendship not found")
		return nil, domain.ErrFriendshipNotFound
	}

	if err != nil {
		dblogger.Error("Failed to get friendship", zap.Error(err))
		return nil, fmt.Errorf("failed to get friendship: %w", err)
	}

	dblogger.Info("Friendship retrieved successfully")
	return &friendship, nil
}

// UpdateFriendshipStatus обновляет статус дружбы
func (store *DBFriendStore) UpdateFriendshipStatus(ctx context.Context, userID1, userID2 int, status domain.FriendshipStatus) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateFriendshipStatus",
		zap.Int("userID1", userID1),
		zap.Int("userID2", userID2),
		zap.String("status", string(status)))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	firstUserID, secondUserID := ensureUserOrder(userID1, userID2)

	query := `
		UPDATE friend_relationships
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE first_user_id = $2 AND second_user_id = $3
	`

	dblogger = dblogger.With(zap.String("query", query))
	result, err := store.db.ExecContext(ctx, query, status, firstUserID, secondUserID)
	if err != nil {
		dblogger.Error("Failed to update friendship status", zap.Error(err))
		return fmt.Errorf("Failed to update friendship status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		dblogger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("Failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		dblogger.Warn("Friendship not found for update")
		return domain.ErrFriendshipNotFound
	}

	dblogger.Info("Friendship status updated successfully")
	return nil
}

// GetUserFriends получает список друзей пользователя с профилями (с пагинацией)
func (store *DBFriendStore) GetUserFriends(ctx context.Context, userID int, limit, offset int) ([]domain.ShortProfile, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetUserFriends", zap.Int("userID", userID), zap.Int("offset", offset), zap.Int("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	// Запрос для получения друзей с пагинацией
	query := `
		SELECT p.user_id, p.first_name || ' '||p.last_name, p.avatar_path
		FROM profiles p
		WHERE p.user_id IN (
			SELECT CASE 
				WHEN fr.first_user_id = $1 THEN fr.second_user_id 
				ELSE fr.first_user_id
			END as friend_id
			FROM friend_relationships fr
			WHERE (fr.first_user_id = $1 OR fr.second_user_id = $1)
			AND fr.status = 'accepted'
		)
		ORDER BY p.first_name, p.last_name
		LIMIT $2 OFFSET $3
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query user friends", zap.Error(err))
		return nil, fmt.Errorf("failed to query user friends: %w", err)
	}
	defer rows.Close()

	friends := []domain.ShortProfile{}
	for rows.Next() {
		var friend domain.ShortProfile
		err := rows.Scan(
			&friend.UserID,
			&friend.FullName,
			&friend.AvatarPath,
		)
		if err != nil {
			dblogger.Error("Failed to scan friend profile", zap.Error(err))
			return nil, fmt.Errorf("failed to scan friend profile: %w", err)
		}
		friends = append(friends, friend)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating friend rows", zap.Error(err))
		return nil, fmt.Errorf("Error iterating friend rows:%w", err)
	}

	dblogger.Info("User friends retrieved successfully",
		zap.Int("friendsCount", len(friends)))

	return friends, nil
}

func (store *DBFriendStore) GetAllUsers(ctx context.Context, userID int, limit, offset int) ([]domain.ShortProfile, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetAllUsers",
		zap.Int("userID", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
        SELECT user_id, first_name || ' ' || last_name as full_name, avatar_path
        FROM profiles
        WHERE user_id != $1
        ORDER BY user_id
        LIMIT $2 OFFSET $3
    `

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query all users", zap.Error(err))
		return nil, fmt.Errorf("failed to query all users: %w", err)
	}
	defer rows.Close()

	users := []domain.ShortProfile{}
	for rows.Next() {
		var user domain.ShortProfile
		err := rows.Scan(
			&user.UserID,
			&user.FullName,
			&user.AvatarPath,
		)
		if err != nil {
			dblogger.Error("Failed to scan user profile", zap.Error(err))
			return nil, fmt.Errorf("failed to scan user profile: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating user rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	dblogger.Info("All users retrieved successfully",
		zap.Int("usersCount", len(users)))

	return users, nil
}

// GetFriendshipRequests получает входящие запросы в друзья с профилями (с пагинацией)
func (store *DBFriendStore) GetFriendshipRequests(ctx context.Context, userID int, limit, offset int) ([]domain.FriendshipWithProfile, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetFriendshipRequests",
		zap.Int("userID", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	// Запросы где пользователь НЕ является action_user_id (получатель запроса)
	query := `
		SELECT 
			fr.first_user_id, fr.second_user_id, fr.action_user_id, fr.status, fr.created_at, fr.updated_at,
			p.user_id, p.first_name|| ' '||p.last_name, p.avatar_path
		FROM friend_relationships fr
		JOIN profiles p ON p.user_id = fr.action_user_id
		WHERE (fr.first_user_id = $1 OR fr.second_user_id = $1) 
		AND fr.action_user_id != $1 
		AND fr.status = 'pending'
		ORDER BY fr.created_at DESC
		LIMIT $2 OFFSET $3
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query friendship requests", zap.Error(err))
		return nil, fmt.Errorf("Failed to query friendship requests:%w", err)
	}
	defer rows.Close()

	requests := []domain.FriendshipWithProfile{}
	for rows.Next() {
		var request domain.FriendshipWithProfile
		err := rows.Scan(
			&request.FirstUserID,
			&request.SecondUserID,
			&request.ActionUserID,
			&request.Status,
			&request.CreatedAt,
			&request.UpdatedAt,
			&request.Friend.UserID,
			&request.Friend.FullName,
			&request.Friend.AvatarPath,
		)
		if err != nil {
			dblogger.Error("Failed to scan friendship request", zap.Error(err))
			return nil, fmt.Errorf("failed to scan friendship request: %w", err)
		}
		requests = append(requests, request)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating request rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating request rows: %w", err)
	}

	dblogger.Info("Friendship requests retrieved successfully",
		zap.Int("requestsCount", len(requests)))

	return requests, nil
}

// GetSentRequests получает отправленные запросы в друзья (с пагинацией)
func (store *DBFriendStore) GetSentRequests(ctx context.Context, userID int, limit, offset int) ([]domain.FriendshipWithProfile, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetSentRequests",
		zap.Int("userID", userID),
		zap.Int("offset", offset),
		zap.Int("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	// Запросы где пользователь является action_user_id (отправителем запроса)
	query := `
		SELECT 
			fr.first_user_id, fr.second_user_id, fr.action_user_id, fr.status, fr.created_at, fr.updated_at,
			p.user_id, p.first_name ||' '|| p.last_name, p.avatar_path
		FROM friend_relationships fr
		JOIN profiles p ON p.user_id = CASE 
			WHEN fr.first_user_id = $1 THEN fr.second_user_id 
			ELSE fr.first_user_id 
		END
		WHERE fr.action_user_id = $1 AND fr.status = 'pending'
		ORDER BY fr.created_at DESC
		LIMIT $2 OFFSET $3
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query sent requests", zap.Error(err))
		return nil, fmt.Errorf("failed to query sent requests: %w", err)
	}
	defer rows.Close()

	requests := []domain.FriendshipWithProfile{}
	for rows.Next() {
		var request domain.FriendshipWithProfile
		err := rows.Scan(
			&request.FirstUserID,
			&request.SecondUserID,
			&request.ActionUserID,
			&request.Status,
			&request.CreatedAt,
			&request.UpdatedAt,
			&request.Friend.UserID,
			&request.Friend.FullName,
			&request.Friend.AvatarPath,
		)
		if err != nil {
			dblogger.Error("Failed to scan sent request", zap.Error(err))
			return nil, fmt.Errorf("failed to scan sent request: %w", err)
		}
		requests = append(requests, request)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating sent request rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating sent request rows: %w", err)
	}

	dblogger.Info("Sent requests retrieved successfully",
		zap.Int("requestsCount", len(requests)))

	return requests, nil
}

// DeleteFriendship удаляет запись о дружбе
func (store *DBFriendStore) DeleteFriendship(ctx context.Context, userID1, userID2 int) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start DeleteFriendship", zap.Int("userID1", userID1), zap.Int("userID2", userID2))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	firstUserID, secondUserID := ensureUserOrder(userID1, userID2)

	query := `
		DELETE FROM friend_relationships 
		WHERE first_user_id = $1 AND second_user_id = $2
	`

	dblogger = dblogger.With(zap.String("query", query))
	result, err := store.db.ExecContext(ctx, query, firstUserID, secondUserID)
	if err != nil {
		dblogger.Error("Failed to delete friendship", zap.Error(err))
		return fmt.Errorf("failed to delete friendship: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		dblogger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		dblogger.Warn("Friendship not found for deletion")
		return domain.ErrFriendshipNotFound
	}

	dblogger.Info("Friendship deleted successfully")
	return nil
}

// AreFriends проверяет, являются ли пользователи друзьями
func (store *DBFriendStore) AreFriends(ctx context.Context, userID1, userID2 int) (bool, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start AreFriends", zap.Int("userID1", userID1), zap.Int("userID2", userID2))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	firstUserID, secondUserID := ensureUserOrder(userID1, userID2)

	query := `
		SELECT EXISTS(
			SELECT 1 FROM friend_relationships 
			WHERE first_user_id = $1 AND second_user_id = $2 AND status = 'accepted'
		)
	`

	dblogger = dblogger.With(zap.String("query", query))
	var exists bool
	err := store.db.QueryRowContext(ctx, query, firstUserID, secondUserID).Scan(&exists)
	if err != nil {
		dblogger.Error("Failed to check friendship", zap.Error(err))
		return false, fmt.Errorf("failed to check friendship: %w", err)
	}

	dblogger.Info("Friendship check completed", zap.Bool("areFriends", exists))
	return exists, nil
}

// GetFriendshipStatus получает статус дружбы между пользователями
func (store *DBFriendStore) GetFriendshipStatus(ctx context.Context, userID1, userID2 int) (domain.FriendshipStatus, error) {
	friendship, err := store.GetFriendship(ctx, userID1, userID2)
	if err != nil {
		if errors.Is(err, domain.ErrFriendshipNotFound) {
			return "", nil // Нет записи - нет статуса
		}
		return "", err
	}
	return friendship.Status, nil
}

// CountUserRelations подсчитывает количество отношений пользователя по типу
func (store *DBFriendStore) CountUserRelations(ctx context.Context, userID int, countType domain.FriendshipCountType) (int, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CountUserRelations",
		zap.Int("userID", userID),
		zap.String("countType", string(countType)))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	var query string
	var args []interface{}

	switch countType {
	case domain.CountAll:
		query = `
            SELECT COUNT(*)
            FROM friend_relationships
            WHERE (first_user_id = $1 OR second_user_id = $1)
        `
		args = []interface{}{userID}

	case domain.CountAccepted:
		query = `
            SELECT COUNT(*)
            FROM friend_relationships
            WHERE (first_user_id = $1 OR second_user_id = $1)
            AND status = 'accepted'
        `
		args = []interface{}{userID}

	case domain.CountPending:
		// Все pending запросы где пользователь является получателем
		query = `
            SELECT COUNT(*)
            FROM friend_relationships
            WHERE (first_user_id = $1 OR second_user_id = $1)
            AND status = 'pending'
            AND action_user_id != $1
        `
		args = []interface{}{userID}

	case domain.CountSent:
		// Все pending запросы где пользователь является отправителем
		query = `
            SELECT COUNT(*)
            FROM friend_relationships
            WHERE (first_user_id = $1 OR second_user_id = $1)
            AND status = 'pending'
            AND action_user_id = $1
        `
		args = []interface{}{userID}

	case domain.CountReceived:
		// Все pending запросы где пользователь является получателем
		query = `
            SELECT COUNT(*)
            FROM friend_relationships
            WHERE (first_user_id = $1 OR second_user_id = $1)
            AND status = 'pending'
            AND action_user_id != $1
        `
		args = []interface{}{userID}

	case domain.CountBlocked:
		query = `
            SELECT COUNT(*)
            FROM friend_relationships
            WHERE (first_user_id = $1 OR second_user_id = $1)
            AND status = 'blocked'
        `
		args = []interface{}{userID}

	case domain.CountRejected:
		query = `
            SELECT COUNT(*)
            FROM friend_relationships
            WHERE (first_user_id = $1 OR second_user_id = $1)
            AND status = 'rejected'
        `
		args = []interface{}{userID}

	default:
		dblogger.Error("Unknown count type", zap.String("countType", string(countType)))
		return 0, domain.ErrInvalidInput
	}

	dblogger = dblogger.With(zap.String("query", query))
	var count int
	err := store.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		dblogger.Error("Failed to count user relations", zap.Error(err))
		return 0, fmt.Errorf("failed to count user relations: %w", err)
	}

	dblogger.Info("User relations counted successfully",
		zap.Int("count", count),
		zap.String("countType", string(countType)))
	return count, nil
}
