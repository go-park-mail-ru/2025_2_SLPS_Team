package domain

import (
	"context"
	"mime/multipart"
	"time"
)

type Profile struct {
	UserID         int                 `json:"userID"`
	FirstName      string              `json:"firstName"`
	LastName       string              `json:"lastName"`
	AvatarPath     *string             `json:"avatarPath"`
	HeaderPath     *string             `json:"headerPath"`
	AboutMyself    *string             `json:"aboutMyself"`
	Gender         string              `json:"gender"`
	Dob            time.Time           `json:"dob"`
	RelationsCount UserRelationsCounts `json:"relationsCount"`
	RelationStatus FriendshipStatus    `json:"relationStatus"`
}

type ShortProfile struct {
	UserID     int       `json:"userID"`
	FullName   string    `json:"fullName"`
	AvatarPath *string   `json:"avatarPath"`
	Dob        time.Time `json:"dob"`
}

type ProfileService interface {
	UpdateProfile(ctx context.Context, profile Profile, userID int, files []*multipart.FileHeader) error
	UpdateAvatar(ctx context.Context, userID int, files []*multipart.FileHeader) error
	UpdateHeader(ctx context.Context, userID int, files []*multipart.FileHeader) error
	GetProfileByUserID(ctx context.Context, selfUserID, userID int) (*Profile, error)
	DeleteAvatarByUserID(ctx context.Context, userID int) error
}

type ElasticProfileStore interface {
	CreateProfile(ctx context.Context, fullName string, userID int) error
	UpdateProfile(ctx context.Context, fullName string, userID int) error
	DeleteProfile(ctx context.Context, userID int) error
	SearchProfileIDsByFullName(ctx context.Context, fullName string) ([]int, error)
}

type ProfileStore interface {
	UpdateProfile(ctx context.Context, profile Profile, userID int) error
	UpdateAvatar(ctx context.Context, avatarPath string, userID int) error
	UpdateHeader(ctx context.Context, avatarPath string, UserID int) error
	GetProfileByUserID(ctx context.Context, userID int) (Profile, error)
	GetShortProfileByUserIDs(ctx context.Context, userIDs []int) (map[int]ShortProfile, error)
	GetAvatarByUserID(ctx context.Context, userID int) (*string, error)
	GetHeaderByUserID(ctx context.Context, userID int) (*string, error)
	DeleteAvatarByUserID(ctx context.Context, userID int) (*string, error)
	//DeleteAvatar
	//DeleteHeader
}
