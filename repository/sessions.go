package repository

import (
	"sync"
)

type Session struct {
	ID     string
	UserId uint
}

type SessionStore struct {
	sessions map[string]Session
	mu       sync.RWMutex
}

func NewSessionStore(sessions map[string]Session) *SessionStore {
	return &SessionStore{
		sessions: sessions,
		mu:       sync.RWMutex{},
	}
}

func (store *SessionStore) AddSession(userID uint, sessionID string) string {
	store.mu.Lock()
	defer store.mu.Unlock()
	for {
		if _, exists := store.sessions[sessionID]; !exists {

			session := Session{
				ID:     sessionID,
				UserId: userID,
			}

			store.sessions[sessionID] = session

			return sessionID
		}
	}

}

func (store *SessionStore) GetSessionByID(sessionID string) (Session, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	session, ok := store.sessions[sessionID]
	if !ok {
		return Session{}, false
	}

	return session, ok
}

func (store *SessionStore) DeleteSession(sessionID string) {
	store.mu.Lock()
	delete(store.sessions, sessionID)
	store.mu.Unlock()
}
