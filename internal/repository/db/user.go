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

type DBUserStore struct {
	db *sql.DB
}

func NewDBUserStore(db *sql.DB) domain.UserStore {
	return &DBUserStore{db: db}
}

func (store *DBUserStore) CreateUser(ctx context.Context, user domain.User) (int32, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "userStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CreateUser")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	tx, err := store.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}

	var userID int32
	queryUser := `INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id`
	err = tx.QueryRow(queryUser, user.Email, user.Password).Scan(&userID)
	if err != nil {
		tx.Rollback()
		dblogger.Error("Failed to insert user", zap.String("query", queryUser), zap.Error(err))
		return 0, fmt.Errorf("insert user: %w", err)
	}

	// Если всё ок, коммитим
	if err := tx.Commit(); err != nil {
		dblogger.Error("Failed to commit tx", zap.Error(err))
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	dblogger.Info("User created", zap.Int32("userID", userID))
	return userID, nil

}

func (store *DBUserStore) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "userStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetUserByEmail")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	query := `SELECT id, email, password FROM users WHERE email = $1`

	var user domain.User
	err := store.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password)
	dblogger = dblogger.With(zap.String("query", query))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dblogger.Info("User not found")
			return nil, domain.ErrNotFound
		}
		dblogger.Error("Failed to get User", zap.Error(err))
		return nil, err
	}

	dblogger.Info("User found and return")
	return &user, nil
}
func (store *DBUserStore) GetUserByID(ctx context.Context, userID int32) (domain.User, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "userStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetUserByID")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `SELECT id, email, password, role FROM users WHERE id = $1`

	var user domain.User
	err := store.db.QueryRow(query, userID).Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	dblogger = dblogger.With(zap.Int32("userID", userID), zap.String("query", query))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			dblogger.Info("User not found")
			return domain.User{}, domain.ErrNotFound
		}
		dblogger.Error("Failed to get User", zap.Error(err))
		return domain.User{}, err
	}

	dblogger.Info("User found and return")
	return user, nil
}

func (store *DBUserStore) IsUserExists(ctx context.Context, userID int32) (bool, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "userStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start IsUserExists")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"
	err := store.db.QueryRow(query, userID).Scan(&exists)
	dblogger = dblogger.With(zap.Int32("userID", userID), zap.String("query", query))
	if err != nil {
		dblogger.Error("failed to find user", zap.Error(err))
		return false, err
	}

	dblogger.Info("User find successfully")
	return exists, nil
}

func (store *DBUserStore) IsUserAdmin(ctx context.Context) (bool, error) {
	TempSessionInfo, _ := ctx.Value(domain.TempSessionCtxKey).(*domain.TempSessionInfo)
	if TempSessionInfo == nil {
		TempSessionInfo = &domain.TempSessionInfo{}
	}
	if TempSessionInfo.UserID != nil {
		user, err := store.GetUserByID(ctx, *TempSessionInfo.UserID)
		if err != nil {
			return false, err
		}
		if user.Role == "admin" {
			return true, nil
		}
	}
	return false, nil
}
