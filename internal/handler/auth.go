package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"project/domain"
	"project/internal/service"
	"time"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	sessionStore domain.SessionStore
	userStore    domain.UserStore
}

func NewAuthHandler(userStore domain.UserStore, sessionStore domain.SessionStore) *AuthHandler {
	return &AuthHandler{
		sessionStore: sessionStore,
		userStore:    userStore,
	}
}

func (api *AuthHandler) IsLoggedIn(r *http.Request) (*domain.Session, error) {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		service.Error(r.Context(), "Cookie session_id not found:", err)
		return nil, domain.ErrNotFound
	}

	session, authorizedErr := api.sessionStore.GetSessionBySessionID(r.Context(), sessionCookie.Value)
	if authorizedErr != nil {
		service.Error(r.Context(), "Session not found or error:", authorizedErr)
		return nil, authorizedErr
	}

	service.Info(r.Context(), "Session loaded")
	return session, nil
}

type IsLoggedInResponse struct {
	UserID int `json:"userID"`
}

// IsLoggedInHandler проверяет, авторизован ли пользователь по cookie сессии.
// @Summary Проверить авторизацию пользователя
// @Description Возвращает ID пользователя, если сессия валидна
// @Tags auth
// @Produce json
// @Success 200 {object} IsLoggedInResponse "Пользователь авторизован"
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /auth/isloggedin [get]
func (api *AuthHandler) IsLoggedInHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	session, err := api.IsLoggedIn(r)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			sendJSONResponse(w, domain.NotFound, http.StatusNotFound)
			service.Warn(r.Context(), domain.NotFound)
			return
		} else {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "failed to check registration", err)
			return
		}
	}

	res := IsLoggedInResponse{
		UserID: session.UserID,
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", service.StructName(res)))
	}
	service.Info(r.Context(), "registration success")
}

func (api *AuthHandler) AddSession(w http.ResponseWriter, r *http.Request, userID int) error {
	tokens, err := api.sessionStore.AddSession(r.Context(), userID)
	if err != nil {
		return err
	}

	sessionCookie := &http.Cookie{
		Name:     "session_id",
		Value:    tokens.SID,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: true,
	}
	http.SetCookie(w, sessionCookie)

	CSRFCookie := &http.Cookie{
		Name:     "CSRF_token",
		Value:    tokens.CSRFToken,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: false,
	}
	http.SetCookie(w, CSRFCookie)
	return nil
}

type LoginRequest struct {
	Email    string `json:"email" valid:"required"`
	Password string `json:"password" valid:"required, stringlength(6|20)"`
}

// Login выполняет авторизацию пользователя и создает сессию.
// @Summary Авторизация пользователя
// @Description Логин с email и паролем, возвращает cookie сессии
// @Tags auth
// @Accept json
// @Produce json
// @Param loginRequest body LoginRequest true "Данные для входа"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /auth/login [post]
func (api *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", service.StructName(req)))
		return
	}

	user, err := api.userStore.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			sendJSONResponse(w, domain.UserNotExist, http.StatusBadRequest)
			service.Error(r.Context(), "User by email does not exist", err)
			return
		} else {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to get user by email", err)
			return
		}

	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		sendJSONResponse(w, domain.IncorrectPassword, http.StatusBadRequest)
		service.Warn(r.Context(), domain.IncorrectPassword)
		return
	}

	if err := api.AddSession(w, r, user.ID); err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to add session", err)
		return
	}

	sendJSONResponse(w, "User logged in", http.StatusOK)
	service.Info(r.Context(), "User logged in", zap.Int("userID", user.ID))
}

// Logout удаляет текущую сессию пользователя.
// @Summary Выход пользователя
// @Description Удаляет cookie сессии, разлогинивает пользователя
// @Tags auth
// @Produce json
// @Success 200 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Failure 403 {object} JSONResponse
// @Router /auth/logout [post]
func (api *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {

	session, err := r.Cookie("session_id")
	if err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		service.Error(r.Context(), "Failed to logout", err)
		return
	}
	err = api.sessionStore.DeleteSession(r.Context(), session.Value)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to logout", err)
		return
	}

	sessionCookie := &http.Cookie{
		Name:     "session_id",
		Value:    session.Value,
		Path:     "/",
		Expires:  time.Now().AddDate(0, 0, -1),
		HttpOnly: true,
	}
	http.SetCookie(w, sessionCookie)

	CSRFTokenCookie, _ := r.Cookie("CSRF_token")
	CSRFToken := &http.Cookie{
		Name:     "CSRF_token",
		Value:    CSRFTokenCookie.Value,
		Path:     "/",
		Expires:  time.Now().AddDate(0, 0, -1),
		HttpOnly: false,
	}
	http.SetCookie(w, CSRFToken)

	sendJSONResponse(w, "User logged out", http.StatusOK)
	service.Info(r.Context(), "User logged out")
}

type RegisterRequest struct {
	FirstName       string    `json:"firstName" valid:"required"`
	LastName        string    `json:"lastName" valid:"required"`
	Email           string    `json:"email" valid:"email, required" example:"example@example.ru"`
	Password        string    `json:"password" valid:"required, stringlength(5|20)" example:"123123"`
	ConfirmPassword string    `json:"confirmPassword" valid:"required, stringlength(5|20)" example:"123123"`
	Dob             time.Time `json:"dob" valid:"-" example:"1990-01-01T00:00:00Z"`
	Gender          string    `json:"gender" valid:"-"`
}

// Register регистрирует нового пользователя.
// @Summary Регистрация пользователя
// @Description Создает нового пользователя с профилем и устанавливает сессию
// @Tags auth
// @Accept json
// @Produce json
// @Param registerRequest body RegisterRequest true "Данные для регистрации"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Failure 403 {object} JSONResponse
// @Router /auth/register [post]
func (api *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", service.StructName(req)))
		return
	}

	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		sendJSONResponse(w, domain.InvalidData, http.StatusBadRequest)
		service.Error(r.Context(), "Register validate failed", err)
		return
	}

	_, err = api.userStore.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to get user by email", err)
			return
		}
	} else {
		sendJSONResponse(w, "User already exist", http.StatusBadRequest)
		service.Warn(r.Context(), "User already exist")
		return
	}

	if req.Password != req.ConfirmPassword {
		sendJSONResponse(w, "Password field doesn't match", http.StatusBadRequest)
		service.Info(r.Context(), "Register validate failed: password filed doesn't match")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to generate hashed password", err)
		return
	}
	user := domain.User{
		Email:    req.Email,
		Password: string(hashedPassword),
	}
	profile := domain.Profile{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Dob:       req.Dob,
		Gender:    req.Gender,
	}
	userID, err := api.userStore.CreateUser(r.Context(), user, profile)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to create user", err)
		return
	}

	if err := api.AddSession(w, r, userID); err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to add session", err)
		return
	}

	sendJSONResponse(w, "User created", http.StatusOK)
	service.Info(r.Context(), "User created, registration complete", zap.Int("userID", userID))
}
