package domain

import (
	"context"
	"time"
)

type RegisterRequest struct {
	FirstName       string    `json:"firstName" valid:"required"`
	LastName        string    `json:"lastName" valid:"required"`
	Email           string    `json:"email" valid:"email, required" example:"example@example.ru"`
	Password        string    `json:"password" valid:"required, stringlength(5|20)" example:"123123"`
	ConfirmPassword string    `json:"confirmPassword" valid:"required, stringlength(5|20)" example:"123123"`
	Dob             time.Time `json:"dob" valid:"-" example:"1990-01-01T00:00:00Z"`
	Gender          string    `json:"gender" valid:"-"`
}
type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type Session struct {
	UserID    int    `json:"userID"`
	CSRFToken string `json:"CSRFToken"`
}

type SIDAndSCRFToken struct {
	CSRFToken string `json:"CSRFToken"`
	SID       string `json:"SID"`
}
type AuthService interface {
	IsLoggedIn(ctx context.Context, sessionCookie string) (*Session, error)
	AddSession(ctx context.Context, userID int) (*SIDAndSCRFToken, error)
	Login(ctx context.Context, req User) (int, error)
	Logout(ctx context.Context, sessionCookie string) error
	Register(ctx context.Context, req RegisterRequest) (int, error)
	GetUserRole(ctx context.Context, userID int) (string, error)
	IsUserExists(ctx context.Context, userID int) (bool, error)
}
