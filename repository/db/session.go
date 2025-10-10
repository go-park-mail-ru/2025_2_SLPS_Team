package db

import (
	"database/sql"
	"errors"
	"project/domain"
)

type DBSessionStore struct {
	db *sql.DB
}

func NewDBSessionStore(db *sql.DB) domain.SessionStore {
	return &DBSessionStore{db: db}
}

func (store *DBSessionStore) AddSession(userID int, sessionID string) error {
	querySession := `INSERT INTO sessions (session_id, user_id) VALUES ($1, $2)`
	_, err := store.db.Exec(querySession, sessionID, userID)
	return err
}

func (store *DBSessionStore) GetSessionBySessionID(sessionID string) (domain.Session, error) {
	query := `SELECT id, session_id, user_id FROM sessions WHERE session_id = $1`

	var session domain.Session
	err := store.db.QueryRow(query, sessionID).Scan(&session.ID, &session.SessionID, &session.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Session{}, domain.ErrNotFound
		}
		return domain.Session{}, err
	}

	return session, nil
}

func (store *DBSessionStore) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE session_id = $1`
	_, err := store.db.Exec(query, sessionID)
	return err
}
