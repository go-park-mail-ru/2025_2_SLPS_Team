package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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

func (api *AuthHandler) IsLoggedIn(r *http.Request) (int, bool) {
	authorized := false
	sessionCookie, err := r.Cookie("session_id")
	var session domain.Session

	if err == nil && sessionCookie != nil {
		session, authorized = api.sessionStore.GetSessionBySessionID(sessionCookie.Value)
	}

	return session.UserID, authorized
}

type IsLoggedInResponse struct {
	IsLoggedIn bool `json:"isloggedin"`
}

func (api *AuthHandler) IsLoggedInHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, isloggedin := api.IsLoggedIn(r)
	var res = IsLoggedInResponse{
		IsLoggedIn: isloggedin,
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

func (api *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONSuccess(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	user, ok := api.userStore.GetUserByEmail(req.Email)
	if !ok {
		sendJSONSuccess(w, "User doesn't exist", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		sendJSONSuccess(w, "Incorrect password", http.StatusBadRequest)
		return
	}

	SID, err := generateSessionID()
	if err != nil {
		sendJSONSuccess(w, "Server error", http.StatusInternalServerError)
		return
	}

	api.sessionStore.AddSession(user.ID, SID)

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    SID,
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	sendJSONSuccess(w, "User logged in", http.StatusOK)
}

func (api *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {

	session, _ := r.Cookie("session_id")
	api.sessionStore.DeleteSession(session.Value)
	session.Expires = time.Now().AddDate(0, 0, -1)
	http.SetCookie(w, session)

	sendJSONSuccess(w, "User logged out", http.StatusOK)
}

type RegisterRequest struct {
	Username        string `json:"username" valid:"required"`
	Email           string `json:"email" valid:"email, required"`
	Password        string `json:"password" valid:"required, stringlength(5|20)"`
	ConfirmPassword string `json:"confirm_password" valid:"required, stringlength(5|20)"`
	Age             int    `json:"age" valid:"-"`
	Gender          string `json:"gender" valid:"-"`
}

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

	_, ok = api.userStore.GetUserByEmail(req.Email)
	if ok {
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

	username := api.userStore.AddUser(req.Username, req.Email, req.Gender, string(hashedPassword), req.Age)
	user, _ := api.userStore.GetUserByEmail(username)

	SID, err := generateSessionID()
	if err != nil {
		sendJSONSuccess(w, "Server error", http.StatusInternalServerError)
		return
	}

	api.sessionStore.AddSession(user.ID, SID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    SID,
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	log.Println(user)
	sendJSONSuccess(w, "User created", http.StatusOK)
}
