package domain

import "errors"

// Общие доменные ошибки
var (
	ErrNotFound     = errors.New("Объект не найден")
	ErrAccessDenied = errors.New("access denied")
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal server error") //Какая-то внутренняя ошибка
)

// Ошибки для ПОСТОВ
var (
	ErrPostNotFound      = errors.New("post not found")
	ErrPostTextTooShort  = errors.New("post text too short")
	ErrPostTextTooLong   = errors.New("post text too long")
	ErrPostTextEmpty     = errors.New("post text cannot be empty")
	ErrPostInvalidAuthor = errors.New("invalid author")
)

//Ошибки для ДРУЗЕЙ
var (
	ErrFriendshipNotFound      = errors.New("friendship not found")
	ErrAlreadyFriends          = errors.New("users are already friends")
	ErrFriendRequestPending    = errors.New("friend request already pending")
	ErrFriendRequestExists     = errors.New("friend request already exists")
	ErrCannotFriendSelf        = errors.New("cannot send friend request to yourself")
	ErrInvalidFriendshipStatus = errors.New("invalid friendship status")
	ErrFriendshipBlocked       = errors.New("friendship is blocked")
	
)

// Ошибки для пользователей
var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already taken") //Такой емейл уже занят
	ErrInvalidEmail = errors.New("invalid email")       //Кривой неправильный емейл
)

const (
	InvalidJSON           = "Invalid JSON"
	Unauthorized          = "Unauthorized"
	NotFound              = "Not found"
	ServerErr             = "Internal Server error"
	UserNotExist          = "User doesn't exist"
	IncorrectPassword     = "Incorrect password"
	FailToEncode          = "Fail to ecnode in JSON"
	InvalidData           = "Invalid data"
	InvalidParams         = "Invalid query parameters"
	Forbidden             = "Forbidden"
	FriendRequestSent     = "Friend request sent successfully"
	FriendRequestAccepted = "Friend request accepted successfully"
	FriendRequestRejected = "Friend request rejected successfully"
	FriendRemoved         = "Friend removed successfully"
)
