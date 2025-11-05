package service

import (
	"context"
	"errors"
	"net/http"
	"project/domain"
	"project/internal/repository/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func newAuthServiceMocks(t *testing.T) (*AuthService, *mocks.MockUserStore, *mocks.MockSessionStore, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	userStore := mocks.NewMockUserStore(ctrl)
	sessionStore := mocks.NewMockSessionStore(ctrl)
	svc := &AuthService{userStore: userStore, sessionStore: sessionStore}
	return svc, userStore, sessionStore, ctrl
}

func TestAuthService_IsLoggedIn(t *testing.T) {
	svc, _, sessionStore, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cookie := &http.Cookie{Name: "session_id", Value: "abc"}

	t.Run("Success", func(t *testing.T) {
		session := &domain.Session{UserID: 1}
		sessionStore.EXPECT().GetSessionBySessionID(ctx, cookie.Value).Return(session, nil)
		res, err := svc.IsLoggedIn(ctx, cookie)
		assert.NoError(t, err)
		assert.Equal(t, session, res)
	})

	t.Run("Not found", func(t *testing.T) {
		sessionStore.EXPECT().GetSessionBySessionID(ctx, cookie.Value).Return(nil, domain.ErrNotFound)
		res, err := svc.IsLoggedIn(ctx, cookie)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error", func(t *testing.T) {
		sessionStore.EXPECT().GetSessionBySessionID(ctx, cookie.Value).Return(nil, errors.New("db"))
		res, err := svc.IsLoggedIn(ctx, cookie)
		assert.Nil(t, res)
		assert.Error(t, err)
	})
}

func TestAuthService_AddSession(t *testing.T) {
	svc, _, sessionStore, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := 1

	t.Run("Success", func(t *testing.T) {
		tokens := &domain.SIDAndSCRFToken{SID: "sid", CSRFToken: "csrf"}
		sessionStore.EXPECT().AddSession(ctx, userID).Return(tokens, nil)
		res, err := svc.AddSession(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, tokens, res)
	})

	t.Run("Error", func(t *testing.T) {
		sessionStore.EXPECT().AddSession(ctx, userID).Return(nil, errors.New("db"))
		res, err := svc.AddSession(ctx, userID)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestAuthService_Login(t *testing.T) {
	svc, userStore, _, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := domain.User{Email: "test@test.com", Password: "123"}

	t.Run("Success", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user := domain.User{ID: 10, Email: req.Email, Password: string(hashed)}
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(&user, nil)
		id, err := svc.Login(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, 10, id)
	})

	t.Run("User not found", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, domain.ErrNotFound)
		id, err := svc.Login(ctx, req)
		assert.Equal(t, 0, id)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, errors.New("db"))
		id, err := svc.Login(ctx, req)
		assert.Equal(t, 0, id)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Invalid password", func(t *testing.T) {
		user := domain.User{ID: 10, Email: req.Email, Password: "wrong-hash"}
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(&user, nil)
		id, err := svc.Login(ctx, req)
		assert.Equal(t, 0, id)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}

func TestAuthService_Logout(t *testing.T) {
	svc, _, sessionStore, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cookie := &http.Cookie{Value: "abc"}

	t.Run("Success", func(t *testing.T) {
		sessionStore.EXPECT().DeleteSession(ctx, cookie.Value).Return(nil)
		err := svc.Logout(ctx, cookie)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		sessionStore.EXPECT().DeleteSession(ctx, cookie.Value).Return(errors.New("db"))
		err := svc.Logout(ctx, cookie)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestAuthService_Register(t *testing.T) {
	svc, userStore, _, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()

	req := domain.RegisterRequest{
		FirstName:       "Misha",
		LastName:        "Beztoksa",
		Email:           "misha@email.ru",
		Password:        "123456",
		ConfirmPassword: "123456",
		Gender:          "man",
	}

	t.Run("Success", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, domain.ErrNotFound)
		userStore.EXPECT().CreateUser(ctx, gomock.Any(), gomock.Any()).Return(1, nil)
		id, err := svc.Register(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, 1, id)
	})

	t.Run("Validation failed", func(t *testing.T) {
		invalid := domain.RegisterRequest{}
		id, err := svc.Register(ctx, invalid)
		assert.Equal(t, 0, id)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("User already exists", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(&domain.User{}, nil)
		id, err := svc.Register(ctx, req)
		assert.Equal(t, 0, id)
		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("DB error on GetUserByEmail", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, errors.New("db"))
		id, err := svc.Register(ctx, req)
		assert.Equal(t, 0, id)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Password mismatch", func(t *testing.T) {
		req2 := req
		req2.ConfirmPassword = "wrong"
		userStore.EXPECT().GetUserByEmail(ctx, req2.Email).Return(nil, domain.ErrNotFound)
		id, err := svc.Register(ctx, req2)
		assert.Equal(t, 0, id)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("CreateUser error", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, domain.ErrNotFound)
		userStore.EXPECT().CreateUser(ctx, gomock.Any(), gomock.Any()).Return(0, errors.New("db"))
		id, err := svc.Register(ctx, req)
		assert.Equal(t, 0, id)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}
