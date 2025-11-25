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
func ensureUserOrder(userID1, userID2 int32) (int32, int32) {
	if userID1 < userID2 {
		return userID1, userID2
	}
	return userID2, userID1
}

// CreateFriendship создает запрос в друзья
func (store *DBFriendStore) CreateFriendship(ctx context.Context, actionUserID, targetUserID int32) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CreateFriendship",
		zap.Int32("actionUserID", actionUserID),
		zap.Int32("targetUserID", targetUserID))

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
func (store *DBFriendStore) GetFriendship(ctx context.Context, userID1, userID2 int32) (*domain.Friendship, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetFriendship", zap.Int32("userID1", userID1), zap.Int32("userID2", userID2))

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

// UpdateFriendshipStatus обновляет статус дружбы. ТEБЕ кинули pending. Ты можешь accepted, rejected, blocked и ТЫ actionUserID
func (store *DBFriendStore) UpdateFriendshipStatus(ctx context.Context, actionUserID, targetUserID int32, status domain.FriendshipStatus) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateFriendshipStatus",
		zap.Int32("actionUserID", actionUserID),
		zap.Int32("targetUserID", targetUserID),
		zap.String("status", string(status)))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	firstUserID, secondUserID := ensureUserOrder(actionUserID, targetUserID)

	query := `
		UPDATE friend_relationships
		SET action_user_id = $1, status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE first_user_id = $3 AND second_user_id = $4
	`

	dblogger = dblogger.With(zap.String("query", query))
	result, err := store.db.ExecContext(ctx, query, actionUserID, status, firstUserID, secondUserID)
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
func (store *DBFriendStore) GetUserFriends(ctx context.Context, userID int32, limit, offset int32) ([]int32, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetUserFriends", zap.Int32("userID", userID), zap.Int32("offset", offset), zap.Int32("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	// Запрос для получения друзей с пагинацией
	query := `
SELECT CASE 
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS friend_id
FROM friend_relationships fr
WHERE (fr.first_user_id = $1 OR fr.second_user_id = $1)
  AND fr.status = 'accepted'
ORDER BY friend_id
LIMIT $2 OFFSET $3;
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query user friends", zap.Error(err))
		return nil, fmt.Errorf("failed to query user friends: %w", err)
	}
	defer rows.Close()

	var friendIDs []int32
	for rows.Next() {
		var friendID int32
		err := rows.Scan(
			&friendID,
		)
		if err != nil {
			dblogger.Error("Failed to scan friend profile", zap.Error(err))
			return nil, fmt.Errorf("failed to scan friend profile: %w", err)
		}
		friendIDs = append(friendIDs, friendID)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating friend rows", zap.Error(err))
		return nil, fmt.Errorf("Error iterating friend rows:%w", err)
	}

	dblogger.Info("User friends retrieved successfully",
		zap.Int("friendsCount", len(friendIDs)))

	return friendIDs, nil
}

func (store *DBFriendStore) GetAllUsers(ctx context.Context, userID int32) ([]int32, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetAllUsers",
		zap.Int32("userID", userID),
	)
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	// Исправленный запрос - убираем условие AND fr.first_user_id IS NULL
	query := `
SELECT CASE 
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS friend_id
FROM friend_relationships fr
    WHERE (fr.first_user_id = $1 or fr.second_user_id = $1)
ORDER BY friend_id
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID)
	if err != nil {
		dblogger.Error("Failed to query all users", zap.Error(err))
		return nil, fmt.Errorf("failed to query all users: %w", err)
	}
	defer rows.Close()

	var friendIDs []int32
	for rows.Next() {
		var friendID int32
		err := rows.Scan(
			&friendID,
		)
		if err != nil {
			dblogger.Error("Failed to scan friend profile", zap.Error(err))
			return nil, fmt.Errorf("failed to scan friend profile: %w", err)
		}
		friendIDs = append(friendIDs, friendID)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating user rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	dblogger.Info("All users retrieved successfully",
		zap.Int("usersCount", len(friendIDs)))

	return friendIDs, nil
}

// GetFriendshipRequests получает входящие запросы в друзья с профилями (с пагинацией) ВОЗРАСТ
func (store *DBFriendStore) GetFriendshipRequests(ctx context.Context, userID int32, limit, offset int32) ([]int32, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetFriendshipRequests",
		zap.Int32("userID", userID),
		zap.Int32("offset", offset),
		zap.Int32("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
SELECT CASE 
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS friend_id
		FROM friend_relationships fr
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

	var friendIDs []int32
	for rows.Next() {
		var friendID int32
		err := rows.Scan(
			&friendID,
		)
		if err != nil {
			dblogger.Error("Failed to scan friend profile", zap.Error(err))
			return nil, fmt.Errorf("failed to scan friend profile: %w", err)
		}
		friendIDs = append(friendIDs, friendID)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating request rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating request rows: %w", err)
	}

	dblogger.Info("Friendship requests retrieved successfully",
		zap.Int("requestsCount", len(friendIDs)))

	return friendIDs, nil
}

// GetSentRequests получает отправленные запросы в друзья (с пагинацией)
func (store *DBFriendStore) GetSentRequests(ctx context.Context, userID int32, limit, offset int32) ([]int32, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetSentRequests",
		zap.Int32("userID", userID),
		zap.Int32("offset", offset),
		zap.Int32("limit", limit))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
SELECT CASE 
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS friend_id
		FROM friend_relationships fr
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

	var friendIDs []int32
	for rows.Next() {
		var friendID int32
		err := rows.Scan(
			&friendID,
		)
		if err != nil {
			dblogger.Error("Failed to scan friend profile", zap.Error(err))
			return nil, fmt.Errorf("failed to scan friend profile: %w", err)
		}
		friendIDs = append(friendIDs, friendID)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating sent request rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating sent request rows: %w", err)
	}

	dblogger.Info("Sent requests retrieved successfully",
		zap.Int("requestsCount", len(friendIDs)))

	return friendIDs, nil
}

// DeleteFriendship удаляет запись о дружбе.
func (store *DBFriendStore) DeleteFriendship(ctx context.Context, userID1, userID2 int32) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start DeleteFriendship", zap.Int32("userID1", userID1), zap.Int32("userID2", userID2))

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
func (store *DBFriendStore) AreFriends(ctx context.Context, userID1, userID2 int32) (bool, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start AreFriends", zap.Int32("userID1", userID1), zap.Int32("userID2", userID2))

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
func (store *DBFriendStore) GetFriendshipStatus(ctx context.Context, userID1, userID2 int32) (domain.FriendshipStatus, error) {
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
func (store *DBFriendStore) CountUserRelations(ctx context.Context, userID int32) (*domain.UserRelationsCounts, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CountUserRelations",
		zap.Int32("userID", userID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
WITH counts AS (
    SELECT 
        COUNT(*) FILTER (WHERE status = 'accepted') AS accepted_count,
        COUNT(*) FILTER (WHERE status = 'pending' AND action_user_id != $1) AS pending_count,
        COUNT(*) FILTER (WHERE status = 'pending' AND action_user_id = $1) AS sent_count,
        COUNT(*) FILTER (WHERE status = 'rejected' AND action_user_id = $1) AS rejected_by_me_count,
        COUNT(*) FILTER (WHERE status = 'rejected' AND action_user_id != $1) AS rejected_by_other_count,
        COUNT(*) FILTER (WHERE status = 'blocked') AS blocked_count
    FROM friend_relationships
    WHERE first_user_id = $1 OR second_user_id = $1
)
SELECT 
	accepted_count,
	pending_count + rejected_by_me_count,
	sent_count + rejected_by_other_count,
	blocked_count
FROM counts;
    `

	var counts domain.UserRelationsCounts
	err := store.db.QueryRowContext(ctx, query, userID).Scan(
		&counts.Accepted,
		&counts.Pending,
		&counts.Sent,
		&counts.Blocked,
	)
	if err != nil {
		dblogger.Error("Failed to count user relations", zap.Error(err))
		return nil, fmt.Errorf("failed to count user relations: %w", err)
	}

	dblogger.Info("User relations counted successfully", zap.Any("counts", counts))
	return &counts, nil
}

func (store *DBFriendStore) GetUserIDsByFriendType(ctx context.Context, userID int32, fType domain.FriendshipCountType) ([]int32, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetShortProfilesBySearchIDSAndFriendType")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	var statusClause string
	switch fType {
	case domain.CountNotFriends:
		statusClause = `1=1`
	case domain.CountAccepted:
		statusClause = `fr.status = 'accepted'`
	case domain.CountPending:
		statusClause = `(fr.status = 'pending' AND fr.action_user_id != $1) OR (fr.status = 'rejected' AND fr.action_user_id = $1)`
	case domain.CountSent:
		statusClause = `(fr.status = 'pending' AND fr.action_user_id = $1) OR (fr.status = 'rejected' AND fr.action_user_id != $1)`
	case domain.CountBlocked:
		statusClause = `fr.status = 'blocked'`
	default:
		return nil, fmt.Errorf("unknown statusType: %s", fType)
	}

	query := `
SELECT CASE
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS related_user_id
FROM friend_relationships fr
WHERE (fr.first_user_id = $1 OR fr.second_user_id = $1)
  AND (` + statusClause + `)
`

	rows, err := store.db.QueryContext(ctx, query, userID)
	dblogger = dblogger.With(zap.Int32("UserID", userID), zap.String("type", string(fType)))
	if err != nil {
		dblogger.Error("Failed to get rows", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var ids []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			dblogger.Error("Failed to scan row", zap.Error(err))
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		dblogger.Error("rows error", zap.Error(err))
		return nil, err
	}

	return ids, nil
}
