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

func TestCreateFriendship(t *testing.T) {
	_, mock, store := setupMockDB(t)
	mock.ExpectExec(`INSERT INTO friend_relationships`).
		WithArgs(1, 2, 1, domain.FriendshipPending).
		WillReturnResult(sqlmock.NewResult(1, 1))
	err := store.CreateFriendship(context.Background(), 1, 2)
	assert.NoError(t, err)
}

func TestGetFriendship(t *testing.T) {
	_, mock, store := setupMockDB(t)
	rows := sqlmock.NewRows([]string{"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at"}).
		AddRow(1, 2, 1, domain.FriendshipPending, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT first_user_id, second_user_id, action_user_id, status`).
		WithArgs(1, 2).
		WillReturnRows(rows)
	f, err := store.GetFriendship(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, 1, f.FirstUserID)
}

func TestUpdateFriendshipStatus(t *testing.T) {
	_, mock, store := setupMockDB(t)
	mock.ExpectExec(`UPDATE friend_relationships`).
		WithArgs(domain.FriendshipAccepted, 1, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	err := store.UpdateFriendshipStatus(context.Background(), 1, 2, domain.FriendshipAccepted)
	assert.NoError(t, err)
}

func TestDeleteFriendship(t *testing.T) {
	_, mock, store := setupMockDB(t)
	mock.ExpectExec(`DELETE FROM friend_relationships`).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	err := store.DeleteFriendship(context.Background(), 1, 2)
	assert.NoError(t, err)
}

func TestAreFriends(t *testing.T) {
	_, mock, store := setupMockDB(t)
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(1, 2).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	ok, err := store.AreFriends(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestGetFriendshipStatus(t *testing.T) {
	_, mock, store := setupMockDB(t)
	rows := sqlmock.NewRows([]string{"first_user_id", "second_user_id", "action_user_id", "status", "created_at", "updated_at"}).
		AddRow(1, 2, 1, domain.FriendshipPending, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT first_user_id, second_user_id`).
		WithArgs(1, 2).
		WillReturnRows(rows)
	status, err := store.GetFriendshipStatus(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, domain.FriendshipPending, status)
}

func TestCountUserRelations(t *testing.T) {
	_, mock, store := setupMockDB(t)
	mock.ExpectQuery(`SELECT COUNT\(\*\)`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	count, err := store.CountUserRelations(context.Background(), 1, domain.CountAll)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}
