package domain

import "errors"

//Общие доменные ошибки
var (
	ErrNotFound     = errors.New("Объект не найден")
	ErrAccessDenied = errors.New("access denied")
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal server error") //Какая-то внутренняя ошибка
)

//Ошибки для ПОСТОВ
var (
	ErrPostNotFound      = errors.New("post not found")
	ErrPostTextTooShort  = errors.New("post text too short")
	ErrPostTextTooLong   = errors.New("post text too long")
	ErrPostTextEmpty     = errors.New("post text cannot be empty")
	ErrPostInvalidAuthor = errors.New("invalid author")
)

//Специфичные ошибки для пользователей
var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already taken") //Такой емейл уже занят
	ErrInvalidEmail = errors.New("invalid email")       //Кривой неправильный емейл
)
