package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"project/auth"
	"project/store"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/schema"
	"golang.org/x/crypto/bcrypt"
)

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(SuccessResponse{
		Success: false,
		Message: message,
		Code:    statusCode,
	}); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func sendJSONSuccess(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: message,
		Code:    statusCode,
	}); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

type PostsHandler struct {
	postsStore *store.PostsStore
}

func NewPostsHandler(Posts []store.Post) *PostsHandler {
	return &PostsHandler{
		postsStore: store.NewPostStore(Posts),
	}
}

type AuthHandler struct {
	sessionStore *auth.SessionStore
	userStore    *auth.UserStore
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		sessionStore: auth.NewSessionStore(),
		userStore:    auth.NewUserStore(),
	}
}

func (api *AuthHandler) IsLoggedIn(r *http.Request) bool {
	authorized := false
	session, err := r.Cookie("session_id")
	if err == nil && session != nil {
		_, authorized = api.sessionStore.GetSessionByID(session.Value)
	}
	return authorized
}

type IsLoggedInResponse struct {
	IsLoggedIn bool
}

func (api *AuthHandler) IsLoggedInHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var res = IsLoggedInResponse{
		IsLoggedIn: api.IsLoggedIn(r),
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

type LoginRequest struct {
	Username string `json:"username" valid:"required"`
	Password string `json:"password" valid:"required, stringlength(6|20)"`
}

func (api *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	user, ok := api.userStore.GetUserByUsername(req.Username)
	if !ok {
		sendJSONError(w, "User doesn't exist", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		sendJSONError(w, "Incorrect password", http.StatusBadRequest)
		return
	}

	SID, err := api.sessionStore.AddSession(user.ID)
	if err != nil {
		sendJSONError(w, "Server error", http.StatusInternalServerError)
		return
	}

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
	Password        string `json:"password" valid:"required, stringlength(6|20)"`
	ConfirmPassword string `json:"confirm_password" valid:"required, stringlength(6|20)"`
}

func (api *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ok, err := govalidator.ValidateStruct(req)
	if !ok {
		sendJSONError(w, "Invalid data", http.StatusBadRequest)
		return
	}

	_, ok = api.userStore.GetUserByUsername(req.Username)
	if ok {
		sendJSONError(w, "User already exist", http.StatusBadRequest)
		return
	}

	if req.Password != req.ConfirmPassword {
		sendJSONError(w, "Password field doesn't match", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	username := api.userStore.AddUser(req.Username, req.Email, string(hashedPassword))
	user, _ := api.userStore.GetUserByUsername(username)

	SID, err := api.sessionStore.AddSession(user.ID)
	if err != nil {
		sendJSONError(w, "Internal server error", http.StatusInternalServerError)
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

type PostsRequest struct {
	Page  int `schema:"page"`
	Limit int `schema:"limit"`
}
type PostsResponse struct {
	Posts      []store.Post `json:"posts"`
	PagesCount int          `json:"pages"`
}

func (api *PostsHandler) PostsPaginate(w http.ResponseWriter, r *http.Request) {
	var req PostsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONError(w, "Invalid params", http.StatusBadRequest)
		return
	}

	if req.Page <= 0 || req.Limit <= 0 {
		sendJSONError(w, "Invalid params", http.StatusBadRequest)
		return
	}

	paginatedPostList, pagesCount := api.postsStore.PostsPaginatedList(req.Page, req.Limit)

	res := PostsResponse{
		Posts:      paginatedPostList,
		PagesCount: pagesCount,
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func SPAHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

var ForbiddenPathsWithAuth = map[string]bool{
	"/api/auth/login":    true,
	"/api/auth/register": true,
}
var AllowedPathsWithOutAuth = map[string]bool{
	"/api/auth/login":      true,
	"/api/auth/register":   true,
	"/api/auth/isloggedin": true,
}

func (api *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if api.IsLoggedIn(r) {
			if ForbiddenPathsWithAuth[path] {
				sendJSONError(w, "Forbidden", http.StatusForbidden)
				return
			}
		} else {
			if !AllowedPathsWithOutAuth[path] {
				sendJSONError(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func StaticHandler(staticDir http.Dir, prefix string) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(r.URL.Path, prefix)
		cleanPath := filepath.Clean("/" + requestPath)
		cleanPath = strings.TrimPrefix(cleanPath, "/")
		fullPath := filepath.Join(string(staticDir), cleanPath)
		absStaticDir, err := filepath.Abs(string(staticDir))
		if err != nil {
			sendJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		absFilePath, err := filepath.Abs(fullPath)
		if err != nil || !strings.HasPrefix(absFilePath, absStaticDir) {
			sendJSONError(w, "Forbidden", http.StatusForbidden)
			return
		}

		info, err := os.Stat(absFilePath)
		if err != nil || info.IsDir() {
			sendJSONError(w, "Not found", http.StatusNotFound)
			return
		}

		fs := http.FileServer(staticDir)
		http.StripPrefix(prefix, fs).ServeHTTP(w, r)
	})
}
