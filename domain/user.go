package domain

import (
	"context"
)

//easyjson:json
type User struct {
	ID       int32  `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UserStore interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user User) (int32, error)
	GetUserByID(ctx context.Context, userID int32) (User, error)
	IsUserExists(ctx context.Context, userID int32) (bool, error)
	IsUserAdmin(ctx context.Context) (bool, error)
	//UpdatePassword()
	//UpdateEmail()
	//DeleteUser()
}
