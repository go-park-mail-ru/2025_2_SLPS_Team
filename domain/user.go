package domain

import (
	"context"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UserStore interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user User, profile Profile) (int, error)
	GetUserByID(ctx context.Context, userID int) (User, error)
	IsUserExists(ctx context.Context, userID int) (bool, error)
	IsUserAdmin(ctx context.Context) (bool, error)
	//UpdatePassword()
	//UpdateEmail()
	//DeleteUser()
}
