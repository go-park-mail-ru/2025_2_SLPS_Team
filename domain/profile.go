package domain

import (
	"context"
	"time"
)

type Profile struct {
	UserID         int32               `json:"userID"`
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
	UserID     int32     `json:"userID"`
	FullName   string    `json:"fullName"`
	AvatarPath *string   `json:"avatarPath"`
	Dob        time.Time `json:"dob"`
}

type ProfileService interface {
	CreateProfile(ctx context.Context, profile Profile) error
	UpdateProfile(ctx context.Context, profile Profile, userID int32, files []*File) error
	UpdateAvatar(ctx context.Context, userID int32, files []*File) error
	UpdateHeader(ctx context.Context, userID int32, files []*File) error
	GetProfileByUserID(ctx context.Context, selfUserID, userID int32) (*Profile, error)
	DeleteAvatarByUserID(ctx context.Context, userID int32) error
	GetShortProfileMapByUserIDs(ctx context.Context, userIDs []int32) (map[int32]ShortProfile, error)
	GetShortProfileByUserIDs(ctx context.Context, userIDs []int32) ([]ShortProfile, error)
	GetOtherShortProfileByUserIDs(ctx context.Context, userIDs []int32, limit, offset int32) ([]ShortProfile, error)
}

type ElasticProfileStore interface {
	CreateProfile(ctx context.Context, fullName string, userID int32) error
	UpdateProfile(ctx context.Context, fullName string, userID int32) error
	DeleteProfile(ctx context.Context, userID int32) error
	SearchUserIDsByFullNameWithFilter(ctx context.Context, fullName string, filterIDs []int32, isTerms bool, limit, offset int32) ([]int32, error)
}

type ProfileStore interface {
	CreateProfile(ctx context.Context, profile Profile) error
	UpdateProfile(ctx context.Context, profile Profile, userID int32) error
	UpdateAvatar(ctx context.Context, avatarPath string, userID int32) error
	UpdateHeader(ctx context.Context, avatarPath string, UserID int32) error
	GetProfileByUserID(ctx context.Context, userID int32) (Profile, error)
	GetShortProfileMapByUserIDs(ctx context.Context, userIDs []int32) (map[int32]ShortProfile, error)
	GetShortProfileByUserIDs(ctx context.Context, userIDs []int32) ([]ShortProfile, error)
	GetOtherShortProfileByUserIDs(ctx context.Context, userIDs []int32, limit, offset int32) ([]ShortProfile, error)
	GetAvatarByUserID(ctx context.Context, userID int32) (*string, error)
	GetHeaderByUserID(ctx context.Context, userID int32) (*string, error)
	DeleteAvatarByUserID(ctx context.Context, userID int32) (*string, error)
	//DeleteAvatar
	//DeleteHeader
}
