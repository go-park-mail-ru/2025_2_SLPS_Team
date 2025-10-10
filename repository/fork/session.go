package fork

import (
	"project/domain"
	"sync"
)

type SessionStore struct {
	sessions map[string]domain.Session
	mu       sync.RWMutex
}
type Session struct {
	ID        int
	SessionId string
	UserId    int
}

func NewSessionStore(sessions map[string]domain.Session) *SessionStore {
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

			session := domain.Session{
				ID:     sessionID,
				UserId: userID,
			}

			store.sessions[sessionID] = session

			return sessionID
		}
	}

}

func (store *SessionStore) GetSessionByID(sessionID string) (domain.Session, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	session, ok := store.sessions[sessionID]
	if !ok {
		return domain.Session{}, false
	}

	return session, ok
}

func (store *SessionStore) DeleteSession(sessionID string) {
	store.mu.Lock()
	delete(store.sessions, sessionID)
	store.mu.Unlock()
}
