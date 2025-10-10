package domain

import "time"

type Profile struct {
	UserID      int       `json:"userID"`
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	AvatarPath  string    `json:"avatarPath"`
	HeaderPath  string    `json:"headerPath"`
	AboutMyself string    `json:"aboutMyself"`
	Gender      string    `json:"gender"`
	Dob         time.Time `json:"dob"`
}

type ProfileStore interface {
	GetUserByEmail(email string) (User, bool)
	AddUser(firstname, lastname, email, gender, hashedPassword string, age int) string
}
