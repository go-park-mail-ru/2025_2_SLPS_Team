package db

import (
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
)

type DBUserStore struct {
	db *sql.DB
}

func NewDBUserStore(db *sql.DB) domain.UserStore {
	return &DBUserStore{db: db}
}

func (store *DBUserStore) CreateUser(user domain.User, profile domain.Profile) (int, error) {
	tx, err := store.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var userID int
	queryUser := `INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id`
	err = tx.QueryRow(queryUser, user.Email, user.Password).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}

	queryProfile := `INSERT INTO profiles 
        (user_id, first_name, last_name, gender, dob) 
        VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.Exec(queryProfile, userID, profile.FirstName, profile.LastName, profile.Gender, profile.Dob)
	if err != nil {
		return 0, fmt.Errorf("insert profile: %w", err)
	}

	return userID, nil
}

func (store *DBUserStore) GetUserByEmail(email string) (domain.User, error) {
	query := `SELECT id, email, password FROM users WHERE email = $1`

	var user domain.User
	err := store.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}

	return user, nil
}
func (store *DBUserStore) GetUserByID(userID int) (domain.User, error) {
	query := `SELECT id, email, password FROM users WHERE id = $1`

	var user domain.User
	err := store.db.QueryRow(query, userID).Scan(&user.ID, &user.Email, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}

	return user, nil
}

func (store *DBUserStore) IsUserExists(userID int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"
	err := store.db.QueryRow(query, userID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
