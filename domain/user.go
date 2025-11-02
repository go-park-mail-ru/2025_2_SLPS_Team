package domain

import (
	"context"
	"time"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	FirstName       string    `json:"firstName" valid:"required"`
	LastName        string    `json:"lastName" valid:"required"`
	Email           string    `json:"email" valid:"email, required" example:"example@example.ru"`
	Password        string    `json:"password" valid:"required, stringlength(5|20)" example:"123123"`
	ConfirmPassword string    `json:"confirmPassword" valid:"required, stringlength(5|20)" example:"123123"`
	Dob             time.Time `json:"dob" valid:"-" example:"1990-01-01T00:00:00Z"`
	Gender          string    `json:"gender" valid:"-"`
}

type UserStore interface {
	GetUserByEmail(ctx context.Context, email string) (User, error)
	CreateUser(ctx context.Context, user User, profile Profile) (int, error)
	GetUserByID(ctx context.Context, userID int) (User, error)
	IsUserExists(ctx context.Context, userID int) (bool, error)
	//UpdatePassword()
	//UpdateEmail()
	//DeleteUser()
}
