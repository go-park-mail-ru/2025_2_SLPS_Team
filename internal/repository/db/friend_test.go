package db

import (
	"context"
	"database/sql"
	"project/domain"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *DBFriendStore) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	store := NewDBFriendStore(mockDB).(*DBFriendStore)
	return mockDB, mock, store
}

// Custom result that returns error for RowsAffected
type errorResult struct{}

func (e errorResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (e errorResult) RowsAffected() (int64, error) {
	return 0, assert.AnError
}

func TestCreateFriendship(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO friend_relationships`).
			WithArgs(1, 2, 1, domain.FriendshipPending).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.CreateFriendship(context.Background(), 1, 2)
		assert.NoError(t, err)
	})

	t.Run("User order correction", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO friend_relationships`).
			WithArgs(1, 2, 2, domain.FriendshipPending).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.CreateFriendship(context.Background(), 2, 1)
		assert.NoError(t, err)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO friend_relationships`).
			WithArgs(1, 2, 1, domain.FriendshipPending).
			WillReturnError(assert.AnError)
		err := store.CreateFriendship(context.Background(), 1, 2)
		assert.Error(t, err)
	})

	t.Run("On conflict update", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO friend_relationships`).
			WithArgs(1, 2, 1, domain.FriendshipPending).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.CreateFriendship(context.Background(), 1, 2)
		assert.NoError(t, err)
	})
}

func TestGetFriendship(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at"}).
			AddRow(1, 2, 1, domain.FriendshipPending, now, now)
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnRows(rows)
		f, err := store.GetFriendship(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, 1, f.FirstUserID)
		assert.Equal(t, 2, f.SecondUserID)
		assert.Equal(t, domain.FriendshipPending, f.Status)
	})

	t.Run("User order correction", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at"}).
			AddRow(1, 2, 2, domain.FriendshipPending, now, now)
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnRows(rows)
		f, err := store.GetFriendship(context.Background(), 2, 1)
		assert.NoError(t, err)
		assert.Equal(t, 1, f.FirstUserID)
		assert.Equal(t, 2, f.SecondUserID)
	})

	t.Run("Not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnError(sql.ErrNoRows)
		f, err := store.GetFriendship(context.Background(), 1, 2)
		assert.ErrorIs(t, err, domain.ErrFriendshipNotFound)
		assert.Nil(t, f)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnError(assert.AnError)
		f, err := store.GetFriendship(context.Background(), 1, 2)
		assert.Error(t, err)
		assert.Nil(t, f)
	})
}

func TestUpdateFriendshipStatus(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE friend_relationships`).
			WithArgs(1, domain.FriendshipAccepted, 1, 2).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.UpdateFriendshipStatus(context.Background(), 1, 2, domain.FriendshipAccepted)
		assert.NoError(t, err)
	})

	t.Run("User order correction", func(t *testing.T) {
		mock.ExpectExec(`UPDATE friend_relationships`).
			WithArgs(2, domain.FriendshipRejected, 1, 2).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.UpdateFriendshipStatus(context.Background(), 2, 1, domain.FriendshipRejected)
		assert.NoError(t, err)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectExec(`UPDATE friend_relationships`).
			WithArgs(1, domain.FriendshipAccepted, 1, 2).
			WillReturnError(assert.AnError)
		err := store.UpdateFriendshipStatus(context.Background(), 1, 2, domain.FriendshipAccepted)
		assert.Error(t, err)
	})

	t.Run("No rows affected", func(t *testing.T) {
		mock.ExpectExec(`UPDATE friend_relationships`).
			WithArgs(1, domain.FriendshipAccepted, 1, 2).
			WillReturnResult(sqlmock.NewResult(0, 0))
		err := store.UpdateFriendshipStatus(context.Background(), 1, 2, domain.FriendshipAccepted)
		assert.ErrorIs(t, err, domain.ErrFriendshipNotFound)
	})

	t.Run("Rows affected error", func(t *testing.T) {
		mock.ExpectExec(`UPDATE friend_relationships`).
			WithArgs(1, domain.FriendshipAccepted, 1, 2).
			WillReturnResult(&errorResult{})
		err := store.UpdateFriendshipStatus(context.Background(), 1, 2, domain.FriendshipAccepted)
		assert.Error(t, err)
	})
}

func TestDeleteFriendship(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM friend_relationships`).
			WithArgs(1, 2).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.DeleteFriendship(context.Background(), 1, 2)
		assert.NoError(t, err)
	})

	t.Run("User order correction", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM friend_relationships`).
			WithArgs(1, 2).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.DeleteFriendship(context.Background(), 2, 1)
		assert.NoError(t, err)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM friend_relationships`).
			WithArgs(1, 2).
			WillReturnError(assert.AnError)
		err := store.DeleteFriendship(context.Background(), 1, 2)
		assert.Error(t, err)
	})

	t.Run("No rows affected", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM friend_relationships`).
			WithArgs(1, 2).
			WillReturnResult(sqlmock.NewResult(0, 0))
		err := store.DeleteFriendship(context.Background(), 1, 2)
		assert.ErrorIs(t, err, domain.ErrFriendshipNotFound)
	})

	t.Run("Rows affected error", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM friend_relationships`).
			WithArgs(1, 2).
			WillReturnResult(&errorResult{})
		err := store.DeleteFriendship(context.Background(), 1, 2)
		assert.Error(t, err)
	})
}

func TestAreFriends(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("True", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(1, 2).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		ok, err := store.AreFriends(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("False", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(1, 2).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
		ok, err := store.AreFriends(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("User order correction", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(1, 2).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		ok, err := store.AreFriends(context.Background(), 2, 1)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(1, 2).
			WillReturnError(assert.AnError)
		ok, err := store.AreFriends(context.Background(), 1, 2)
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestGetFriendshipStatus(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success with status", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at"}).
			AddRow(1, 2, 1, domain.FriendshipAccepted, now, now)
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnRows(rows)
		status, err := store.GetFriendshipStatus(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipAccepted, status)
	})

	t.Run("Not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnError(sql.ErrNoRows)
		status, err := store.GetFriendshipStatus(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipStatus(""), status)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnError(assert.AnError)
		status, err := store.GetFriendshipStatus(context.Background(), 1, 2)
		assert.Error(t, err)
		assert.Equal(t, domain.FriendshipStatus(""), status)
	})
}

func TestGetUserFriends(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"friend_id"}).
			AddRow(2).
			AddRow(3)
		mock.ExpectQuery(`SELECT CASE`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		friendIDs, err := store.GetUserFriends(context.Background(), 1, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, friendIDs, 2)
		assert.Equal(t, []int32{2, 3}, friendIDs)
	})

	t.Run("Empty result", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"})
		mock.ExpectQuery(`SELECT p.user_id, p.first_name \|\| ' '\|\|p.last_name, p.avatar_path`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		friends, err := store.GetUserFriends(context.Background(), 1, 10, 0)
		assert.NoError(t, err)
		assert.Empty(t, friends)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT p.user_id, p.first_name \|\| ' '\|\|p.last_name, p.avatar_path`).
			WithArgs(1, 10, 0).
			WillReturnError(assert.AnError)
		friends, err := store.GetUserFriends(context.Background(), 1, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, friends)
	})

	t.Run("Scan error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"}).
			AddRow("invalid", "John Doe", "/avatar.jpg")
		mock.ExpectQuery(`SELECT p.user_id, p.first_name \|\| ' '\|\|p.last_name, p.avatar_path`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		friends, err := store.GetUserFriends(context.Background(), 1, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, friends)
	})

	t.Run("Rows error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"}).
			AddRow(2, "John Doe", "/avatar.jpg").
			RowError(0, assert.AnError) // Error on first row
		mock.ExpectQuery(`SELECT p.user_id, p.first_name \|\| ' '\|\|p.last_name, p.avatar_path`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		friends, err := store.GetUserFriends(context.Background(), 1, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, friends)
	})
}

func TestGetAllUsers(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"friend_id"}).
			AddRow(2).
			AddRow(3)
		mock.ExpectQuery(`SELECT CASE`).
			WithArgs(1).
			WillReturnRows(rows)
		userIDs, err := store.GetAllUsers(context.Background(), 1)
		assert.NoError(t, err)
		assert.Len(t, userIDs, 2)
		assert.Equal(t, []int32{2, 3}, userIDs)
	})

	t.Run("Empty result", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"friend_id"})
		mock.ExpectQuery(`SELECT CASE`).
			WithArgs(1).
			WillReturnRows(rows)
		userIDs, err := store.GetAllUsers(context.Background(), 1)
		assert.NoError(t, err)
		assert.Empty(t, userIDs)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT CASE`).
			WithArgs(1).
			WillReturnError(assert.AnError)
		userIDs, err := store.GetAllUsers(context.Background(), 1)
		assert.Error(t, err)
		assert.Nil(t, userIDs)
	})
}
func TestGetFriendshipRequests(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"}).
			AddRow(2, "Requester One", "/avatar1.jpg").
			AddRow(3, "Requester Two", "/avatar2.jpg")
		mock.ExpectQuery(`SELECT p.user_id`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		requests, err := store.GetFriendshipRequests(context.Background(), 1, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, requests, 2)
	})

	t.Run("Empty result", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"})
		mock.ExpectQuery(`SELECT p.user_id`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		requests, err := store.GetFriendshipRequests(context.Background(), 1, 10, 0)
		assert.NoError(t, err)
		assert.Empty(t, requests)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT p.user_id`).
			WithArgs(1, 10, 0).
			WillReturnError(assert.AnError)
		requests, err := store.GetFriendshipRequests(context.Background(), 1, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, requests)
	})
}

func TestGetSentRequests(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"}).
			AddRow(2, "Receiver One", "/avatar1.jpg").
			AddRow(3, "Receiver Two", "/avatar2.jpg")
		mock.ExpectQuery(`SELECT p.user_id`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		requests, err := store.GetSentRequests(context.Background(), 1, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, requests, 2)
	})

	t.Run("Empty result", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"})
		mock.ExpectQuery(`SELECT p.user_id`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		requests, err := store.GetSentRequests(context.Background(), 1, 10, 0)
		assert.NoError(t, err)
		assert.Empty(t, requests)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT p.user_id`).
			WithArgs(1, 10, 0).
			WillReturnError(assert.AnError)
		requests, err := store.GetSentRequests(context.Background(), 1, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, requests)
	})
}

func TestCountUserRelations(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"accepted_count", "pending_count", "sent_count", "blocked_count"}).
			AddRow(5, 3, 2, 1)
		mock.ExpectQuery(`WITH counts AS`).
			WithArgs(1).
			WillReturnRows(rows)
		counts, err := store.CountUserRelations(context.Background(), 1)
		assert.NoError(t, err)
		assert.NotNil(t, counts)
		assert.Equal(t, int32(5), counts.Accepted)
		assert.Equal(t, int32(3), counts.Pending)
		assert.Equal(t, int32(2), counts.Sent)
		assert.Equal(t, int32(1), counts.Blocked)
	})

	t.Run("Empty counts", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"accepted_count", "pending_count", "sent_count", "blocked_count"}).
			AddRow(0, 0, 0, 0)
		mock.ExpectQuery(`WITH counts AS`).
			WithArgs(1).
			WillReturnRows(rows)
		counts, err := store.CountUserRelations(context.Background(), 1)
		assert.NoError(t, err)
		assert.NotNil(t, counts)
		assert.Equal(t, int32(0), counts.Accepted)
		assert.Equal(t, int32(0), counts.Pending)
		assert.Equal(t, int32(0), counts.Sent)
		assert.Equal(t, int32(0), counts.Blocked)
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectQuery(`WITH counts AS`).
			WithArgs(1).
			WillReturnError(assert.AnError)
		counts, err := store.CountUserRelations(context.Background(), 1)
		assert.Error(t, err)
		assert.Nil(t, counts)
	})

	t.Run("Scan error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"accepted_count", "pending_count", "sent_count", "blocked_count"}).
			AddRow("invalid", 3, 2, 1)
		mock.ExpectQuery(`WITH counts AS`).
			WithArgs(1).
			WillReturnRows(rows)
		counts, err := store.CountUserRelations(context.Background(), 1)
		assert.Error(t, err)
		assert.Nil(t, counts)
	})
}
func TestEnsureUserOrder(t *testing.T) {
	t.Run("User1 less than User2", func(t *testing.T) {
		u1, u2 := ensureUserOrder(1, 2)
		assert.Equal(t, 1, u1)
		assert.Equal(t, 2, u2)
	})

	t.Run("User1 greater than User2", func(t *testing.T) {
		u1, u2 := ensureUserOrder(2, 1)
		assert.Equal(t, 1, u1)
		assert.Equal(t, 2, u2)
	})

	t.Run("Equal users", func(t *testing.T) {
		u1, u2 := ensureUserOrder(1, 1)
		assert.Equal(t, 1, u1)
		assert.Equal(t, 1, u2)
	})

	t.Run("Negative user IDs", func(t *testing.T) {
		u1, u2 := ensureUserOrder(-2, -1)
		assert.Equal(t, -2, u1)
		assert.Equal(t, -1, u2)
	})
}

func TestEdgeCases(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Zero limit and offset", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"})
		mock.ExpectQuery(`SELECT p.user_id, p.first_name \|\| ' '\|\|p.last_name, p.avatar_path`).
			WithArgs(1, 0, 0).
			WillReturnRows(rows)
		friends, err := store.GetUserFriends(context.Background(), 1, 0, 0)
		assert.NoError(t, err)
		assert.Empty(t, friends)
	})

	t.Run("Large limit and offset", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"})
		mock.ExpectQuery(`SELECT p.user_id, p.first_name \|\| ' '\|\|p.last_name, p.avatar_path`).
			WithArgs(1, 1000, 500).
			WillReturnRows(rows)
		friends, err := store.GetUserFriends(context.Background(), 1, 1000, 500)
		assert.NoError(t, err)
		assert.Empty(t, friends)
	})

	t.Run("Different friendship statuses", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at"}).
			AddRow(1, 2, 1, domain.FriendshipRejected, now, now)
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnRows(rows)
		f, err := store.GetFriendship(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipRejected, f.Status)
	})

	t.Run("Friendship with blocked status", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at"}).
			AddRow(1, 2, 1, domain.FriendshipBlocked, now, now)
		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnRows(rows)
		f, err := store.GetFriendship(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipBlocked, f.Status)
	})
}

func TestContextCancellation(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
			WithArgs(1, 2).
			WillReturnError(context.Canceled)

		f, err := store.GetFriendship(ctx, 1, 2)
		assert.Error(t, err)
		assert.Nil(t, f)
	})
}

func TestNullAvatarPath(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Null avatar path", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"friend_id"}).
			AddRow(2)
		mock.ExpectQuery(`SELECT CASE`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		friendIDs, err := store.GetUserFriends(context.Background(), 1, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, friendIDs, 1)
		assert.Equal(t, int32(2), friendIDs[0])
	})
}
func TestEmptyUserName(t *testing.T) {
	_, mock, store := setupMockDB(t)

	t.Run("Empty user name", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"friend_id"}).
			AddRow(2)
		mock.ExpectQuery(`SELECT CASE`).
			WithArgs(1, 10, 0).
			WillReturnRows(rows)
		friendIDs, err := store.GetUserFriends(context.Background(), 1, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, friendIDs, 1)
		assert.Equal(t, int32(2), friendIDs[0])
	})
}
