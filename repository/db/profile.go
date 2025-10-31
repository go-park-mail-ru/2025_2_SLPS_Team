package db

import (
	"context"
	"database/sql"
	"errors"
	"project/domain"
	"project/internal/service"
	"time"

	"go.uber.org/zap"
)

type DBProfileStore struct {
	db *sql.DB
}

func NewDBProfileStore(db *sql.DB) domain.ProfileStore {
	return &DBProfileStore{db: db}
}

func (store *DBProfileStore) UpdateProfile(ctx context.Context, profile domain.Profile, userID int) error {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "profileStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateProfile")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	queryProfile := `UPDATE profiles SET first_name = $2, last_name = $3, gender = $4, dob = $5, about_myself = $6
WHERE user_id = $1`
	_, err := store.db.Exec(queryProfile,
		userID,
		profile.FirstName,
		profile.LastName,
		profile.Gender,
		profile.Dob,
		profile.AboutMyself)

	dblogger = dblogger.With(zap.Int("userID", userID), zap.String("query", queryProfile))
	if err != nil {
		dblogger.Error("Failed to update profile", zap.Error(err))
	} else {
		dblogger.Info("Profile updated")
	}
	return err
}

func (store *DBProfileStore) UpdateAvatar(ctx context.Context, avatarPath string, userID int) error {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "profileStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateAvatar")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	queryProfile := `UPDATE profiles SET  avatar_path = $2 WHERE user_id = $1`
	_, err := store.db.Exec(queryProfile, userID, avatarPath)

	dblogger = dblogger.With(zap.Int("userID", userID), zap.String("query", queryProfile))
	if err != nil {
		dblogger.Error("Failed to update avatar", zap.Error(err))
	} else {
		dblogger.Info("Avatar updated")
	}

	return err
}

func (store *DBProfileStore) UpdateHeader(ctx context.Context, headerPath string, userID int) error {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "profileStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateHeader")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	queryProfile := `UPDATE profiles SET  header_path = $2 WHERE user_id = $1`
	_, err := store.db.Exec(queryProfile, userID, headerPath)

	dblogger = dblogger.With(zap.Int("userID", userID), zap.String("query", queryProfile))
	if err != nil {
		dblogger.Error("Failed to update header", zap.Error(err))
	} else {
		dblogger.Info("Header updated")
	}

	return err
}

func (store *DBProfileStore) GetShortProfileByUserIDs(ctx context.Context, userIDs []int) ([]domain.ShortProfile, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "profileStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetShortProfileByUserIDs")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	if len(userIDs) == 0 {
		return []domain.ShortProfile{}, nil
	}

	query := `SELECT user_id, first_name || ' ' || last_name as full_name , avatar_path FROM profiles WHERE user_id = ANY($1)`

	dblogger = dblogger.With(zap.Ints("userIDs", userIDs), zap.String("query", query))

	rows, err := store.db.Query(query, userIDs)
	if err != nil {
		dblogger.Error("Failed to get profiles by user ids", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var users []domain.ShortProfile

	for rows.Next() {
		var u domain.ShortProfile
		err := rows.Scan(&u.UserID, &u.FullName, &u.AvatarPath)
		if err != nil {
			dblogger.Error("Failed to read profile rows", zap.Error(err))
			return nil, err
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		dblogger.Error("Failed to read profile rows", zap.Error(err))
		return nil, err
	}

	return users, nil
}

func (store *DBProfileStore) GetProfileByUserID(ctx context.Context, userID int) (domain.Profile, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "profileStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetProfileByUserID")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `SELECT user_id, first_name, last_name, avatar_path, header_path, about_myself, gender, dob  FROM profiles WHERE user_id = $1`
	dblogger = dblogger.With(zap.Int("userID", userID), zap.String("query", query))
	//добавить null проверку легче всего просто добавить указатели на возможные нул поля
	// можно возвращать указатель на объект так будет проще понять что его нет
	var profile domain.Profile
	err := store.db.QueryRow(query, userID).Scan(
		&profile.UserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.AvatarPath,
		&profile.HeaderPath,
		&profile.AboutMyself,
		&profile.Gender,
		&profile.Dob)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dblogger.Info("Profile not found")
			return domain.Profile{}, domain.ErrNotFound
		}
		dblogger.Error("Failed to get profile", zap.Error(err))
		return domain.Profile{}, err
	}

	dblogger.Info("Profile found and return")
	return profile, nil
}

func (store *DBProfileStore) GetAvatarByUserID(ctx context.Context, userID int) (*string, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "profileStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetAvatarByUserID")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	query := `SELECT avatar_path FROM profiles WHERE user_id = $1`
	dblogger = dblogger.With(zap.Int("userID", userID), zap.String("query", query))
	var avatar *string
	err := store.db.QueryRow(query, userID).Scan(
		&avatar)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dblogger.Info("Avatar not found")
			return nil, domain.ErrNotFound
		}
		dblogger.Error("Failed to get avatar", zap.Error(err))
		return nil, err
	}

	dblogger.Info("Avatar found and return")
	return avatar, nil
}

func (store *DBProfileStore) GetHeaderByUserID(ctx context.Context, userID int) (*string, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "profileStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetHeaderByUserID")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	query := `SELECT header_path FROM profiles WHERE user_id = $1`
	dblogger = dblogger.With(zap.Int("userID", userID), zap.String("query", query))
	var header *string
	err := store.db.QueryRow(query, userID).Scan(
		&header)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dblogger.Info("Header not found")
			return nil, domain.ErrNotFound
		}
		dblogger.Error("Failed to get header", zap.Error(err))
		return nil, err
	}

	dblogger.Info("Header found and return")
	return header, nil
}
