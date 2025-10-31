package domain

import (
	"context"
	"time"
)

type Profile struct {
	UserID      int       `json:"userID"`
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	AvatarPath  *string   `json:"avatarPath"`
	HeaderPath  *string   `json:"headerPath"`
	AboutMyself *string   `json:"aboutMyself"`
	Gender      string    `json:"gender"`
	Dob         time.Time `json:"dob"`
}

type ShortProfile struct {
	UserID     int     `json:"userID"`
	FullName   string  `json:"fullName"`
	AvatarPath *string `json:"avatarPath"`
}
type ProfileStore interface {
	UpdateProfile(ctx context.Context, profile Profile, userID int) error
	UpdateAvatar(ctx context.Context, avatarPath string, userID int) error
	UpdateHeader(ctx context.Context, avatarPath string, UserID int) error
	GetProfileByUserID(ctx context.Context, userID int) (Profile, error)
	GetShortProfileByUserIDs(ctx context.Context, userIDs []int) ([]ShortProfile, error)
	GetAvatarByUserID(ctx context.Context, userID int) (*string, error)
	GetHeaderByUserID(ctx context.Context, userID int) (*string, error)
	//DeleteAvatar
	//DeleteHeader
}
