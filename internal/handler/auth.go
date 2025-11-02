package handler

import (
	"encoding/json"
	"net/http"
	"project/domain"
	"project/internal/service"
	"time"

	"go.uber.org/zap"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (api *AuthHandler) IsLoggedIn(r *http.Request) (*domain.Session, error) {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		service.Error(r.Context(), "Cookie session_id not found:", err)
		return nil, domain.ErrNotFound
	}
	session, err := api.authService.IsLoggedIn(r.Context(), sessionCookie)
	if err != nil {
		service.Error(r.Context(), "Session not found or error:", err)
		return nil, err
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
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		sendJSONError(w, err)
		return
	}
	session, err := api.authService.IsLoggedIn(r.Context(), sessionCookie)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	res := IsLoggedInResponse{
		UserID: session.UserID,
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		service.FromContext(r.Context()).Error(domain.FailToEncode, zap.Error(err), zap.String("struct", service.StructName(res)))
	}
	service.FromContext(r.Context()).Info("registration success")
}

func (api *AuthHandler) AddSession(w http.ResponseWriter, r *http.Request, userID int) error {
	tokens, err := api.authService.AddSession(r.Context(), userID)
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
	var req domain.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		service.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", service.StructName(req)))
		return
	}
	userID, err := api.authService.Login(r.Context(), req)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	err = api.AddSession(w, r, userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "User logged in", http.StatusOK)
	service.FromContext(r.Context()).Info("User logged in", zap.Int("userID", userID))
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
		service.FromContext(r.Context()).Error("Failed to logout", zap.Error(err))
		return
	}
	err = api.authService.Logout(r.Context(), session)
	if err != nil {
		sendJSONError(w, err)
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
	service.FromContext(r.Context()).Info("User loggged out")
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
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		service.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", service.StructName(req)))
		return
	}
	userID, err := api.authService.Register(r.Context(), req)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	err = api.AddSession(w, r, userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "User created", http.StatusOK)
	service.FromContext(r.Context()).Info("User created, registration complete", zap.Int("userID", userID))
}
