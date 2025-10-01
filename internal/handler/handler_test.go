package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"project/repository"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

const sessionID = "session_id"

func FakeHttpAuth[T any](handler func(w http.ResponseWriter, r *http.Request), body io.Reader, cookie *http.Cookie, url string) (T, *http.Cookie, error) {
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	handler(w, req)

	var res T
	err := json.NewDecoder(w.Body).Decode(&res)

	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	if cookies != nil {
		for _, c := range cookies {
			if c.Name == sessionID {
				sessionCookie = c
				break
			}
		}
	}

	return res, sessionCookie, err
}
func TestRegister_OK(t *testing.T) {
	api := NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))

	body := RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
		Age:             12,
		Gender:          "man",
	}
	bodyJSON, err := json.Marshal(body)
	assert.NoError(t, err)

	res, SID, err := FakeHttpAuth[SuccessResponse](api.Register, bytes.NewReader(bodyJSON), nil, "/api/auth/register")
	assert.NoError(t, err)
	assert.Equal(t, res.Message, "User created")
	assert.Equal(t, res.Code, http.StatusOK)

	assert.NotNil(t, SID)
	assert.NotEmpty(t, SID)
}

func TestRegister_Fail_InvalidJSON(t *testing.T) {
	api := NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))

	// в body лишняя запятая
	body := `{
        "username": "misha",
        "email": "misha@email.ru",
        "confirm_password": "qwerty123",
        "password": "qwerty123", 
    }`

	res, SID, err := FakeHttpAuth[SuccessResponse](api.Register, strings.NewReader(body), nil, "/api/auth/register")

	assert.NoError(t, err)

	assert.Equal(t, res.Message, "Invalid JSON")
	assert.Equal(t, res.Code, http.StatusBadRequest)
	assert.Nil(t, SID)

}
func TestRegister_Fail_InvalidData(t *testing.T) {
	cases := []struct {
		name string
		body RegisterRequest
	}{
		{"Invalid email",
			RegisterRequest{
				Username:        "misha",
				Email:           "misha@email......ru",
				ConfirmPassword: "qwerty123",
				Password:        "qwerty123",
			}},
		{"Password too short",
			RegisterRequest{
				Username:        "misha",
				Email:           "misha@email.ru",
				ConfirmPassword: "123",
				Password:        "123",
			}},
		{"Password too long",
			RegisterRequest{
				Username:        "misha",
				Email:           "misha@email.ru",
				ConfirmPassword: "12345612343123124123123123123123123123123",
				Password:        "12345612343123124123123123123123123123123",
			}},
		{"Empty username",
			RegisterRequest{
				Username:        "",
				Email:           "misha@email.ru",
				ConfirmPassword: "123456",
				Password:        "123456",
			}},
		{"Empty password",
			RegisterRequest{
				Username:        "misha",
				Email:           "misha@email.ru",
				ConfirmPassword: "",
				Password:        "",
			}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			api := NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))

			bodyJSON, err := json.Marshal(test.body)
			assert.NoError(t, err)

			res, SID, err := FakeHttpAuth[SuccessResponse](api.Register, bytes.NewReader(bodyJSON), nil, "/api/auth/register")

			assert.NoError(t, err)

			assert.Equal(t, res.Message, "Invalid data")
			assert.Equal(t, res.Code, http.StatusBadRequest)
			assert.Nil(t, SID)

		})
	}
}
func TestRegister_Fail_PasswordsFieldsDoesntMatch(t *testing.T) {
	api := NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))

	body := RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "123456",
		Password:        "qwerty",
	}
	bodyJSON, err := json.Marshal(body)
	assert.NoError(t, err)

	res, SID, err := FakeHttpAuth[SuccessResponse](api.Register, bytes.NewReader(bodyJSON), nil, "/api/auth/register")

	assert.NoError(t, err)

	assert.Equal(t, res.Message, "Password field doesn't match")
	assert.Equal(t, res.Code, http.StatusBadRequest)
	assert.Nil(t, SID)

}
func TestRegister_UserAlreadyExist(t *testing.T) {
	users := map[string]repository.User{
		"misha@email.ru": {
			Username:       "misha",
			Email:          "misha@email.ru",
			HashedPassword: "qwerty123",
			Age:            0,
			Gender:         "",
		},
	}

	api := NewAuthHandler(users, make(map[string]repository.Session))

	body := RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, err := json.Marshal(body)
	assert.NoError(t, err)

	res, SID, err := FakeHttpAuth[SuccessResponse](api.Register, bytes.NewReader(bodyJSON), nil, "/api/auth/register")

	assert.NoError(t, err)

	assert.Equal(t, res.Message, "User already exist")
	assert.Equal(t, res.Code, http.StatusBadRequest)
	assert.Nil(t, SID)

}

func TestLogout_OK(t *testing.T) {
	SID := "dfajdfakdjsfklasd"
	sessions := map[string]repository.Session{
		SID: {
			ID:     SID,
			UserId: 1,
		},
	}

	api := NewAuthHandler(make(map[string]repository.User), sessions)

	cookie := &http.Cookie{
		Name:     sessionID,
		Value:    SID,
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: true,
	}
	res, SessionC, err := FakeHttpAuth[SuccessResponse](api.Logout, nil, cookie, "/api/auth/logout")

	assert.NotNil(t, SessionC)
	assert.Less(t, SessionC.Expires, time.Now())
	assert.NoError(t, err)

	assert.Equal(t, res.Message, "User logged out")
	assert.Equal(t, res.Code, http.StatusOK)
}

func TestIsloggedin_True(t *testing.T) {
	SID := "dfajdfakdjsfklasd"
	sessions := map[string]repository.Session{
		SID: {
			ID:     SID,
			UserId: 1,
		},
	}

	api := NewAuthHandler(make(map[string]repository.User), sessions)

	cookie := &http.Cookie{
		Name:     sessionID,
		Value:    SID,
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: true,
	}
	res, _, err := FakeHttpAuth[IsLoggedInResponse](api.IsLoggedInHandler, nil, cookie, "/api/auth/isloggedin")

	assert.NoError(t, err)
	assert.Equal(t, res.IsLoggedIn, true)
}

func TestIsloggedinFalse(t *testing.T) {
	api := NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))

	res, SessionC, err := FakeHttpAuth[IsLoggedInResponse](api.IsLoggedInHandler, nil, nil, "/api/auth/isloggedin")

	assert.Nil(t, SessionC)
	assert.NoError(t, err)
	assert.Equal(t, res.IsLoggedIn, false)
}
func TestLogin_OK(t *testing.T) {
	password := "qwerty123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	users := map[string]repository.User{
		"misha@email.ru": {
			Username:       "misha",
			Email:          "misha@email.ru",
			HashedPassword: string(hashedPassword),
			Age:            0,
			Gender:         "",
		},
	}

	api := NewAuthHandler(users, make(map[string]repository.Session))

	body := LoginRequest{
		Email:    "misha@email.ru",
		Password: password,
	}
	bodyJSON, err := json.Marshal(body)
	assert.NoError(t, err)

	res, SID, err := FakeHttpAuth[SuccessResponse](api.Login, bytes.NewReader(bodyJSON), nil, "/api/auth/login")

	assert.NoError(t, err)

	assert.Equal(t, res.Message, "User logged in")
	assert.Equal(t, res.Code, http.StatusOK)
	assert.NotNil(t, SID)
	assert.NotEmpty(t, SID)

}

func TestLogin_Fail_InvalidJSON(t *testing.T) {
	api := NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))

	body := `{
        "username": "misha",
        "email": "misha@email.ru",
        "confirm_password": "qwerty123",
        "password": "qwerty123", 
    }`

	res, SID, err := FakeHttpAuth[SuccessResponse](api.Login, strings.NewReader(body), nil, "/api/auth/login")

	assert.NoError(t, err)

	assert.Equal(t, res.Message, "Invalid JSON")
	assert.Equal(t, res.Code, http.StatusBadRequest)
	assert.Nil(t, SID)
}

func TestLogin_Fail_UserDoesnotExist(t *testing.T) {
	api := NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))

	body := LoginRequest{
		Email:    "misha_l",
		Password: "",
	}
	bodyJSON, err := json.Marshal(body)
	assert.NoError(t, err)

	res, SID, err := FakeHttpAuth[SuccessResponse](api.Login, bytes.NewReader(bodyJSON), nil, "/api/auth/login")

	assert.NoError(t, err)

	assert.Equal(t, res.Message, "User doesn't exist")
	assert.Equal(t, res.Code, http.StatusBadRequest)
	assert.Nil(t, SID)
}

func TestLogin_Fail_IncorrectPassword(t *testing.T) {
	password := "qwerty123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	users := map[string]repository.User{
		"misha@email.ru": {
			Username:       "misha",
			Email:          "misha@email.ru",
			HashedPassword: string(hashedPassword),
			Age:            0,
			Gender:         "",
		},
	}

	api := NewAuthHandler(users, make(map[string]repository.Session))

	body := LoginRequest{
		Email:    "misha@email.ru",
		Password: "",
	}

	bodyJSON, err := json.Marshal(body)
	assert.NoError(t, err)

	res, SID, err := FakeHttpAuth[SuccessResponse](api.Login, bytes.NewReader(bodyJSON), nil, "/api/auth/login")

	assert.NoError(t, err)

	assert.Equal(t, res.Message, "Incorrect password")
	assert.Equal(t, res.Code, http.StatusBadRequest)
	assert.Nil(t, SID)
}

func TestMiddleWare_WithAuth(t *testing.T) {
	cases := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{"Test register with auth", "/api/auth/register", http.StatusForbidden},
		{"Test login with auth", "/api/auth/login", http.StatusForbidden},
		{"Test logout with auth", "/api/auth/logout", http.StatusOK},
	}
	SID := "dfajdfakdjsfklasd"
	sessions := map[string]repository.Session{
		SID: {
			ID:     SID,
			UserId: 1,
		},
	}

	api := NewAuthHandler(make(map[string]repository.User), sessions)

	cookie := &http.Cookie{
		Name:     sessionID,
		Value:    SID,
		Expires:  time.Now().Add(10 * time.Hour),
		HttpOnly: true,
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sendJSONSuccess(w, "ok", http.StatusOK)
			})
			wrapped := api.AuthMiddleware(next)
			req := httptest.NewRequest(http.MethodPost, test.url, nil)
			req.AddCookie(cookie)

			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			var res SuccessResponse
			err := json.NewDecoder(w.Body).Decode(&res)
			assert.NoError(t, err)

			assert.Equal(t, res.Code, test.expectedStatus)

		})
	}
}

func TestMiddleWare_WithoutAuth(t *testing.T) {
	cases := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{"Test register without auth", "POST", "/api/auth/register", http.StatusOK},
		{"Test login without auth", "POST", "/api/auth/login", http.StatusOK},
		{"Test logout without auth", "POST", "/api/auth/logout", http.StatusForbidden},
		{"Test isloggedin without auth", "GET", "/api/auth/isloggedin", http.StatusOK},
	}
	api := NewAuthHandler(make(map[string]repository.User), make(map[string]repository.Session))

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sendJSONSuccess(w, "ok", http.StatusOK)
			})

			wrapped := api.AuthMiddleware(next)
			req := httptest.NewRequest(http.MethodPost, test.url, nil)

			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			var res SuccessResponse
			err := json.NewDecoder(w.Body).Decode(&res)
			assert.NoError(t, err)

			assert.Equal(t, res.Code, test.expectedStatus)
		})
	}
}

var forkPostsForTests = []repository.Post{
	{ID: 1, Text: "Пост 1", LikeCount: 12, RepostsCount: 12, CommentCount: 12, GroupName: "Группа", CommunityAvatar: "/static/images/123.jpg", PhotosPath: []string{"/static/images/123.jpg", "/static/images/123.jpg"}},
	{ID: 2, Text: "Пост 1", LikeCount: 12, RepostsCount: 12, CommentCount: 12, GroupName: "Группа", CommunityAvatar: "/static/images/123.jpg", PhotosPath: []string{"/static/images/123.jpg", "/static/images/123.jpg"}},
	{ID: 3, Text: "Пост 1", LikeCount: 12, RepostsCount: 12, CommentCount: 12, GroupName: "Группа", CommunityAvatar: "/static/images/123.jpg", PhotosPath: []string{"/static/images/123.jpg", "/static/images/123.jpg"}},
	{ID: 4, Text: "Пост 1", LikeCount: 12, RepostsCount: 12, CommentCount: 12, GroupName: "Группа", CommunityAvatar: "/static/images/123.jpg", PhotosPath: []string{"/static/images/123.jpg", "/static/images/123.jpg"}},
	{ID: 5, Text: "Пост 1", LikeCount: 12, RepostsCount: 12, CommentCount: 12, GroupName: "Группа", CommunityAvatar: "/static/images/123.jpg", PhotosPath: []string{"/static/images/123.jpg", "/static/images/123.jpg"}},
	{ID: 6, Text: "Пост 1", LikeCount: 12, RepostsCount: 12, CommentCount: 12, GroupName: "Группа", CommunityAvatar: "/static/images/123.jpg", PhotosPath: []string{"/static/images/123.jpg", "/static/images/123.jpg"}},
	{ID: 7, Text: "Пост 1", LikeCount: 12, RepostsCount: 12, CommentCount: 12, GroupName: "Группа", CommunityAvatar: "/static/images/123.jpg", PhotosPath: []string{"/static/images/123.jpg", "/static/images/123.jpg"}},
	{ID: 8, Text: "Пост 1", LikeCount: 12, RepostsCount: 12, CommentCount: 12, GroupName: "Группа", CommunityAvatar: "/static/images/123.jpg", PhotosPath: []string{"/static/images/123.jpg", "/static/images/123.jpg"}},
}

func TestPostsPaginate_OK(t *testing.T) {
	api := NewPostsHandler(forkPostsForTests)
	cases := []struct {
		name          string
		limit         int
		page          int
		expectedPosts []repository.Post
	}{
		{"Test last page", 5, 2, forkPostsForTests[5:]},
		{"Test empty page", 1000, 2, []repository.Post{}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			q := url.Values{}
			q.Add("limit", fmt.Sprintf("%d", test.limit))
			q.Add("page", fmt.Sprintf("%d", test.page))
			fullURL := fmt.Sprintf("%s?%s", "/api/posts/", q.Encode())
			registerReq := httptest.NewRequest(http.MethodGet, fullURL, nil)

			w := httptest.NewRecorder()
			api.PostsPaginate(w, registerReq)

			var res PostsResponse
			err := json.NewDecoder(w.Body).Decode(&res)
			assert.NoError(t, err)

			assert.Equal(t, res.Posts, test.expectedPosts)
		})
	}
}

func TestPostsPaginate_Fail_InvalidData(t *testing.T) {
	api := NewPostsHandler(forkPostsForTests)
	cases := []struct {
		name               string
		limit              int
		page               int
		expectedStatusCode int
	}{
		{"Negative or zero params", -14, 0, http.StatusBadRequest},
		{"Negative or zero params", 21, -3, http.StatusBadRequest},
		{"Negative or zero params", 0, 1, http.StatusBadRequest},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			q := url.Values{}
			q.Add("limit", fmt.Sprintf("%d", test.limit))
			q.Add("page", fmt.Sprintf("%d", test.page))
			fullURL := fmt.Sprintf("%s?%s", "/api/posts/", q.Encode())
			registerReq := httptest.NewRequest(http.MethodGet, fullURL, nil)

			w := httptest.NewRecorder()
			api.PostsPaginate(w, registerReq)

			assert.Equal(t, w.Result().StatusCode, test.expectedStatusCode)
		})
	}
}
func TestPostsPaginate_Fail_InvalidParams(t *testing.T) {
	api := NewPostsHandler(forkPostsForTests)

	q := url.Values{}
	q.Add("limit", "1.1")
	q.Add("page", "////sdfs")
	fullURL := fmt.Sprintf("%s?%s", "/api/posts/", q.Encode())
	registerReq := httptest.NewRequest(http.MethodGet, fullURL, nil)

	w := httptest.NewRecorder()
	api.PostsPaginate(w, registerReq)

	assert.Equal(t, w.Result().StatusCode, http.StatusBadRequest)

}
