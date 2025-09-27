package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"project/handlers"
	"project/store"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const sessionID = "session_id"

func TestRegister_OK(t *testing.T) {
	api := handlers.NewAuthHandler()
	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Register(w, registerReq)

	var res handlers.SuccessResponse
	err := json.NewDecoder(w.Body).Decode(&res)
	assert.NoError(t, err)

	assert.Equal(t, res.Message, "User created")
	assert.Equal(t, res.Code, http.StatusOK)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionID {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	IsLoggedInReq := httptest.NewRequest(http.MethodGet, "/api/auth/isloggedin", nil)
	IsLoggedInReq.AddCookie(sessionCookie)

	w = httptest.NewRecorder()
	api.IsLoggedInHandler(w, IsLoggedInReq)

	var loggedRes handlers.IsLoggedInResponse
	err = json.NewDecoder(w.Body).Decode(&loggedRes)
	assert.NoError(t, err)
	assert.Equal(t, loggedRes.IsLoggedIn, true)
}

func TestRegister_Fail_InvalidJSON(t *testing.T) {
	api := handlers.NewAuthHandler()
	// в body лишняя запятая
	body := `{
        "username": "misha",
        "email": "misha@email.ru",
        "confirm_password": "qwerty123",
        "password": "qwerty123", 
    }`

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	registerReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Register(w, registerReq)

	var res handlers.SuccessResponse
	err := json.NewDecoder(w.Body).Decode(&res)
	assert.NoError(t, err)

	assert.Equal(t, res.Message, "Invalid JSON")
	assert.Equal(t, res.Code, http.StatusBadRequest)

}
func TestRegister_Fail_InvalidData(t *testing.T) {
	cases := []struct {
		name string
		body handlers.RegisterRequest
	}{
		{"Invalid email",
			handlers.RegisterRequest{
				Username:        "misha",
				Email:           "misha@email......ru",
				ConfirmPassword: "qwerty123",
				Password:        "qwerty123",
			}},
		{"Password too short",
			handlers.RegisterRequest{
				Username:        "misha",
				Email:           "misha@email.ru",
				ConfirmPassword: "123",
				Password:        "123",
			}},
		{"Password too long",
			handlers.RegisterRequest{
				Username:        "misha",
				Email:           "misha@email.ru",
				ConfirmPassword: "12345612343123124123123123123123123123123",
				Password:        "12345612343123124123123123123123123123123",
			}},
		{"Empty username",
			handlers.RegisterRequest{
				Username:        "",
				Email:           "misha@email.ru",
				ConfirmPassword: "123456",
				Password:        "123456",
			}},
		{"Empty password",
			handlers.RegisterRequest{
				Username:        "misha",
				Email:           "misha@email.ru",
				ConfirmPassword: "",
				Password:        "",
			}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {

			api := handlers.NewAuthHandler()
			bodyJSON, _ := json.Marshal(test.body)

			registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
			registerReq.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			api.Register(w, registerReq)

			var res handlers.SuccessResponse
			err := json.NewDecoder(w.Body).Decode(&res)
			assert.NoError(t, err)

			assert.Equal(t, res.Message, "Invalid data")
			assert.Equal(t, res.Code, http.StatusBadRequest)

		})
	}
}
func TestRegister_Fail_PasswordsFieldsDontMatch(t *testing.T) {
	api := handlers.NewAuthHandler()
	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "123456",
		Password:        "qwerty",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Register(w, registerReq)

	var res handlers.SuccessResponse
	err := json.NewDecoder(w.Body).Decode(&res)
	assert.NoError(t, err)

	assert.Equal(t, res.Message, "Password field doesn't match")
	assert.Equal(t, res.Code, http.StatusBadRequest)

}
func TestRegister_UserAlreadyExist(t *testing.T) {
	api := handlers.NewAuthHandler()
	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq1 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	api.Register(w1, registerReq1)

	var res1 handlers.SuccessResponse
	err := json.NewDecoder(w1.Body).Decode(&res1)
	assert.NoError(t, err)

	assert.Equal(t, res1.Message, "User created")
	assert.Equal(t, res1.Code, http.StatusOK)

	registerReq2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	api.Register(w2, registerReq2)

	var res handlers.SuccessResponse
	err = json.NewDecoder(w2.Body).Decode(&res)
	assert.NoError(t, err)

	assert.Equal(t, res.Message, "User already exist")
	assert.Equal(t, res.Code, http.StatusBadRequest)

}

func TestLogout_OK(t *testing.T) {
	api := handlers.NewAuthHandler()
	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	api.Register(w1, registerReq)

	var res1 handlers.SuccessResponse
	err := json.NewDecoder(w1.Body).Decode(&res1)
	assert.NoError(t, err)

	assert.Equal(t, res1.Message, "User created")
	assert.Equal(t, res1.Code, http.StatusOK)

	var sessionCookie *http.Cookie
	for _, c := range w1.Result().Cookies() {
		if c.Name == sessionID {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutReq.AddCookie(sessionCookie)

	w2 := httptest.NewRecorder()
	api.Logout(w2, logoutReq)

	var expiredSessionCookie *http.Cookie
	for _, c := range w2.Result().Cookies() {
		if c.Name == sessionID {

			expiredSessionCookie = c
			break
		}
	}
	assert.Less(t, expiredSessionCookie.Expires, time.Now())
	var res2 handlers.SuccessResponse
	err = json.NewDecoder(w2.Body).Decode(&res2)
	assert.NoError(t, err)

	assert.Equal(t, res2.Message, "User logged out")
	assert.Equal(t, res2.Code, http.StatusOK)

	IsLoggedInReq := httptest.NewRequest(http.MethodGet, "/api/auth/isloggedin", nil)
	IsLoggedInReq.AddCookie(expiredSessionCookie)

	w3 := httptest.NewRecorder()
	api.IsLoggedInHandler(w3, IsLoggedInReq)

	var loggedRes handlers.IsLoggedInResponse
	err = json.NewDecoder(w3.Body).Decode(&loggedRes)
	assert.NoError(t, err)
	assert.Equal(t, loggedRes.IsLoggedIn, false)
}

func TestLogin_OK(t *testing.T) {
	api := handlers.NewAuthHandler()
	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	api.Register(w1, registerReq)

	var res1 handlers.SuccessResponse
	err := json.NewDecoder(w1.Body).Decode(&res1)
	assert.NoError(t, err)

	assert.Equal(t, res1.Message, "User created")
	assert.Equal(t, res1.Code, http.StatusOK)

	var sessionCookie *http.Cookie
	for _, c := range w1.Result().Cookies() {
		if c.Name == sessionID {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	bodyLogin := handlers.LoginRequest{
		Username: "misha",
		Password: "qwerty123",
	}
	bodyLoginJSON, _ := json.Marshal(bodyLogin)

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyLoginJSON))
	logoutReq.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	api.Login(w2, logoutReq)

	var loginSessionCookie *http.Cookie
	for _, c := range w2.Result().Cookies() {
		if c.Name == sessionID {

			loginSessionCookie = c
			break
		}
	}
	var res2 handlers.SuccessResponse
	err = json.NewDecoder(w2.Body).Decode(&res2)
	assert.NoError(t, err)

	assert.Equal(t, res2.Message, "User logged in")
	assert.Equal(t, res2.Code, http.StatusOK)

	IsLoginReq := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	IsLoginReq.AddCookie(loginSessionCookie)

	w3 := httptest.NewRecorder()
	api.IsLoggedInHandler(w3, IsLoginReq)

	var loggedRes handlers.IsLoggedInResponse
	err = json.NewDecoder(w3.Body).Decode(&loggedRes)
	assert.NoError(t, err)
	assert.Equal(t, loggedRes.IsLoggedIn, true)
}

func TestLogin_Fail_InvalidJSON(t *testing.T) {
	api := handlers.NewAuthHandler()
	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	api.Register(w1, registerReq)

	var res1 handlers.SuccessResponse
	err := json.NewDecoder(w1.Body).Decode(&res1)
	assert.NoError(t, err)

	assert.Equal(t, res1.Message, "User created")
	assert.Equal(t, res1.Code, http.StatusOK)

	var sessionCookie *http.Cookie
	for _, c := range w1.Result().Cookies() {
		if c.Name == sessionID {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	loginBody := `{
        "username": "misha",
        "email": "misha@email.ru",
        "confirm_password": "qwerty123",
        "password": "qwerty123", 
    }`

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginBody))
	logoutReq.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	api.Login(w2, logoutReq)

	var res2 handlers.SuccessResponse
	err = json.NewDecoder(w2.Body).Decode(&res2)
	assert.NoError(t, err)

	assert.Equal(t, res2.Message, "Invalid JSON")
	assert.Equal(t, res2.Code, http.StatusBadRequest)
}
func TestLogin_Fail_UserDoesnotExist(t *testing.T) {
	api := handlers.NewAuthHandler()
	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	api.Register(w1, registerReq)

	var res1 handlers.SuccessResponse
	err := json.NewDecoder(w1.Body).Decode(&res1)
	assert.NoError(t, err)

	assert.Equal(t, res1.Message, "User created")
	assert.Equal(t, res1.Code, http.StatusOK)

	var sessionCookie *http.Cookie
	for _, c := range w1.Result().Cookies() {
		if c.Name == sessionID {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	bodyLogin := handlers.LoginRequest{
		Username: "misha_l",
		Password: "",
	}
	bodyLoginJSON, _ := json.Marshal(bodyLogin)

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyLoginJSON))
	logoutReq.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	api.Login(w2, logoutReq)

	var res2 handlers.SuccessResponse
	err = json.NewDecoder(w2.Body).Decode(&res2)
	assert.NoError(t, err)

	assert.Equal(t, res2.Message, "User doesn't exist")
	assert.Equal(t, res2.Code, http.StatusBadRequest)
}

func TestLogin_Fail_IncorrectPassword(t *testing.T) {
	api := handlers.NewAuthHandler()
	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	api.Register(w1, registerReq)

	var res1 handlers.SuccessResponse
	err := json.NewDecoder(w1.Body).Decode(&res1)
	assert.NoError(t, err)

	assert.Equal(t, res1.Message, "User created")
	assert.Equal(t, res1.Code, http.StatusOK)

	var sessionCookie *http.Cookie
	for _, c := range w1.Result().Cookies() {
		if c.Name == sessionID {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	bodyLogin := handlers.LoginRequest{
		Username: "misha",
		Password: "123456",
	}
	bodyLoginJSON, _ := json.Marshal(bodyLogin)

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyLoginJSON))
	logoutReq.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	api.Login(w2, logoutReq)

	var res2 handlers.SuccessResponse
	err = json.NewDecoder(w2.Body).Decode(&res2)
	assert.NoError(t, err)

	assert.Equal(t, res2.Message, "Incorrect password")
	assert.Equal(t, res2.Code, http.StatusBadRequest)
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

	r := NewMuxRouter()

	ts := httptest.NewServer(r)
	defer ts.Close()

	client := &http.Client{}

	body := handlers.RegisterRequest{
		Username:        "misha",
		Email:           "misha@email.ru",
		ConfirmPassword: "qwerty123",
		Password:        "qwerty123",
	}
	bodyJSON, _ := json.Marshal(body)

	registerReq, err := http.NewRequest(http.MethodPost, ts.URL+"/api/auth/register", bytes.NewReader(bodyJSON))

	if err != nil {
		t.Fatal(err)
	}

	registerReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(registerReq)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	var res handlers.SuccessResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	assert.NoError(t, err)

	assert.Equal(t, res.Message, "User created")
	assert.Equal(t, res.Code, http.StatusOK)

	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == sessionID {
			sessionCookie = c
			break
		}
	}

	assert.NotNil(t, sessionCookie)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			forbiddenReq, err := http.NewRequest(http.MethodPost, ts.URL+test.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			forbiddenReq.AddCookie(sessionCookie)

			forbiddenResp, err := client.Do(forbiddenReq)
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				err := forbiddenResp.Body.Close()
				assert.NoError(t, err)
			}()

			assert.Equal(t, forbiddenResp.StatusCode, test.expectedStatus)

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
		{"Test register without auth", "POST", "/api/auth/register", http.StatusBadRequest},
		{"Test login without auth", "POST", "/api/auth/login", http.StatusBadRequest},
		{"Test logout without auth", "POST", "/api/auth/logout", http.StatusForbidden},
		{"Test isloggedin without auth", "GET", "/api/auth/isloggedin", http.StatusOK},
	}

	r := NewMuxRouter()

	ts := httptest.NewServer(r)
	defer ts.Close()

	client := &http.Client{}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			forbiddenReq, err := http.NewRequest(test.method, ts.URL+test.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			forbiddenResp, err := client.Do(forbiddenReq)
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				err := forbiddenResp.Body.Close()
				assert.NoError(t, err)
			}()

			assert.Equal(t, forbiddenResp.StatusCode, test.expectedStatus)
		})
	}
}

var forkPostsForTests = []store.Post{
	{ID: 1, Text: "Пост 1", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
	{ID: 2, Text: "Пост 2", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
	{ID: 3, Text: "Пост 3", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
	{ID: 4, Text: "Пост 4", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
	{ID: 5, Text: "Пост 5", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
	{ID: 6, Text: "Пост 6", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
	{ID: 7, Text: "Пост 7", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
	{ID: 8, Text: "Пост 8", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
}

func TestPostsPaginate_OK(t *testing.T) {
	api := handlers.NewPostsHandler(forkPostsForTests)
	cases := []struct {
		name          string
		limit         int
		page          int
		expectedPosts []store.Post
	}{
		{"Test last page", 5, 2, []store.Post{
			{ID: 6, Text: "Пост 6", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
			{ID: 7, Text: "Пост 7", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
			{ID: 8, Text: "Пост 8", LikeCount: 12, ImagePath: "/static/images/123.jpg"},
		}},
		{"Test empty page", 1000, 2, []store.Post{}},
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

			var res handlers.PostsResponse
			err := json.NewDecoder(w.Body).Decode(&res)
			assert.NoError(t, err)

			assert.Equal(t, res.Posts, test.expectedPosts)
		})
	}
}

func TestPostsPaginate_Fail_InvalidData(t *testing.T) {
	api := handlers.NewPostsHandler(forkPostsForTests)
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
	api := handlers.NewPostsHandler(forkPostsForTests)

	q := url.Values{}
	q.Add("limit", "1.1")
	q.Add("page", "////sdfs")
	fullURL := fmt.Sprintf("%s?%s", "/api/posts/", q.Encode())
	registerReq := httptest.NewRequest(http.MethodGet, fullURL, nil)

	w := httptest.NewRecorder()
	api.PostsPaginate(w, registerReq)

	assert.Equal(t, w.Result().StatusCode, http.StatusBadRequest)

}
