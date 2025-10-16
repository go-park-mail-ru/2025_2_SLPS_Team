package domain

import "time"

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
	FirstName  string  `json:"firstName"`
	LastName   string  `json:"lastName"`
	AvatarPath *string `json:"avatarPath"`
}
type ProfileStore interface {
	UpdateProfile(profile Profile, userID int) error
	UpdateAvatar(avatarPath string, userID int) error
	UpdateHeader(avatarPath string, UserID int) error
	GetProfileByUserID(userID int) (Profile, error)
	GetShortProfileByUserIDs(userIDs []int) ([]ShortProfile, error)
	GetAvatarByUserID(userID int) (*string, error)
	GetHeaderByUserID(userID int) (*string, error)
	//DeleteAvatar
	//DeleteHeader
}
