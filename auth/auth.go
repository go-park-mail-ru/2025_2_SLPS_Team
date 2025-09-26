package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

type Session struct {
	ID     string
	UserId uint
}

type SessionStore struct {
	Sessions map[string]*Session
	mu       sync.RWMutex
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		Sessions: make(map[string]*Session),
		mu:       sync.RWMutex{},
	}
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)

	cryptoReader := rand.Reader
	_, err := cryptoReader.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

func (store *SessionStore) AddSession(userID uint) (string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}

	session := Session{
		ID:     sessionID,
		UserId: userID,
	}

	store.Sessions[sessionID] = &session

	return sessionID, nil
}

func (store *SessionStore) GetSessionByID(sessionID string) (Session, bool) {
	store.mu.RLock()
	defer store.mu.Unlock()

	session, ok := store.Sessions[sessionID]
	if !ok {
		return Session{}, false
	}

	return *session, ok
}

func (store *SessionStore) DeleteSession(sessionID string) {
	store.mu.Lock()
	delete(store.Sessions, sessionID)
	store.mu.Unlock()
}

type User struct {
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	HashedPassword string `json:"password"`
}

type UserStore struct {
	Users  map[string]*User
	NextID uint
	mu     sync.RWMutex
}

func NewUserStore() *UserStore {
	return &UserStore{
		Users:  make(map[string]*User),
		NextID: 1,
		mu:     sync.RWMutex{},
	}
}

func (store *UserStore) AddUser(username, email, hashedPassword string) string {
	store.mu.Lock()
	user := User{
		ID:             store.NextID,
		Username:       username,
		Email:          email,
		HashedPassword: hashedPassword,
	}

	store.Users[user.Username] = &user
	store.NextID++
	store.mu.Unlock()

	return user.Username
}

func (store *UserStore) GetUserByUsername(username string) (User, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	user, ok := store.Users[username]
	if !ok {
		return User{}, false
	}

	return *user, ok
}
