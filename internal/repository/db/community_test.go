package db

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"project/domain"
)

func newCommunityStoreMock(t *testing.T) (*DBCommunityStore, sqlmock.Sqlmock, *sql.DB) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err, "failed to create sqlmock")
	store := NewDBCommunityStore(dbConn).(*DBCommunityStore)
	return store, mock, dbConn
}

// Helper to create context with logger
var ava = "cover"

func TestCreateCommunity_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	community := &domain.Community{
		Name:        "Test Community",
		Description: "Test Description",
		CreatorID:   1,
		AvatarPath:  &ava,
		CoverPath:   &ava,
	}
	createdAt := time.Now()
	updatedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO communities (name, description, creator_id, avatar_path, cover_path)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`)).
		WithArgs(community.Name, community.Description, community.CreatorID,
			community.AvatarPath, community.CoverPath).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int32(42), createdAt, updatedAt))

	err := store.CreateCommunity(ctx, community)
	assert.NoError(t, err)
	assert.Equal(t, int32(42), community.ID)
	assert.Equal(t, createdAt, community.CreatedAt)
	assert.Equal(t, updatedAt, community.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateCommunity_Error(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	community := &domain.Community{
		Name:        "Test Community",
		Description: "Test Description",
		CreatorID:   1,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO communities`)).
		WithArgs(community.Name, community.Description, community.CreatorID,
			community.AvatarPath, community.CoverPath).
		WillReturnError(errors.New("database error"))

	err := store.CreateCommunity(ctx, community)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create community")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateCommunity_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	community := &domain.Community{
		ID:          1,
		Name:        "Updated Name",
		Description: "Updated Description",
		CreatorID:   1,
		AvatarPath:  &ava,
		CoverPath:   &ava,
	}
	updatedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE communities 
		SET name = $1, description = $2, avatar_path = $3, cover_path = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND creator_id = $6
		RETURNING updated_at
	`)).
		WithArgs(community.Name, community.Description, community.AvatarPath,
			community.CoverPath, community.ID, community.CreatorID).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(updatedAt))

	err := store.UpdateCommunity(ctx, community)
	assert.NoError(t, err)
	assert.Equal(t, updatedAt, community.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateCommunity_NotFound(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	community := &domain.Community{
		ID:        1,
		CreatorID: 1,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE communities`)).
		WithArgs(community.Name, community.Description, community.AvatarPath,
			community.CoverPath, community.ID, community.CreatorID).
		WillReturnError(sql.ErrNoRows)

	err := store.UpdateCommunity(ctx, community)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateCommunity_Error(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	community := &domain.Community{
		ID:        1,
		CreatorID: 1,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE communities`)).
		WithArgs(community.Name, community.Description, community.AvatarPath,
			community.CoverPath, community.ID, community.CreatorID).
		WillReturnError(errors.New("database error"))

	err := store.UpdateCommunity(ctx, community)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update community")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteCommunity_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	creatorID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM communities WHERE id = $1 AND creator_id = $2`)).
		WithArgs(communityID, creatorID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := store.DeleteCommunity(ctx, communityID, creatorID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteCommunity_NotFound(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	creatorID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM communities WHERE id = $1 AND creator_id = $2`)).
		WithArgs(communityID, creatorID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.DeleteCommunity(ctx, communityID, creatorID)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteCommunity_Error(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	creatorID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM communities WHERE id = $1 AND creator_id = $2`)).
		WithArgs(communityID, creatorID).
		WillReturnError(errors.New("database error"))

	err := store.DeleteCommunity(ctx, communityID, creatorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete community")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommunityByID_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	createdAt := time.Now()
	updatedAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(communityID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "creator_id", "avatar_path", "cover_path",
			"created_at", "updated_at", "subscribers_count",
		}).AddRow(
			int32(1), "Test Community", "Test Description", int32(1),
			"/avatar.jpg", "/cover.jpg", createdAt, updatedAt, int32(100),
		))

	community, err := store.GetCommunityByID(ctx, communityID)
	assert.NoError(t, err)
	assert.NotNil(t, community)
	assert.Equal(t, int32(1), community.ID)
	assert.Equal(t, "Test Community", community.Name)
	assert.Equal(t, int32(100), community.SubscribersCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommunityByID_NotFound(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(communityID).
		WillReturnError(sql.ErrNoRows)

	community, err := store.GetCommunityByID(ctx, communityID)
	assert.Error(t, err)
	assert.Nil(t, community)
	assert.Equal(t, domain.ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommunityByID_Error(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(communityID).
		WillReturnError(errors.New("database error"))

	community, err := store.GetCommunityByID(ctx, communityID)
	assert.Error(t, err)
	assert.Nil(t, community)
	assert.Contains(t, err.Error(), "failed to get community")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserCommunities_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "avatar_path", "subscribers_count",
		}).
			AddRow(int32(1), "Community 1", "Desc 1", "/avatar1.jpg", int32(10)).
			AddRow(int32(2), "Community 2", "Desc 2", "/avatar2.jpg", int32(20)))

	communities, err := store.GetUserCommunities(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, communities, 2)
	assert.Equal(t, int32(1), communities[0].ID)
	assert.Equal(t, "Community 1", communities[0].Name)
	assert.Equal(t, int32(10), communities[0].SubscribersCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserCommunities_EmptyResult(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "avatar_path", "subscribers_count",
		}))

	communities, err := store.GetUserCommunities(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Empty(t, communities)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserCommunities_Error(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID, limit, offset).
		WillReturnError(errors.New("database error"))

	communities, err := store.GetUserCommunities(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, communities)
	assert.Contains(t, err.Error(), "failed to query user communities")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOtherCommunities_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "avatar_path", "subscribers_count",
		}).
			AddRow(int32(3), "Other Community 1", "Desc 1", "/avatar3.jpg", int32(30)).
			AddRow(int32(4), "Other Community 2", "Desc 2", "/avatar4.jpg", int32(40)))

	communities, err := store.GetOtherCommunities(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, communities, 2)
	assert.Equal(t, int32(3), communities[0].ID)
	assert.Equal(t, "Other Community 1", communities[0].Name)
	assert.Equal(t, int32(30), communities[0].SubscribersCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscribe_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	userID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO community_subscriptions (community_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (community_id, user_id) DO NOTHING
	`)).
		WithArgs(communityID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := store.Subscribe(ctx, communityID, userID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscribe_Error(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	userID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO community_subscriptions`)).
		WithArgs(communityID, userID).
		WillReturnError(errors.New("database error"))

	err := store.Subscribe(ctx, communityID, userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to subscribe")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUnsubscribe_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	userID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM community_subscriptions WHERE community_id = $1 AND user_id = $2`)).
		WithArgs(communityID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := store.Unsubscribe(ctx, communityID, userID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUnsubscribe_NotFound(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	userID := int32(1)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM community_subscriptions WHERE community_id = $1 AND user_id = $2`)).
		WithArgs(communityID, userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.Unsubscribe(ctx, communityID, userID)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsSubscribed_True(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM community_subscriptions WHERE community_id = $1 AND user_id = $2)`)).
		WithArgs(communityID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	isSubscribed, err := store.IsSubscribed(ctx, communityID, userID)
	assert.NoError(t, err)
	assert.True(t, isSubscribed)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsSubscribed_False(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM community_subscriptions WHERE community_id = $1 AND user_id = $2)`)).
		WithArgs(communityID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	isSubscribed, err := store.IsSubscribed(ctx, communityID, userID)
	assert.NoError(t, err)
	assert.False(t, isSubscribed)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsSubscribed_Error(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM community_subscriptions WHERE community_id = $1 AND user_id = $2)`)).
		WithArgs(communityID, userID).
		WillReturnError(errors.New("database error"))

	isSubscribed, err := store.IsSubscribed(ctx, communityID, userID)
	assert.Error(t, err)
	assert.False(t, isSubscribed)
	assert.Contains(t, err.Error(), "failed to check subscription")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommunitiesByIDs_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityIDs := []int32{1, 2, 3}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT 
			c.id,
			c.name,
			c.description,
			c.avatar_path,
			(SELECT COUNT(*) FROM community_subscriptions WHERE community_id = c.id) as subscribers_count
		FROM communities c
		WHERE c.id = ANY($1)
		ORDER BY c.created_at DESC
	`)).
		WithArgs(pq.Array(communityIDs)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "avatar_path", "subscribers_count",
		}).
			AddRow(int32(1), "Community 1", "Desc 1", "/avatar1.jpg", int32(10)).
			AddRow(int32(2), "Community 2", "Desc 2", "/avatar2.jpg", int32(20)).
			AddRow(int32(3), "Community 3", "Desc 3", "/avatar3.jpg", int32(30)))

	communities, err := store.GetCommunitiesByIDs(ctx, communityIDs)
	assert.NoError(t, err)
	assert.Len(t, communities, 3)
	assert.Equal(t, int32(1), communities[0].ID)
	assert.Equal(t, int32(30), communities[2].SubscribersCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommunitiesByIDs_EmptyInput(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	var communityIDs []int32

	// Should not make database call for empty slice
	communities, err := store.GetCommunitiesByIDs(ctx, communityIDs)
	assert.NoError(t, err)
	assert.Empty(t, communities)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommunitiesByIDs_Error(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityIDs := []int32{1, 2}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(pq.Array(communityIDs)).
		WillReturnError(errors.New("database error"))

	communities, err := store.GetCommunitiesByIDs(ctx, communityIDs)
	assert.Error(t, err)
	assert.Nil(t, communities)
	assert.Contains(t, err.Error(), "failed to query communities by IDs")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCreatedCommunities_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT 
			c.id, 
			c.name, 
			c.avatar_path
		FROM communities c
		WHERE c.creator_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3
	`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "avatar_path"}).
			AddRow(int32(1), "My Community 1", ava).
			AddRow(int32(2), "My Community 2", "/avatar2.jpg"))

	communities, err := store.GetCreatedCommunities(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, communities, 2)
	assert.Equal(t, int32(1), communities[0].ID)
	assert.Equal(t, "My Community 1", communities[0].Name)
	assert.Equal(t, ava, *communities[0].AvatarPath)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCommunitySubscribers_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT user_id
		FROM community_subscriptions cs
		WHERE cs.community_id = $1
		ORDER BY cs.created_at DESC
		LIMIT $2 OFFSET $3
	`)).
		WithArgs(communityID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).
			AddRow(int32(10)).
			AddRow(int32(20)).
			AddRow(int32(30)))

	subscribers, err := store.GetCommunitySubscribers(ctx, communityID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, subscribers, 3)
	assert.Equal(t, int32(10), subscribers[0])
	assert.Equal(t, int32(30), subscribers[2])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserSubscribedCommunityIDs_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT community_id
		FROM community_subscriptions 
		WHERE user_id = $1
		ORDER BY created_at DESC
	`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"community_id"}).
			AddRow(int32(10)).
			AddRow(int32(20)).
			AddRow(int32(30)))

	communityIDs, err := store.GetUserSubscribedCommunityIDs(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, communityIDs, 3)
	assert.Equal(t, int32(10), communityIDs[0])
	assert.Equal(t, int32(30), communityIDs[2])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserCommunities_ScanError(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	rows := sqlmock.NewRows([]string{
		"id", "name", "description", "avatar_path", "subscribers_count",
	}).AddRow(int32(1), "Community 1", "Desc 1", "/avatar1.jpg", int32(10)).
		// RowError устанавливает ошибку для следующего Next(), но нам нужно сэмулировать ошибку сканирования
		// Для этого можно вернуть несовпадающие типы данных
		AddRow("invalid", "Community 2", "Desc 2", "/avatar2.jpg", "invalid") // Неправильные типы для id и subscribers_count

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(rows)

	communities, err := store.GetUserCommunities(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, communities)
	assert.Contains(t, err.Error(), "failed to scan community")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test for rows.Err() error in GetCommunitiesByIDs
func TestGetCommunitiesByIDs_RowsErr(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	communityIDs := []int32{1, 2}

	rows := sqlmock.NewRows([]string{
		"id", "name", "description", "avatar_path", "subscribers_count",
	}).AddRow(int32(1), "Community 1", "Desc 1", "/avatar1.jpg", int32(10)).
		AddRow(int32(2), "Community 2", "Desc 2", "/avatar2.jpg", int32(20)).
		RowError(1, errors.New("row iteration error"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(pq.Array(communityIDs)).
		WillReturnRows(rows)

	communities, err := store.GetCommunitiesByIDs(ctx, communityIDs)
	assert.Error(t, err)
	assert.Nil(t, communities)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test for GetUserCommunitiesByID (should work the same as GetUserCommunities)
func TestGetUserCommunitiesByID_Success(t *testing.T) {
	store, mock, dbConn := newCommunityStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	targetUserID := int32(2)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(targetUserID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "avatar_path", "subscribers_count",
		}).
			AddRow(int32(1), "Community 1", "Desc 1", "/avatar1.jpg", int32(10)))

	communities, err := store.GetUserCommunitiesByID(ctx, targetUserID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, communities, 1)
	assert.Equal(t, targetUserID, int32(2)) // Verify we're using the correct user ID
	assert.NoError(t, mock.ExpectationsWereMet())
}
