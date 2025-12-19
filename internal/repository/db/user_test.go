package db

import (
	"context"
	"database/sql"
	"errors"
	"project/domain"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateUser_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	user := domain.User{Email: "test@example.com", Password: "pass"}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id")).
		WithArgs(user.Email, user.Password).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	id, err := store.CreateUser(ctx, user)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUser_BeginTxError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	user := domain.User{Email: "test@example.com", Password: "pass"}

	mock.ExpectBegin().WillReturnError(errors.New("tx begin error"))

	id, err := store.CreateUser(ctx, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin tx")
	assert.Equal(t, int32(0), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUser_InsertUserFail(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	user := domain.User{Email: "test@example.com", Password: "pass"}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id")).
		WithArgs(user.Email, user.Password).
		WillReturnError(errors.New("insert user error"))
	mock.ExpectRollback()

	id, err := store.CreateUser(ctx, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert user")
	assert.Equal(t, int32(0), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUser_CommitError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	user := domain.User{Email: "test@example.com", Password: "pass"}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id")).
		WithArgs(user.Email, user.Password).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	id, err := store.CreateUser(ctx, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commit tx")
	assert.Equal(t, int32(0), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByEmail_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	email := "test@example.com"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password FROM users WHERE email = $1")).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password"}).AddRow(int32(1), email, "pass"))

	user, err := store.GetUserByEmail(ctx, email)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int32(1), user.ID) // Исправлено: сравниваем int32 с int32
	assert.Equal(t, email, user.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByEmail_DBError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	email := "test@example.com"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password FROM users WHERE email = $1")).
		WithArgs(email).
		WillReturnError(errors.New("db error"))

	user, err := store.GetUserByEmail(ctx, email)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	email := "missing@example.com"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password FROM users WHERE email = $1")).
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	user, err := store.GetUserByEmail(ctx, email)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByID_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password, role FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "role"}).AddRow(userID, "test@example.com", "pass", "user"))

	user, err := store.GetUserByID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "user", user.Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByID_DBError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password, role FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnError(errors.New("db error"))

	user, err := store.GetUserByID(ctx, userID)
	assert.Error(t, err)
	assert.Equal(t, domain.User{}, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByID_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password, role FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	user, err := store.GetUserByID(ctx, userID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Equal(t, domain.User{}, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserExists_True(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := store.IsUserExists(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserExists_False(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := store.IsUserExists(ctx, userID)
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserExists_DBError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)")).
		WithArgs(userID).
		WillReturnError(errors.New("db error"))

	exists, err := store.IsUserExists(ctx, userID)
	assert.Error(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserAdmin_WithAdminUser(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	tempSession := &domain.TempSessionInfo{UserID: &userID}
	ctx = context.WithValue(ctx, domain.TempSessionCtxKey, tempSession)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password, role FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "role"}).AddRow(userID, "admin@example.com", "pass", "admin"))

	isAdmin, err := store.IsUserAdmin(ctx)
	assert.NoError(t, err)
	assert.True(t, isAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserAdmin_WithRegularUser(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	tempSession := &domain.TempSessionInfo{UserID: &userID}
	ctx = context.WithValue(ctx, domain.TempSessionCtxKey, tempSession)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password, role FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "role"}).AddRow(userID, "user@example.com", "pass", "user"))

	isAdmin, err := store.IsUserAdmin(ctx)
	assert.NoError(t, err)
	assert.False(t, isAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserAdmin_UserNotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	tempSession := &domain.TempSessionInfo{UserID: &userID}
	ctx = context.WithValue(ctx, domain.TempSessionCtxKey, tempSession)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password, role FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	isAdmin, err := store.IsUserAdmin(ctx)
	assert.Error(t, err)
	assert.False(t, isAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserAdmin_NoTempSession(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	// Не добавляем TempSessionInfo в контекст

	isAdmin, err := store.IsUserAdmin(ctx)
	assert.NoError(t, err)
	assert.False(t, isAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserAdmin_NilUserID(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	tempSession := &domain.TempSessionInfo{UserID: nil} // nil UserID
	ctx = context.WithValue(ctx, domain.TempSessionCtxKey, tempSession)

	isAdmin, err := store.IsUserAdmin(ctx)
	assert.NoError(t, err)
	assert.False(t, isAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserAdmin_EmptyTempSession(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	tempSession := &domain.TempSessionInfo{} // Пустая структура
	ctx = context.WithValue(ctx, domain.TempSessionCtxKey, tempSession)

	isAdmin, err := store.IsUserAdmin(ctx)
	assert.NoError(t, err)
	assert.False(t, isAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserAdmin_DBError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := int32(1)
	tempSession := &domain.TempSessionInfo{UserID: &userID}
	ctx = context.WithValue(ctx, domain.TempSessionCtxKey, tempSession)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password, role FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnError(errors.New("db error"))

	isAdmin, err := store.IsUserAdmin(ctx)
	assert.Error(t, err)
	assert.False(t, isAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}