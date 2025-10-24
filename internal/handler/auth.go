package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"project/domain"
	"time"

	"github.com/asaskevich/govalidator"
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

func (api *AuthHandler) IsLoggedIn(r *http.Request) (int, error) {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		log.Println("Cookie session_id not found:", err)
		return 0, domain.ErrNotFound
	}

	log.Printf("Found session_id: %s\n", sessionCookie.Value)

	session, authorizedErr := api.sessionStore.GetSessionBySessionID(sessionCookie.Value)
	if authorizedErr != nil {
		log.Println("Session not found or error:", authorizedErr)
		return 0, authorizedErr
	}

	log.Printf("Session loaded: %+v\n", session)
	return session.UserID, nil
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
	UserID, err := api.IsLoggedIn(r)
	log.Println("IsLoggedInHandler called")
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			sendJSONSuccess(w, "Not found", http.StatusNotFound)
			return
		} else {
			sendJSONSuccess(w, "Server error", http.StatusInternalServerError)
			return
		}
	}

	res := IsLoggedInResponse{
		UserID: UserID,
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 31)

	cryptoReader := rand.Reader
	_, err := cryptoReader.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
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
		sendJSONSuccess(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	user, err := api.userStore.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			sendJSONSuccess(w, "User doesn't exist", http.StatusBadRequest)
			return
		} else {
			sendJSONSuccess(w, "Server error", http.StatusInternalServerError)
			return
		}

	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		sendJSONSuccess(w, "Incorrect password", http.StatusBadRequest)
		return
	}

	SID, err := generateSessionID()
	if err != nil {
		sendJSONSuccess(w, "Intrnal Server error", http.StatusInternalServerError)
		return
	}

	err = api.sessionStore.AddSession(user.ID, SID)
	if err != nil {
		sendJSONSuccess(w, "Intrnal Server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    SID,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	sendJSONSuccess(w, "User logged in", http.StatusOK)
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

	session, _ := r.Cookie("session_id")
	err := api.sessionStore.DeleteSession(session.Value)
	if err != nil {
		sendJSONSuccess(w, "Internal Server error", http.StatusInternalServerError)
		return
	}

	session.Expires = time.Now().AddDate(0, 0, -1)
	http.SetCookie(w, session)

	sendJSONSuccess(w, "User logged out", http.StatusOK)
}

type RegisterRequest struct {
	FirstName       string    `json:"firstName" valid:"required"`
	LastName        string    `json:"lastName" valid:"required"`
	Email           string    `json:"email" valid:"email, required"`
	Password        string    `json:"password" valid:"required, stringlength(5|20)"`
	ConfirmPassword string    `json:"confirmPassword" valid:"required, stringlength(5|20)"`
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
		sendJSONSuccess(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		sendJSONSuccess(w, "Invalid data", http.StatusBadRequest)
		return
	}

	_, err = api.userStore.GetUserByEmail(req.Email)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			sendJSONSuccess(w, "Internal Server error", http.StatusInternalServerError)
			return
		}
	} else {
		sendJSONSuccess(w, "User already exist", http.StatusBadRequest)
		return
	}

	if req.Password != req.ConfirmPassword {
		sendJSONSuccess(w, "Password field doesn't match", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
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
	userID, err := api.userStore.CreateUser(user, profile)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	SID, err := generateSessionID()
	if err != nil {
		sendJSONSuccess(w, "Internal Server error", http.StatusInternalServerError)
		return
	}

	err = api.sessionStore.AddSession(userID, SID)
	if err != nil {

		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    SID,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
	sendJSONSuccess(w, "User created", http.StatusOK)
}
