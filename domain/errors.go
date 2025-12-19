package domain

import (
	"errors"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//easyjson:json
type JSONResponse struct {
	Message string `json:"message"`
	Code    int32  `json:"code"`
}

// общие ошибки на уровне сервиса и бд
// Общие доменные ошибки
var (
	ErrNotFound      = errors.New("not found")
	ErrAccessDenied  = errors.New("access denied")
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotExist      = errors.New("not exist")
	ErrDB            = errors.New("dbconn error")
	ErrAlreadyExists = errors.New("already exist")
	ErrService       = errors.New("service error") //Какая-то внутренняя ошибк
	ErrInvalidParams = errors.New("invalid params")
)

// Ошибки для ПОСТОВ
var (
	ErrPostNotFound      = errors.New("post not found")
	ErrPostTextTooLong   = errors.New("post text too long")
	ErrPostInvalidAuthor = errors.New("invalid author")
)

// Ошибки для ДРУЗЕЙ
var (
	ErrFriendshipNotFound = errors.New("friendship not found")
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

func MapErrorToHTTP(err error) (int32, string) {
	switch {
	case errors.Is(err, ErrNotFound) || errors.Is(err, ErrPostNotFound):
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
	case errors.Is(err, ErrInvalidParams):
		return http.StatusBadRequest, InvalidParams

	default:
		return http.StatusInternalServerError, ServerErr
	}
}
func ToGrpcError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrNotExist):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, ErrDB):
		return status.Error(codes.Internal, "database error")

	case errors.Is(err, ErrService):
		return status.Error(codes.Internal, "internal service error")

	default:
		return status.Error(codes.Unknown, err.Error())
	}
}
func FromGrpcError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return ErrService
	}

	switch st.Code() {
	case codes.NotFound:
		return ErrNotFound

	case codes.PermissionDenied:
		return ErrAccessDenied

	case codes.InvalidArgument:
		return ErrInvalidInput

	case codes.AlreadyExists:
		return ErrAlreadyExists

	case codes.Unauthenticated:
		return ErrAccessDenied

	case codes.FailedPrecondition:
		return ErrInvalidInput

	case codes.Internal:
		// различать БД и сервис?
		if strings.Contains(st.Message(), "database error") {
			return ErrDB
		}
		return ErrService

	case codes.Unavailable:
		return ErrService

	default:
		return ErrService
	}
}
