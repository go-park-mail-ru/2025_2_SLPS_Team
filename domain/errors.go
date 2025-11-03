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
	ErrFriendRequestExists     = errors.New("friend request already exists") // Добавляем эту ошибку
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

func MapErrorToHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "Resource not found"

	case errors.Is(err, ErrAccessDenied):
		return http.StatusForbidden, "Access denied"

	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest, "Invalid input"

	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict, "Already exists"

	case errors.Is(err, ErrDB):
		return http.StatusInternalServerError, "Database error"

	// Добавляем обработку ошибок друзей
	case errors.Is(err, ErrFriendshipNotFound):
		return http.StatusNotFound, "Friendship not found"

	case errors.Is(err, ErrAlreadyFriends):
		return http.StatusConflict, "Already friends"

	case errors.Is(err, ErrFriendRequestPending):
		return http.StatusConflict, "Friend request already pending"

	case errors.Is(err, ErrFriendRequestExists):
		return http.StatusConflict, "Friend request already exists"

	case errors.Is(err, ErrCannotFriendSelf):
		return http.StatusBadRequest, "Cannot send friend request to yourself"

	case errors.Is(err, ErrFriendshipBlocked):
		return http.StatusForbidden, "Friendship is blocked"

	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound, "User not found"

	case errors.Is(err, ErrPostNotFound):
		return http.StatusNotFound, "Post not found"

	default:
		return http.StatusInternalServerError, "Internal server error"
	}
}