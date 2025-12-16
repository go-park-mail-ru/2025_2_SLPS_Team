package handler

import (
	"net/http"
	"project/config"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"
	"time"

	"go.uber.org/zap"
)

type AuthHandler struct {
	authService pb.AuthServiceClient
	config      *config.Config
}

func NewAuthHandler(authService pb.AuthServiceClient, config *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		config:      config,
	}
}

func (api *AuthHandler) IsLoggedIn(r *http.Request) (*domain.Session, error) {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		domain.FromContext(r.Context()).Info("Cookie session_id not found:", zap.Error(err))
		return nil, domain.ErrNotFound
	}
	resp, err := api.authService.IsLoggedIn(r.Context(), &pb.SessionCookieRequest{SessionCookie: sessionCookie.Value})
	if err != nil {
		err = domain.FromGrpcError(err)
		return nil, err
	}

	return &domain.Session{UserID: resp.UserId, CSRFToken: resp.CsrfToken}, err
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
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		sendJSONError(w, domain.ErrNotFound)
		return
	}

	resp, err := api.authService.IsLoggedIn(r.Context(), &pb.SessionCookieRequest{SessionCookie: sessionCookie.Value})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	resp2, err := api.authService.GetUserRole(r.Context(), &pb.UserIDRequest{UserId: resp.UserId})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	res := domain.IsLoggedInResponse{
		UserID: resp.UserId,
		Role:   resp2.Role,
	}

	sendJSONData(r.Context(), w, res)
}

func (api *AuthHandler) AddSession(w http.ResponseWriter, r *http.Request, userID int32) error {
	resp, err := api.authService.AddSession(r.Context(), &pb.UserIDRequest{UserId: userID})
	if err != nil {
		err = domain.FromGrpcError(err)
		return err
	}

	tokens := domain.SIDAndSCRFToken{SID: resp.Sid, CSRFToken: resp.CsrfToken}
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

// Login выполняет авторизацию пользователя и создает сессию.
// @Summary Авторизация пользователя
// @Description Логин с email и паролем, возвращает cookie сессии
// @Tags auth
// @Accept json
// @Produce json
// @Param loginRequest body domain.User true "Данные для входа"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /auth/login [post]
func (api *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	req, err := DecodeJSONBody[domain.User](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	resp, err := api.authService.Login(r.Context(), &pb.LoginRequest{Email: req.Email, Password: req.Password})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	userID := resp.UserId
	err = api.AddSession(w, r, userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}
	sendJSONSuccess(w, r, "User logged in")
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
		domain.FromContext(r.Context()).Error("Failed to logout", zap.Error(err))
		return
	}

	_, err = api.authService.Logout(r.Context(), &pb.SessionCookieRequest{SessionCookie: session.Value})
	if err != nil {
		err = domain.FromGrpcError(err)
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
	sendJSONSuccess(w, r, "User logged out")
}

// Register регистрирует нового пользователя.
// @Summary Регистрация пользователя
// @Description Создает нового пользователя с профилем и устанавливает сессию
// @Tags auth
// @Accept json
// @Produce json
// @Param registerRequest body domain.RegisterRequest true "Данные для регистрации"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Failure 403 {object} JSONResponse
// @Router /auth/register [post]
func (api *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	req, err := DecodeJSONBody[domain.RegisterRequest](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	resp, err := api.authService.Register(r.Context(), generated.RegisterRequestToProto(req))
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	userID := resp.UserId

	err = api.AddSession(w, r, userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "User created, registration complete")
}
