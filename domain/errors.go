package domain

import (
	"errors"
	"net/http"
)

// общие ошибки на уровне сервиса и бд
// Общие доменные ошибки
var (
	ErrNotFound      = errors.New("not found")
	ErrAccessDenied  = errors.New("access denied")
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotExist      = errors.New("not exist")
	ErrDB            = errors.New("db error")
	ErrAlreadyExists = errors.New("already exist")
	ErrService       = errors.New("service error") //Какая-то внутренняя ошибка
)

// Ошибки для ПОСТОВ
var (
	ErrPostNotFound      = errors.New("post not found")
	ErrPostTextTooShort  = errors.New("post text too short")
	ErrPostTextTooLong   = errors.New("post text too long")
	ErrPostTextEmpty     = errors.New("post text cannot be empty")
	ErrPostInvalidAuthor = errors.New("invalid author")
)

// Ошибки для ДРУЗЕЙ
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

// общие сообщения пользователю
const (
	InvalidJSON           = "Invalid JSON"
	Unauthorized          = "Unauthorized"
	NotFound              = "Resource Not found"
	ServerErr             = "Internal Server error"
	UserNotExist          = "User doesn't exist"
	IncorrectPassword     = "Incorrect password"
	FailToEncode          = "Fail to ecnode in JSON"
	InvalidData           = "Invalid data"
	InvalidParams         = "Invalid query parameters"
	Forbidden             = "Access denied"
	AleradyExist          = "Already exist"
	FriendRequestSent     = "Friend request sent successfully"
	FriendRequestAccepted = "Friend request accepted successfully"
	FriendRequestRejected = "Friend request rejected successfully"
	FriendRemoved         = "Friend removed successfully"
)

func MapErrorToHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, NotFound
	case errors.Is(err, ErrAccessDenied):
		return http.StatusForbidden, Forbidden
	case errors.Is(err, ErrService):
		return http.StatusInternalServerError, ServerErr
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest, InvalidData
	case errors.Is(err, ErrNotExist):
		return http.StatusBadRequest, InvalidData
	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict, AleradyExist
	case errors.Is(err, ErrDB):
		return http.StatusInternalServerError, ServerErr

	default:
		return http.StatusInternalServerError, ServerErr
	}
}
