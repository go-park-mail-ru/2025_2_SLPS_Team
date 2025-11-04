package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"project/domain"
	"project/internal/service/mocks"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func JSONReader(t *testing.T, v any) io.Reader {
	b, err := json.Marshal(v)
	assert.NoError(t, err)
	return bytes.NewReader(b)
}

func newJSONRequest(method, url string, body any, t *testing.T) (*http.Request, *httptest.ResponseRecorder) {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, url, JSONReader(t, body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	return r, httptest.NewRecorder()
}

func getCookies(result *http.Response) (session, csrf *http.Cookie) {
	for _, c := range result.Cookies() {
		if c.Name == "session_id" {
			session = c
		}
		if c.Name == "CSRF_token" {
			csrf = c
		}
	}
	return
}

func decodeResponse[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	var res T
	err := json.NewDecoder(w.Body).Decode(&res)
	assert.NoError(t, err)
	return res
}

func TestAuthHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockAuthService := mocks.NewMockAuthService(ctrl)
	handler := &AuthHandler{authService: mockAuthService}

	t.Run("Success", func(t *testing.T) {
		user := domain.User{Email: "a@b.com", Password: "pass"}
		tokens := domain.SIDAndSCRFToken{SID: "session123", CSRFToken: "CSRF123"}
		mockAuthService.EXPECT().Login(gomock.Any(), user).Return(0, nil)
		mockAuthService.EXPECT().AddSession(gomock.Any(), 0).Return(&tokens, nil)

		r, w := newJSONRequest(http.MethodPost, "/api/auth/login", user, t)
		handler.Login(w, r)

		res := decodeResponse[JSONResponse](t, w)
		session, csrf := getCookies(w.Result())

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, "User logged in", res.Message)
		assert.Equal(t, http.StatusOK, res.Code)
		assert.NotNil(t, session)
		assert.NotNil(t, csrf)
		assert.Equal(t, tokens.SID, session.Value)
		assert.Equal(t, tokens.CSRFToken, csrf.Value)
	})

	t.Run("ServiceLogin send err", func(t *testing.T) {
		user := domain.User{Email: "a@b.com", Password: "pass"}
		mockAuthService.EXPECT().Login(gomock.Any(), user).Return(0, domain.ErrInvalidInput)

		r, w := newJSONRequest(http.MethodPost, "/api/auth/login", user, t)
		handler.Login(w, r)

		res := decodeResponse[JSONResponse](t, w)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
		assert.Equal(t, domain.InvalidData, res.Message)
	})

	t.Run("HandlerAddSession send err", func(t *testing.T) {
		user := domain.User{Email: "a@b.com", Password: "pass"}
		mockAuthService.EXPECT().Login(gomock.Any(), user).Return(0, nil)
		mockAuthService.EXPECT().AddSession(gomock.Any(), 0).Return(nil, domain.ErrDB)

		r, w := newJSONRequest(http.MethodPost, "/api/auth/login", user, t)
		handler.Login(w, r)

		session, csrf := getCookies(w.Result())
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
		assert.Nil(t, session)
		assert.Nil(t, csrf)
	})
}

func TestAuthHandler_IsLoggedInHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockAuthService := mocks.NewMockAuthService(ctrl)
	handler := &AuthHandler{authService: mockAuthService}

	t.Run("Success", func(t *testing.T) {
		session := &domain.Session{UserID: 1}
		cookie := &http.Cookie{Name: "session_id", Value: "session123"}
		mockAuthService.EXPECT().IsLoggedIn(gomock.Any(), cookie).Return(session, nil)

		r, w := newJSONRequest(http.MethodGet, "/auth/isloggedin", nil, t)
		r.AddCookie(cookie)
		handler.IsLoggedInHandler(w, r)

		res := decodeResponse[IsLoggedInResponse](t, w)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, session.UserID, res.UserID)
	})

	t.Run("Cookie missing", func(t *testing.T) {
		r, w := newJSONRequest(http.MethodGet, "/auth/isloggedin", nil, t)
		handler.IsLoggedInHandler(w, r)
		assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	})

	t.Run("Service returns error", func(t *testing.T) {
		cookie := &http.Cookie{Name: "session_id", Value: "session123"}
		mockAuthService.EXPECT().IsLoggedIn(gomock.Any(), cookie).Return(nil, domain.ErrNotFound)

		r, w := newJSONRequest(http.MethodGet, "/auth/isloggedin", nil, t)
		r.AddCookie(cookie)
		handler.IsLoggedInHandler(w, r)
		assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockAuthService := mocks.NewMockAuthService(ctrl)
	handler := &AuthHandler{authService: mockAuthService}

	t.Run("Success", func(t *testing.T) {
		cookie := &http.Cookie{Name: "session_id", Value: "session123"}
		csrf := &http.Cookie{Name: "CSRF_token", Value: "csrf123"}
		mockAuthService.EXPECT().Logout(gomock.Any(), cookie).Return(nil)

		r, w := newJSONRequest(http.MethodPost, "/auth/logout", nil, t)
		r.AddCookie(cookie)
		r.AddCookie(csrf)
		handler.Logout(w, r)

		session, csrfC := getCookies(w.Result())
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.NotNil(t, session)
		assert.NotNil(t, csrfC)
		assert.True(t, session.Expires.Before(time.Now()))
		assert.True(t, csrfC.Expires.Before(time.Now()))
	})

	t.Run("Cookie missing", func(t *testing.T) {
		r, w := newJSONRequest(http.MethodPost, "/auth/logout", nil, t)
		handler.Logout(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Service returns error", func(t *testing.T) {
		cookie := &http.Cookie{Name: "session_id", Value: "session123"}
		mockAuthService.EXPECT().Logout(gomock.Any(), cookie).Return(domain.ErrDB)

		r, w := newJSONRequest(http.MethodPost, "/auth/logout", nil, t)
		r.AddCookie(cookie)
		handler.Logout(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

func TestAuthHandler_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockAuthService := mocks.NewMockAuthService(ctrl)
	handler := &AuthHandler{authService: mockAuthService}

	t.Run("Success", func(t *testing.T) {
		req := domain.RegisterRequest{
			FirstName:       "misha",
			LastName:        "beztoksa",
			Email:           "misha@email.ru",
			Password:        "qwerty123",
			ConfirmPassword: "qwerty123",
			Gender:          "man",
		}
		tokens := domain.SIDAndSCRFToken{SID: "session123", CSRFToken: "csrf123"}

		mockAuthService.EXPECT().Register(gomock.Any(), req).Return(1, nil)
		mockAuthService.EXPECT().AddSession(gomock.Any(), 1).Return(&tokens, nil)

		r, w := newJSONRequest(http.MethodPost, "/auth/register", req, t)
		handler.Register(w, r)

		res := decodeResponse[JSONResponse](t, w)
		session, csrf := getCookies(w.Result())

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, "User created", res.Message)
		assert.Equal(t, tokens.SID, session.Value)
		assert.Equal(t, tokens.CSRFToken, csrf.Value)
	})

	t.Run("Service register error", func(t *testing.T) {
		req := domain.RegisterRequest{
			FirstName:       "misha",
			LastName:        "beztoksa",
			Password:        "qwerty123",
			ConfirmPassword: "qwerty123",
			Gender:          "man",
		}
		mockAuthService.EXPECT().Register(gomock.Any(), req).Return(0, domain.ErrInvalidInput)

		r, w := newJSONRequest(http.MethodPost, "/auth/register", req, t)
		handler.Register(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("AddSession error", func(t *testing.T) {
		req := domain.RegisterRequest{
			FirstName:       "misha",
			LastName:        "beztoksa",
			Email:           "misha@email.ru",
			Password:        "qwerty123",
			ConfirmPassword: "qwerty123",
			Gender:          "man",
		}
		mockAuthService.EXPECT().Register(gomock.Any(), req).Return(1, nil)
		mockAuthService.EXPECT().AddSession(gomock.Any(), 1).Return(nil, domain.ErrDB)

		r, w := newJSONRequest(http.MethodPost, "/auth/register", req, t)
		handler.Register(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}
