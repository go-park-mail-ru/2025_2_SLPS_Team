package db

import (
	"context"
	"database/sql"
	"errors"
	"project/domain"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateUser_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	dob := time.Now()
	user := domain.User{Email: "test@example.com", Password: "pass"}
	profile := domain.Profile{FirstName: "John", LastName: "Doe", Gender: "M", Dob: dob}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id")).
		WithArgs(user.Email, user.Password).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profiles (user_id, first_name, last_name, gender, dob) VALUES ($1, $2, $3, $4, $5)")).
		WithArgs(1, profile.FirstName, profile.LastName, profile.Gender, profile.Dob).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	id, err := store.CreateUser(ctx, user, profile)
	assert.NoError(t, err)
	assert.Equal(t, 1, id)
}

func TestCreateUser_InsertUserFail(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	user := domain.User{Email: "test@example.com", Password: "pass"}
	profile := domain.Profile{}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id")).
		WithArgs(user.Email, user.Password).
		WillReturnError(errors.New("insert user error"))
	mock.ExpectRollback()

	id, err := store.CreateUser(ctx, user, profile)
	assert.Error(t, err)
	assert.Equal(t, 0, id)
}

func TestGetUserByEmail_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	email := "test@example.com"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password FROM users WHERE email = $1")).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password"}).AddRow(1, email, "pass"))

	user, err := store.GetUserByEmail(ctx, email)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, email, user.Email)
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
}

func TestGetUserByID_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := 1
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password"}).AddRow(userID, "test@example.com", "pass"))

	user, err := store.GetUserByID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, userID, user.ID)
}

func TestGetUserByID_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := 1
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, password FROM users WHERE id = $1")).
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	user, err := store.GetUserByID(ctx, userID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Equal(t, domain.User{}, user)
}

func TestIsUserExists_True(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := 1
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := store.IsUserExists(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestIsUserExists_False(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBUserStore(dbConn)
	ctx := context.Background()

	userID := 1
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := store.IsUserExists(ctx, userID)
	assert.NoError(t, err)
	assert.False(t, exists)
}
