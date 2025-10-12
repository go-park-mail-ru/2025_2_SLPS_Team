package domain

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserStore interface {
	GetUserByEmail(email string) (User, error)
	CreateUser(user User, profile Profile) (int, error)
	GetUserByID(userID int) (User, error)
	IsUserExists(userID int) (bool, error)
	//UpdatePassword()
	//UpdateEmail()
	//DeleteUser()
}
