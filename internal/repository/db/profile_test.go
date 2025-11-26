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
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestUpdateProfile_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()
	var about string
	about = "About"
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE profiles SET first_name = $2, last_name = $3, gender = $4, dob = $5, about_myself = $6
WHERE user_id = $1`)).
		WithArgs(1, "John", "Doe", "M", time.Now(), &about).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.UpdateProfile(ctx, domain.Profile{
		FirstName:   "John",
		LastName:    "Doe",
		Gender:      "M",
		Dob:         time.Now(),
		AboutMyself: &about,
	}, 1)
	assert.NoError(t, err)
}

func TestUpdateProfile_ExecError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE profiles SET first_name = $2, last_name = $3, gender = $4, dob = $5, about_myself = $6
WHERE user_id = $1`)).
		WillReturnError(errors.New("exec failed"))

	err := store.UpdateProfile(ctx, domain.Profile{}, 1)
	assert.Error(t, err)
}

func TestUpdateAvatar_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE profiles SET  avatar_path = $2 WHERE user_id = $1`)).
		WithArgs(1, "path/to/avatar.png").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.UpdateAvatar(ctx, "path/to/avatar.png", 1)
	assert.NoError(t, err)
}

func TestUpdateAvatar_ExecError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE profiles SET  avatar_path = $2 WHERE user_id = $1`)).
		WillReturnError(errors.New("exec failed"))

	err := store.UpdateAvatar(ctx, "avatar.png", 1)
	assert.Error(t, err)
}

func TestGetShortProfileByUserIDs_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	userIDs := []int32{1, 2}
	rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path"}).
		AddRow(1, "John Doe", "avatar1.png").
		AddRow(2, "Jane Doe", "avatar2.png")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name || ' ' || last_name as full_name , avatar_path FROM profiles WHERE user_id = ANY($1)`)).
		WithArgs(pq.Array(userIDs)).
		WillReturnRows(rows)

	result, err := store.GetShortProfileByUserIDs(ctx, userIDs)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "John Doe", result[1].FullName)
}

func TestGetShortProfileByUserIDs_QueryError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	userIDs := []int32{1, 2}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name || ' ' || last_name as full_name , avatar_path FROM profiles WHERE user_id = ANY($1)`)).
		WillReturnError(errors.New("query failed"))

	_, err := store.GetShortProfileByUserIDs(ctx, userIDs)
	assert.Error(t, err)
}

func TestGetProfileByUserID_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	profile := domain.Profile{UserID: 1, FirstName: "John", LastName: "Doe"}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name, last_name, avatar_path, header_path, about_myself, gender, dob  FROM profiles WHERE user_id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "first_name", "last_name", "avatar_path", "header_path", "about_myself", "gender", "dob"}).
			AddRow(1, "John", "Doe", "avatar.png", "header.png", "About", "M", time.Now()))

	result, err := store.GetProfileByUserID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, profile.UserID, result.UserID)
	assert.Equal(t, profile.FirstName, result.FirstName)
	assert.Equal(t, profile.LastName, result.LastName)
}

func TestGetProfileByUserID_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name, last_name, avatar_path, header_path, about_myself, gender, dob  FROM profiles WHERE user_id = $1`)).
		WithArgs(1).
		WillReturnError(sql.ErrNoRows)

	_, err := store.GetProfileByUserID(ctx, 1)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetAvatarByUserID_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	avatar := "avatar.png"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT avatar_path FROM profiles WHERE user_id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"avatar_path"}).AddRow(avatar))

	result, err := store.GetAvatarByUserID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, &avatar, result)
}

func TestGetAvatarByUserID_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT avatar_path FROM profiles WHERE user_id = $1`)).
		WithArgs(1).
		WillReturnError(sql.ErrNoRows)

	result, err := store.GetAvatarByUserID(ctx, 1)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetHeaderByUserID_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	header := "header.png"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT header_path FROM profiles WHERE user_id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"header_path"}).AddRow(header))

	result, err := store.GetHeaderByUserID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, &header, result)
}

func TestGetHeaderByUserID_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT header_path FROM profiles WHERE user_id = $1`)).
		WithArgs(1).
		WillReturnError(sql.ErrNoRows)

	result, err := store.GetHeaderByUserID(ctx, 1)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
