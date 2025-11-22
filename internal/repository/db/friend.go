package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
	"time"

	"github.com/lib/pq"
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

// UpdateFriendshipStatus обновляет статус дружбы. ТEБЕ кинули pending. Ты можешь accepted, rejected, blocked и ТЫ actionUserID
func (store *DBFriendStore) UpdateFriendshipStatus(ctx context.Context, actionUserID, targetUserID int, status domain.FriendshipStatus) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateFriendshipStatus",
		zap.Int("actionUserID", actionUserID),
		zap.Int("targetUserID", targetUserID),
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
		SELECT p.user_id, p.first_name || ' '||p.last_name, p.avatar_path, p.dob
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
			&friend.Dob,
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

	// Исправленный запрос - убираем условие AND fr.first_user_id IS NULL
	query := `
		SELECT 
			p.user_id,
            COALESCE(p.first_name || ' ' || p.last_name, '') AS full_name,
            COALESCE(p.avatar_path, '') AS avatar_path,
            p.dob
        FROM profiles p
        WHERE p.user_id != $1
          AND NOT EXISTS (
              SELECT 1
              FROM friend_relationships fr
              WHERE (fr.first_user_id = $1 AND fr.second_user_id = p.user_id)
                 OR (fr.first_user_id = p.user_id AND fr.second_user_id = $1)
          )
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
			&user.Dob,
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

// GetFriendshipRequests получает входящие запросы в друзья с профилями (с пагинацией) ВОЗРАСТ
func (store *DBFriendStore) GetFriendshipRequests(ctx context.Context, userID int, limit, offset int) ([]domain.ShortProfile, error) {
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

	query := `
		SELECT 
			p.user_id, 
			p.first_name || ' ' || p.last_name as full_name, 
			p.avatar_path as avatar_path,
			p.dob
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

	friends := []domain.ShortProfile{}
	for rows.Next() {
		var friend domain.ShortProfile
		err := rows.Scan(
			&friend.UserID,
			&friend.FullName,
			&friend.AvatarPath,
			&friend.Dob,
		)
		if err != nil {
			dblogger.Error("Failed to scan friend profile", zap.Error(err))
			return nil, fmt.Errorf("failed to scan friend profile: %w", err)
		}
		friends = append(friends, friend)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating request rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating request rows: %w", err)
	}

	dblogger.Info("Friendship requests retrieved successfully",
		zap.Int("requestsCount", len(friends)))

	return friends, nil
}

// GetSentRequests получает отправленные запросы в друзья (с пагинацией)
func (store *DBFriendStore) GetSentRequests(ctx context.Context, userID int, limit, offset int) ([]domain.ShortProfile, error) {
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

	query := `
		SELECT 
			p.user_id, 
			COALESCE(p.first_name || ' ' || p.last_name, '') as full_name, 
			COALESCE(p.avatar_path, '') as avatar_path,
			p.dob
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

	friends := []domain.ShortProfile{}
	for rows.Next() {
		var friend domain.ShortProfile
		err := rows.Scan(
			&friend.UserID,
			&friend.FullName,
			&friend.AvatarPath,
			&friend.Dob,
		)
		if err != nil {
			dblogger.Error("Failed to scan friend profile", zap.Error(err))
			return nil, fmt.Errorf("failed to scan friend profile: %w", err)
		}
		friends = append(friends, friend)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Error iterating sent request rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating sent request rows: %w", err)
	}

	dblogger.Info("Sent requests retrieved successfully",
		zap.Int("requestsCount", len(friends)))

	return friends, nil
}

// DeleteFriendship удаляет запись о дружбе.
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
func (store *DBFriendStore) CountUserRelations(ctx context.Context, userID int) (*domain.UserRelationsCounts, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CountUserRelations",
		zap.Int("userID", userID))

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

func (store *DBFriendStore) GetShortProfilesBySearchIDSAndFriendType(ctx context.Context, userID int, fType domain.FriendshipCountType, targetIDs []int, limit, offset int) ([]domain.ShortProfile, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "friendStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetShortProfilesBySearchIDSAndFriendType")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	if len(targetIDs) == 0 {
		return nil, nil
	}

	var whereClause string
	var query string
	if fType == domain.CountNotFriends {
		query = `
        SELECT p.user_id,
               COALESCE(p.first_name || ' ' || p.last_name, '') AS full_name,
               COALESCE(p.avatar_path, '') AS avatar_path,
               p.dob
        FROM profiles p
        WHERE p.user_id != $1
          AND p.user_id = ANY($2)
          AND NOT EXISTS (
              SELECT 1
              FROM friend_relationships fr
              WHERE (fr.first_user_id = $1 AND fr.second_user_id = p.user_id)
                 OR (fr.first_user_id = p.user_id AND fr.second_user_id = $1)
          )
        LIMIT $3 OFFSET $4
    `
	} else {
		switch fType {
		case domain.CountAccepted:
			whereClause = `
            (fr.first_user_id = $1 OR fr.second_user_id = $1)
            AND fr.status = 'accepted'
        `
		case domain.CountPending:
			whereClause = `
            (fr.first_user_id = $1 OR fr.second_user_id = $1)
            AND (
                (fr.status = 'pending' AND fr.action_user_id != $1)
                OR (fr.status = 'rejected' AND fr.action_user_id = $1)
            )
        `
		case domain.CountSent:
			whereClause = `
            (fr.first_user_id = $1 OR fr.second_user_id = $1)
            AND (
                (fr.status = 'pending' AND fr.action_user_id = $1)
                OR (fr.status = 'rejected' AND fr.action_user_id != $1)
            )
        `
		case domain.CountBlocked:
			whereClause = `
            (fr.first_user_id = $1 OR fr.second_user_id = $1)
            AND fr.status = 'blocked'
        `

		default:
			return nil, fmt.Errorf("unknown statusType: %s", fType)
		}

		query = `
        SELECT p.user_id,
               COALESCE(p.first_name || ' ' || p.last_name, '') AS full_name,
               COALESCE(p.avatar_path, '') AS avatar_path,
               p.dob
        FROM friend_relationships fr
        JOIN profiles p ON p.user_id = CASE
            WHEN fr.first_user_id = $1 THEN fr.second_user_id
            ELSE fr.first_user_id
        END
        WHERE ` + whereClause + `
          AND (fr.first_user_id = ANY($2) OR fr.second_user_id = ANY($2))
			LIMIT $3 OFFSET $4
    `
	}
	rows, err := store.db.QueryContext(ctx, query, userID, pq.Array(targetIDs), limit, offset)
	dblogger = dblogger.With(zap.Int("UserID", userID), zap.String("type", string(fType)))
	if err != nil {
		dblogger.Error("Failed to get rows", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var profiles []domain.ShortProfile
	for rows.Next() {
		var p domain.ShortProfile
		if err := rows.Scan(&p.UserID, &p.FullName, &p.AvatarPath, &p.Dob); err != nil {
			dblogger.Error("Failed to scan row", zap.Error(err))
			return nil, err
		}
		profiles = append(profiles, p)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("rows error", zap.Error(err))
		return nil, err
	}

	return profiles, nil
}
