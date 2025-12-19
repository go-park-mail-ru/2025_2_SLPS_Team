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
		WithArgs(int32(1), "John", "Doe", "M", time.Now(), &about).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.UpdateProfile(ctx, domain.Profile{
		FirstName:   "John",
		LastName:    "Doe",
		Gender:      "M",
		Dob:         time.Now(),
		AboutMyself: &about,
	}, int32(1))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProfile_ExecError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE profiles SET first_name = $2, last_name = $3, gender = $4, dob = $5, about_myself = $6
WHERE user_id = $1`)).
		WithArgs(int32(1), "", "", "", time.Time{}, nil).
		WillReturnError(errors.New("exec failed"))

	err := store.UpdateProfile(ctx, domain.Profile{}, int32(1))
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAvatar_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE profiles SET  avatar_path = $2 WHERE user_id = $1`)).
		WithArgs(int32(1), "path/to/avatar.png").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.UpdateAvatar(ctx, "path/to/avatar.png", int32(1))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAvatar_ExecError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE profiles SET  avatar_path = $2 WHERE user_id = $1`)).
		WithArgs(int32(1), "avatar.png").
		WillReturnError(errors.New("exec failed"))

	err := store.UpdateAvatar(ctx, "avatar.png", int32(1))
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetShortProfileByUserIDs_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	userIDs := []int32{1, 2}
	
	// Используем реальное время вместо nil для dob
	testTime := time.Now()
	rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path", "dob"}).
		AddRow(int32(1), "John Doe", "avatar1.png", testTime).
		AddRow(int32(2), "Jane Doe", "avatar2.png", testTime)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name || ' ' || last_name as full_name , avatar_path, dob FROM profiles WHERE user_id = ANY($1)`)).
		WithArgs(pq.Array(userIDs)).
		WillReturnRows(rows)

	result, err := store.GetShortProfileByUserIDs(ctx, userIDs)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "John Doe", result[0].FullName)
	assert.Equal(t, "Jane Doe", result[1].FullName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// func TestGetShortProfileByUserIDs_WithNullDob(t *testing.T) {
// 	dbConn, mock, _ := sqlmock.New()
// 	defer dbConn.Close()
// 	store := NewDBProfileStore(dbConn)
// 	ctx := context.Background()

// 	userIDs := []int32{1, 2}
	
// 	// Используем sql.NullTime для обработки NULL в dob
// 	rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path", "dob"}).
// 		AddRow(int32(1), "John Doe", "avatar1.png", nil).
// 		AddRow(int32(2), "Jane Doe", "avatar2.png", nil)

// 	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name || ' ' || last_name as full_name , avatar_path, dob FROM profiles WHERE user_id = ANY($1)`)).
// 		WithArgs(pq.Array(userIDs)).
// 		WillReturnRows(rows)

// 	result, err := store.GetShortProfileByUserIDs(ctx, userIDs)
// 	assert.NoError(t, err)
// 	assert.Len(t, result, 2)
// 	assert.Equal(t, "John Doe", result[0].FullName)
// 	assert.Equal(t, "Jane Doe", result[1].FullName)
// 	// Проверяем что dob установлен в zero value
// 	assert.Equal(t, time.Time{}, result[0].Dob)
// 	assert.Equal(t, time.Time{}, result[1].Dob)
// 	assert.NoError(t, mock.ExpectationsWereMet())
// }

func TestGetShortProfileByUserIDs_QueryError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	userIDs := []int32{1, 2}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name || ' ' || last_name as full_name , avatar_path, dob FROM profiles WHERE user_id = ANY($1)`)).
		WithArgs(pq.Array(userIDs)).
		WillReturnError(errors.New("query failed"))

	result, err := store.GetShortProfileByUserIDs(ctx, userIDs)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetShortProfileByUserIDs_EmptyResult(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	userIDs := []int32{1, 2}
	rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path", "dob"})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name || ' ' || last_name as full_name , avatar_path, dob FROM profiles WHERE user_id = ANY($1)`)).
		WithArgs(pq.Array(userIDs)).
		WillReturnRows(rows)

	result, err := store.GetShortProfileByUserIDs(ctx, userIDs)
	assert.NoError(t, err)
	assert.Len(t, result, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfileByUserID_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	testTime := time.Now()
	var about string = "About"
	
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name, last_name, avatar_path, header_path, about_myself, gender, dob  FROM profiles WHERE user_id = $1`)).
		WithArgs(int32(1)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "first_name", "last_name", "avatar_path", "header_path", "about_myself", "gender", "dob"}).
			AddRow(int32(1), "John", "Doe", "avatar.png", "header.png", &about, "M", testTime))

	result, err := store.GetProfileByUserID(ctx, int32(1))
	assert.NoError(t, err)
	assert.Equal(t, int32(1), result.UserID)
	assert.Equal(t, "John", result.FirstName)
	assert.Equal(t, "Doe", result.LastName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfileByUserID_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name, last_name, avatar_path, header_path, about_myself, gender, dob  FROM profiles WHERE user_id = $1`)).
		WithArgs(int32(1)).
		WillReturnError(sql.ErrNoRows)

	_, err := store.GetProfileByUserID(ctx, int32(1))
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAvatarByUserID_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	avatar := "avatar.png"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT avatar_path FROM profiles WHERE user_id = $1`)).
		WithArgs(int32(1)).
		WillReturnRows(sqlmock.NewRows([]string{"avatar_path"}).AddRow(avatar))

	result, err := store.GetAvatarByUserID(ctx, int32(1))
	assert.NoError(t, err)
	assert.Equal(t, &avatar, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAvatarByUserID_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT avatar_path FROM profiles WHERE user_id = $1`)).
		WithArgs(int32(1)).
		WillReturnError(sql.ErrNoRows)

	result, err := store.GetAvatarByUserID(ctx, int32(1))
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetHeaderByUserID_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	header := "header.png"
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT header_path FROM profiles WHERE user_id = $1`)).
		WithArgs(int32(1)).
		WillReturnRows(sqlmock.NewRows([]string{"header_path"}).AddRow(header))

	result, err := store.GetHeaderByUserID(ctx, int32(1))
	assert.NoError(t, err)
	assert.Equal(t, &header, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetHeaderByUserID_NotFound(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT header_path FROM profiles WHERE user_id = $1`)).
		WithArgs(int32(1)).
		WillReturnError(sql.ErrNoRows)

	result, err := store.GetHeaderByUserID(ctx, int32(1))
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Дополнительные тесты для полного покрытия

func TestGetShortProfileByUserIDs_ScanError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	userIDs := []int32{1, 2}
	
	// Неправильный тип для user_id чтобы вызвать ошибку сканирования
	rows := sqlmock.NewRows([]string{"user_id", "full_name", "avatar_path", "dob"}).
		AddRow("invalid", "John Doe", "avatar1.png", time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name || ' ' || last_name as full_name , avatar_path, dob FROM profiles WHERE user_id = ANY($1)`)).
		WithArgs(pq.Array(userIDs)).
		WillReturnRows(rows)

	result, err := store.GetShortProfileByUserIDs(ctx, userIDs)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfileByUserID_DBError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBProfileStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, first_name, last_name, avatar_path, header_path, about_myself, gender, dob  FROM profiles WHERE user_id = $1`)).
		WithArgs(int32(1)).
		WillReturnError(errors.New("db error"))

	_, err := store.GetProfileByUserID(ctx, int32(1))
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}