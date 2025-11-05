package domain

import "context"

type Session struct {
	UserID    int    `json:"userID"`
	CSRFToken string `json:"CSRFToken"`
}

type SIDAndSCRFToken struct {
	CSRFToken string `json:"CSRFToken"`
	SID       string `json:"SID"`
}
type SessionStore interface {
	AddSession(ctx context.Context, userID int) (*SIDAndSCRFToken, error)
	GetSessionBySessionID(ctx context.Context, sessionID string) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
}
