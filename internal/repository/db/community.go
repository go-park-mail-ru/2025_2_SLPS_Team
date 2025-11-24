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

type DBCommunityStore struct {
	db *sql.DB
}

func NewDBCommunityStore(db *sql.DB) domain.CommunityStore {
	return &DBCommunityStore{db: db}
}

func (store *DBCommunityStore) CreateCommunity(ctx context.Context, community *domain.Community) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CreateCommunity", zap.Int("creatorID", community.CreatorID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		INSERT INTO communities (name, description, creator_id, avatar_path, cover_path)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	dblogger = dblogger.With(zap.String("query", query))
	err := store.db.QueryRowContext(ctx, query,
		community.Name,
		community.Description,
		community.CreatorID,
		community.AvatarPath,
		community.CoverPath,
	).Scan(
		&community.ID,
		&community.CreatedAt,
		&community.UpdatedAt,
	)

	if err != nil {
		dblogger.Error("Failed to create community", zap.Error(err))
		return fmt.Errorf("failed to create community: %w", err)
	}

	dblogger.Info("Community created successfully", zap.Int("communityID", community.ID))
	return nil
}

func (store *DBCommunityStore) UpdateCommunity(ctx context.Context, community *domain.Community) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateCommunity", zap.Int("communityID", community.ID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		UPDATE communities 
		SET name = $1, description = $2, avatar_path = $3, cover_path = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND creator_id = $6
		RETURNING updated_at
	`

	dblogger = dblogger.With(zap.String("query", query))
	err := store.db.QueryRowContext(ctx, query,
		community.Name,
		community.Description,
		community.AvatarPath,
		community.CoverPath,
		community.ID,
		community.CreatorID,
	).Scan(&community.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Community not found for update")
		return domain.ErrNotFound
	}
	if err != nil {
		dblogger.Error("Failed to update community", zap.Error(err))
		return fmt.Errorf("failed to update community: %w", err)
	}

	dblogger.Info("Community updated successfully")
	return nil
}

func (store *DBCommunityStore) DeleteCommunity(ctx context.Context, id int, creatorID int) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start DeleteCommunity", zap.Int("communityID", id), zap.Int("creatorID", creatorID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `DELETE FROM communities WHERE id = $1 AND creator_id = $2`
	dblogger = dblogger.With(zap.String("query", query))
	result, err := store.db.ExecContext(ctx, query, id, creatorID)
	if err != nil {
		dblogger.Error("Failed to delete community", zap.Error(err))
		return fmt.Errorf("failed to delete community: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		dblogger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		dblogger.Warn("Community not found for deletion")
		return domain.ErrNotFound
	}

	dblogger.Info("Community deleted successfully")
	return nil
}

func (store *DBCommunityStore) GetCommunityByID(ctx context.Context, id int) (*domain.Community, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetCommunityByID", zap.Int("communityID", id))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT 
			c.id, 
			c.name, 
			c.description,
			c.creator_id, 
			c.avatar_path, 
			c.cover_path, 
			c.created_at,
			c.updated_at,
			(SELECT COUNT(*) FROM community_subscriptions WHERE community_id = c.id) as subscribers_count
		FROM communities c
		WHERE c.id = $1
	`

	dblogger = dblogger.With(zap.String("query", query))
	var community domain.Community
	err := store.db.QueryRowContext(ctx, query, id).Scan(
		&community.ID,
		&community.Name,
		&community.Description,
		&community.CreatorID,
		&community.AvatarPath,
		&community.CoverPath,
		&community.CreatedAt,
		&community.UpdatedAt,
		&community.SubscribersCount,
	)

	if errors.Is(err, sql.ErrNoRows) {
		dblogger.Warn("Community not found")
		return nil, domain.ErrNotFound
	}
	if err != nil {
		dblogger.Error("Failed to get community", zap.Error(err))
		return nil, fmt.Errorf("failed to get community: %w", err)
	}

	dblogger.Info("Community retrieved successfully", zap.Int("subscribersCount", community.SubscribersCount))
	return &community, nil
}

func (store *DBCommunityStore) GetUserCommunities(ctx context.Context, userID int, limit, offset int) ([]domain.ShortCommunity, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetUserCommunities", zap.Int("userID", userID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT 
			c.id, 
			c.name, 
			c.description, 
			c.avatar_path,
			(SELECT COUNT(*) FROM community_subscriptions cs2 WHERE cs2.community_id = c.id) as subscribers_count
		FROM communities c
		INNER JOIN community_subscriptions cs ON c.id = cs.community_id
		WHERE cs.user_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query user communities", zap.Error(err))
		return nil, fmt.Errorf("failed to query user communities: %w", err)
	}
	defer rows.Close()

	communities := []domain.ShortCommunity{}
	for rows.Next() {
		var community domain.ShortCommunity
		err := rows.Scan(
			&community.ID,
			&community.Name,
			&community.Description,
			&community.AvatarPath,
			&community.SubscribersCount,
		)
		if err != nil {
			dblogger.Error("Failed to scan community", zap.Error(err))
			return nil, fmt.Errorf("failed to scan community: %w", err)
		}
		communities = append(communities, community)
	}

	dblogger.Info("User communities retrieved successfully", zap.Int("count", len(communities)))
	return communities, nil
}

func (store *DBCommunityStore) GetOtherCommunities(ctx context.Context, userID int, limit, offset int) ([]domain.ShortCommunity, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetOtherCommunities", zap.Int("userID", userID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT 
			c.id, 
			c.name, 
			c.description, 
			c.avatar_path,
			COUNT(cs_all.community_id) as subscribers_count
		FROM communities c
		LEFT JOIN community_subscriptions cs_all ON c.id = cs_all.community_id
		WHERE c.id NOT IN (
			SELECT community_id 
			FROM community_subscriptions 
			WHERE user_id = $1
		)
		GROUP BY c.id, c.name, c.description, c.avatar_path
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query other communities", zap.Error(err))
		return nil, fmt.Errorf("failed to query other communities: %w", err)
	}
	defer rows.Close()

	communities := []domain.ShortCommunity{}
	for rows.Next() {
		var community domain.ShortCommunity
		err := rows.Scan(
			&community.ID,
			&community.Name,
			&community.Description,
			&community.AvatarPath,
			&community.SubscribersCount,
		)
		if err != nil {
			dblogger.Error("Failed to scan community", zap.Error(err))
			return nil, fmt.Errorf("failed to scan community: %w", err)
		}
		communities = append(communities, community)
	}

	dblogger.Info("Other communities retrieved successfully", zap.Int("count", len(communities)))
	return communities, nil
}

func (store *DBCommunityStore) GetCreatedCommunities(ctx context.Context, userID int, limit, offset int) ([]domain.CommunityForMyCommunity, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetCreatedCommunities", zap.Int("userID", userID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT 
			c.id, 
			c.name, 
			c.avatar_path
		FROM communities c
		WHERE c.creator_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to query created communities", zap.Error(err))
		return nil, fmt.Errorf("failed to query created communities: %w", err)
	}
	defer rows.Close()

	communities := []domain.CommunityForMyCommunity{}
	for rows.Next() {
		var community domain.CommunityForMyCommunity
		err := rows.Scan(
			&community.ID,
			&community.Name,
			&community.AvatarPath,
		)
		if err != nil {
			dblogger.Error("Failed to scan community", zap.Error(err))
			return nil, fmt.Errorf("failed to scan community: %w", err)
		}
		communities = append(communities, community)
	}

	dblogger.Info("Created communities retrieved successfully", zap.Int("count", len(communities)))
	return communities, nil
}

func (store *DBCommunityStore) Subscribe(ctx context.Context, communityID int, userID int) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start Subscribe", zap.Int("communityID", communityID), zap.Int("userID", userID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		INSERT INTO community_subscriptions (community_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (community_id, user_id) DO NOTHING
	`

	dblogger = dblogger.With(zap.String("query", query))
	_, err := store.db.ExecContext(ctx, query, communityID, userID)
	if err != nil {
		dblogger.Error("Failed to subscribe", zap.Error(err))
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	dblogger.Info("User subscribed successfully")
	return nil
}

func (store *DBCommunityStore) Unsubscribe(ctx context.Context, communityID int, userID int) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start Unsubscribe", zap.Int("communityID", communityID), zap.Int("userID", userID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `DELETE FROM community_subscriptions WHERE community_id = $1 AND user_id = $2`
	dblogger = dblogger.With(zap.String("query", query))
	result, err := store.db.ExecContext(ctx, query, communityID, userID)
	if err != nil {
		dblogger.Error("Failed to unsubscribe", zap.Error(err))
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		dblogger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		dblogger.Warn("Subscription not found")
		return domain.ErrNotFound
	}

	dblogger.Info("User unsubscribed successfully")
	return nil
}

func (store *DBCommunityStore) IsSubscribed(ctx context.Context, communityID int, userID int) (bool, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "communityStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start IsSubscribed", zap.Int("communityID", communityID), zap.Int("userID", userID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `SELECT EXISTS(SELECT 1 FROM community_subscriptions WHERE community_id = $1 AND user_id = $2)`
	dblogger = dblogger.With(zap.String("query", query))
	var exists bool
	err := store.db.QueryRowContext(ctx, query, communityID, userID).Scan(&exists)
	if err != nil {
		dblogger.Error("Failed to check subscription", zap.Error(err))
		return false, fmt.Errorf("failed to check subscription: %w", err)
	}

	dblogger.Info("Subscription check completed", zap.Bool("isSubscribed", exists))
	return exists, nil
}
