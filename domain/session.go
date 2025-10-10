package domain

type Session struct {
	ID        int
	SessionID string
	UserID    int
}
type SessionStore interface {
	AddSession(userID int, sessionID string) error
	GetSessionBySessionID(sessionID string) (Session, error)
	DeleteSession(sessionID string) error
}
