package db

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"project/domain"
)

func newFriendStoreMock(t *testing.T) (*DBFriendStore, sqlmock.Sqlmock, *sql.DB) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err, "failed to create sqlmock")
	store := NewDBFriendStore(dbConn).(*DBFriendStore)
	return store, mock, dbConn
}

func testContext() context.Context {
	return context.WithValue(context.Background(), "logger", zap.NewNop())
}

func TestEnsureUserOrder(t *testing.T) {
	tests := []struct {
		name     string
		userID1  int32
		userID2  int32
		expected [2]int32
	}{
		{"Already ordered", 1, 2, [2]int32{1, 2}},
		{"Reverse order", 2, 1, [2]int32{1, 2}},
		{"Equal IDs", 5, 5, [2]int32{5, 5}},
		{"Large numbers", 100, 50, [2]int32{50, 100}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, second := ensureUserOrder(tt.userID1, tt.userID2)
			assert.Equal(t, tt.expected[0], first)
			assert.Equal(t, tt.expected[1], second)
		})
	}
}

func TestCreateFriendship(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	actionUserID := int32(1)
	targetUserID := int32(2)

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO friend_relationships (first_user_id, second_user_id, action_user_id, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (first_user_id, second_user_id) 
		DO UPDATE SET status = $4, action_user_id = $3, updated_at = CURRENT_TIMESTAMP
	`)).
		WithArgs(int32(1), int32(2), actionUserID, domain.FriendshipPending).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.CreateFriendship(ctx, actionUserID, targetUserID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFriendship_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	actionUserID := int32(1)
	targetUserID := int32(2)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO friend_relationships`)).
		WithArgs(int32(1), int32(2), actionUserID, domain.FriendshipPending).
		WillReturnError(errors.New("database error"))

	err := store.CreateFriendship(ctx, actionUserID, targetUserID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create friendship")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFriendship_ReversedOrder(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	actionUserID := int32(5)
	targetUserID := int32(3)

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO friend_relationships (first_user_id, second_user_id, action_user_id, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (first_user_id, second_user_id) 
		DO UPDATE SET status = $4, action_user_id = $3, updated_at = CURRENT_TIMESTAMP
	`)).
		WithArgs(int32(3), int32(5), actionUserID, domain.FriendshipPending).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.CreateFriendship(ctx, actionUserID, targetUserID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFriendship_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)
	now := time.Now()

	expectedFriendship := &domain.Friendship{
		FirstUserID:  1,
		SecondUserID: 2,
		ActionUserID: 1,
		Status:       domain.FriendshipPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT first_user_id, second_user_id, action_user_id, status, created_at, updated_at
		FROM friend_relationships 
		WHERE first_user_id = $1 AND second_user_id = $2
	`)).
		WithArgs(int32(1), int32(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at",
		}).AddRow(
			expectedFriendship.FirstUserID,
			expectedFriendship.SecondUserID,
			expectedFriendship.ActionUserID,
			expectedFriendship.Status,
			expectedFriendship.CreatedAt,
			expectedFriendship.UpdatedAt,
		))

	friendship, err := store.GetFriendship(ctx, userID1, userID2)
	assert.NoError(t, err)
	assert.Equal(t, expectedFriendship, friendship)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFriendship_NotFound(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT first_user_id, second_user_id, action_user_id, status, created_at, updated_at
		FROM friend_relationships 
		WHERE first_user_id = $1 AND second_user_id = $2
	`)).
		WithArgs(int32(1), int32(2)).
		WillReturnError(sql.ErrNoRows)

	friendship, err := store.GetFriendship(ctx, userID1, userID2)
	assert.Error(t, err)
	assert.Nil(t, friendship)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFriendship_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(int32(1), int32(2)).
		WillReturnError(errors.New("database error"))

	friendship, err := store.GetFriendship(ctx, userID1, userID2)
	assert.Error(t, err)
	assert.Nil(t, friendship)
	assert.Contains(t, err.Error(), "failed to get friendship")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFriendshipStatus_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	actionUserID := int32(2)
	targetUserID := int32(1)
	status := domain.FriendshipAccepted

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE friend_relationships
		SET action_user_id = $1, status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE first_user_id = $3 AND second_user_id = $4
	`)).
		WithArgs(actionUserID, status, int32(1), int32(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := store.UpdateFriendshipStatus(ctx, actionUserID, targetUserID, status)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFriendshipStatus_NotFound(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	actionUserID := int32(2)
	targetUserID := int32(1)
	status := domain.FriendshipAccepted

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE friend_relationships`)).
		WithArgs(actionUserID, status, int32(1), int32(2)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.UpdateFriendshipStatus(ctx, actionUserID, targetUserID, status)
	assert.Error(t, err)
}

func TestUpdateFriendshipStatus_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	actionUserID := int32(2)
	targetUserID := int32(1)
	status := domain.FriendshipAccepted

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE friend_relationships`)).
		WithArgs(actionUserID, status, int32(1), int32(2)).
		WillReturnError(errors.New("database error"))

	err := store.UpdateFriendshipStatus(ctx, actionUserID, targetUserID, status)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to update friendship status")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFriendshipStatus_RowsAffectedError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	actionUserID := int32(2)
	targetUserID := int32(1)
	status := domain.FriendshipAccepted

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE friend_relationships`)).
		WithArgs(actionUserID, status, int32(1), int32(2)).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	err := store.UpdateFriendshipStatus(ctx, actionUserID, targetUserID, status)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to get rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFriends_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT CASE 
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS friend_id
FROM friend_relationships fr
WHERE (fr.first_user_id = $1 OR fr.second_user_id = $1)
  AND fr.status = 'accepted'
ORDER BY friend_id
LIMIT $2 OFFSET $3;
	`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"friend_id"}).
			AddRow(int32(2)).
			AddRow(int32(3)).
			AddRow(int32(4)))

	friendIDs, err := store.GetUserFriends(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Equal(t, []int32{2, 3, 4}, friendIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFriends_Empty(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"friend_id"}))

	friendIDs, err := store.GetUserFriends(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Empty(t, friendIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFriends_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID, limit, offset).
		WillReturnError(errors.New("database error"))

	friendIDs, err := store.GetUserFriends(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, friendIDs)
	assert.Contains(t, err.Error(), "failed to query user friends")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllUsers_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT CASE 
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS friend_id
FROM friend_relationships fr
    WHERE (fr.first_user_id = $1 or fr.second_user_id = $1)
ORDER BY friend_id
	`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"friend_id"}).
			AddRow(int32(2)).
			AddRow(int32(3)).
			AddRow(int32(5)))

	userIDs, err := store.GetAllUsers(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, []int32{2, 3, 5}, userIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllUsers_Empty(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"friend_id"}))

	userIDs, err := store.GetAllUsers(ctx, userID)
	assert.NoError(t, err)
	assert.Empty(t, userIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllUsers_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID).
		WillReturnError(errors.New("database error"))

	userIDs, err := store.GetAllUsers(ctx, userID)
	assert.Error(t, err)
	assert.Nil(t, userIDs)
	assert.Contains(t, err.Error(), "failed to query all users")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFriendshipRequests_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT CASE 
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS friend_id
		FROM friend_relationships fr
		WHERE (fr.first_user_id = $1 OR fr.second_user_id = $1) and
((fr.status = 'pending' AND fr.action_user_id != $1) OR (fr.status = 'rejected' AND fr.action_user_id = $1))
		ORDER BY fr.created_at DESC
		LIMIT $2 OFFSET $3
	`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"friend_id"}).
			AddRow(int32(2)).
			AddRow(int32(3)))

	friendIDs, err := store.GetFriendshipRequests(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Equal(t, []int32{2, 3}, friendIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSentRequests_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT CASE 
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS friend_id
		FROM friend_relationships fr
		WHERE (fr.first_user_id = $1 OR fr.second_user_id = $1) and
		((fr.status = 'pending' AND fr.action_user_id = $1) OR (fr.status = 'rejected' AND fr.action_user_id != $1))
		ORDER BY fr.created_at DESC
		LIMIT $2 OFFSET $3
	`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"friend_id"}).
			AddRow(int32(4)).
			AddRow(int32(5)))

	friendIDs, err := store.GetSentRequests(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Equal(t, []int32{4, 5}, friendIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFriendship_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM friend_relationships 
		WHERE first_user_id = $1 AND second_user_id = $2
	`)).
		WithArgs(int32(1), int32(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := store.DeleteFriendship(ctx, userID1, userID2)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFriendship_NotFound(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM friend_relationships`)).
		WithArgs(int32(1), int32(2)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.DeleteFriendship(ctx, userID1, userID2)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFriendship_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM friend_relationships`)).
		WithArgs(int32(1), int32(2)).
		WillReturnError(errors.New("database error"))

	err := store.DeleteFriendship(ctx, userID1, userID2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete friendship")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAreFriends_True(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS(
			SELECT 1 FROM friend_relationships 
			WHERE first_user_id = $1 AND second_user_id = $2 AND status = 'accepted'
		)
	`)).
		WithArgs(int32(1), int32(2)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	areFriends, err := store.AreFriends(ctx, userID1, userID2)
	assert.NoError(t, err)
	assert.True(t, areFriends)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAreFriends_False(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS`)).
		WithArgs(int32(1), int32(2)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	areFriends, err := store.AreFriends(ctx, userID1, userID2)
	assert.NoError(t, err)
	assert.False(t, areFriends)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAreFriends_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS`)).
		WithArgs(int32(1), int32(2)).
		WillReturnError(errors.New("database error"))

	areFriends, err := store.AreFriends(ctx, userID1, userID2)
	assert.Error(t, err)
	assert.False(t, areFriends)
	assert.Contains(t, err.Error(), "failed to check friendship")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFriendshipStatus_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT first_user_id, second_user_id, action_user_id, status, created_at, updated_at
		FROM friend_relationships 
		WHERE first_user_id = $1 AND second_user_id = $2
	`)).
		WithArgs(int32(1), int32(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at",
		}).AddRow(int32(1), int32(2), int32(1), domain.FriendshipAccepted, now, now))

	status, err := store.GetFriendshipStatus(ctx, userID1, userID2)
	assert.NoError(t, err)
	assert.Equal(t, domain.FriendshipAccepted, status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFriendshipStatus_NotFound(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(int32(1), int32(2)).
		WillReturnError(sql.ErrNoRows)

	status, err := store.GetFriendshipStatus(ctx, userID1, userID2)
	assert.NoError(t, err)
	assert.Empty(t, status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountUserRelations_Success(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	expectedCounts := &domain.UserRelationsCounts{
		Accepted: 5,
		Pending:  3,
		Sent:     2,
		Blocked:  1,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
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
    `)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"accepted_count",
			"pending_count",
			"sent_count",
			"blocked_count",
		}).AddRow(
			expectedCounts.Accepted,
			expectedCounts.Pending,
			expectedCounts.Sent,
			expectedCounts.Blocked,
		))

	counts, err := store.CountUserRelations(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, expectedCounts, counts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountUserRelations_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`WITH counts AS`)).
		WithArgs(userID).
		WillReturnError(errors.New("database error"))

	counts, err := store.CountUserRelations(ctx, userID)
	assert.Error(t, err)
	assert.Nil(t, counts)
	assert.Contains(t, err.Error(), "failed to count user relations")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserIDsByFriendType(t *testing.T) {
	tests := []struct {
		name     string
		fType    domain.FriendshipCountType
		expected []int32
	}{
		{"Accepted", domain.CountAccepted, []int32{2, 3}},
		{"Pending", domain.CountPending, []int32{4}},
		{"Sent", domain.CountSent, []int32{5}},
		{"Blocked", domain.CountBlocked, []int32{6}},
		{"NotFriends", domain.CountNotFriends, []int32{2, 3, 4, 5, 6}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock, dbConn := newFriendStoreMock(t)
			defer dbConn.Close()

			ctx := testContext()
			userID := int32(1)

			rows := sqlmock.NewRows([]string{"related_user_id"})
			for _, id := range tt.expected {
				rows.AddRow(id)
			}

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT CASE
         WHEN fr.first_user_id = $1 THEN fr.second_user_id
         ELSE fr.first_user_id
       END AS related_user_id
FROM friend_relationships fr
WHERE (fr.first_user_id = $1 OR fr.second_user_id = $1)
  AND (`)).
				WithArgs(userID).
				WillReturnRows(rows)

			ids, err := store.GetUserIDsByFriendType(ctx, userID, tt.fType)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, ids)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserIDsByFriendType_UnknownType(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	ids, err := store.GetUserIDsByFriendType(ctx, userID, domain.FriendshipCountType("unknown"))
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "unknown statusType")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserIDsByFriendType_WithError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID).
		WillReturnError(errors.New("database error"))

	ids, err := store.GetUserIDsByFriendType(ctx, userID, domain.CountAccepted)
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFriendship_ScanError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID1 := int32(1)
	userID2 := int32(2)

	// Неправильное количество колонок
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(int32(1), int32(2)).
		WillReturnRows(sqlmock.NewRows([]string{"first_user_id"}).AddRow(int32(1)))

	friendship, err := store.GetFriendship(ctx, userID1, userID2)
	assert.Error(t, err)
	assert.Nil(t, friendship)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFriends_RowsError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	rows := sqlmock.NewRows([]string{"friend_id"}).
		AddRow(int32(2)).
		AddRow(int32(3)).
		RowError(1, errors.New("row error"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(rows)

	friendIDs, err := store.GetUserFriends(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, friendIDs)
	assert.Contains(t, err.Error(), "Error iterating friend rows")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllUsers_RowsError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	rows := sqlmock.NewRows([]string{"friend_id"}).
		AddRow(int32(2)).
		RowError(0, errors.New("row error"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID).
		WillReturnRows(rows)

	userIDs, err := store.GetAllUsers(ctx, userID)
	assert.Error(t, err)
	assert.Nil(t, userIDs)
	assert.Contains(t, err.Error(), "error iterating user rows")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCountUserRelations_ScanError(t *testing.T) {
	store, mock, dbConn := newFriendStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	userID := int32(1)

	// Неправильное количество колонок
	mock.ExpectQuery(regexp.QuoteMeta(`WITH counts AS`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"accepted_count"}).AddRow(5))

	counts, err := store.CountUserRelations(ctx, userID)
	assert.Error(t, err)
	assert.Nil(t, counts)
	assert.NoError(t, mock.ExpectationsWereMet())
}
