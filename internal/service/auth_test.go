package service

import (
	"context"
	"errors"
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
	sessionID := "abc123"

	t.Run("Success", func(t *testing.T) {
		session := &domain.Session{UserID: 1}
		sessionStore.EXPECT().GetSessionBySessionID(ctx, sessionID).Return(session, nil)
		res, err := svc.IsLoggedIn(ctx, sessionID)
		assert.NoError(t, err)
		assert.Equal(t, session, res)
	})

	t.Run("Not found", func(t *testing.T) {
		sessionStore.EXPECT().GetSessionBySessionID(ctx, sessionID).Return(nil, domain.ErrNotFound)
		res, err := svc.IsLoggedIn(ctx, sessionID)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error", func(t *testing.T) {
		sessionStore.EXPECT().GetSessionBySessionID(ctx, sessionID).Return(nil, errors.New("dbconn"))
		res, err := svc.IsLoggedIn(ctx, sessionID)
		assert.Nil(t, res)
		assert.Error(t, err)
	})
}

func TestAuthService_AddSession(t *testing.T) {
	svc, _, sessionStore, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := int32(1)

	t.Run("Success", func(t *testing.T) {
		tokens := &domain.SIDAndSCRFToken{SID: "sid", CSRFToken: "csrf"}
		sessionStore.EXPECT().AddSession(ctx, userID).Return(tokens, nil)
		res, err := svc.AddSession(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, tokens, res)
	})

	t.Run("Error", func(t *testing.T) {
		sessionStore.EXPECT().AddSession(ctx, userID).Return(nil, errors.New("dbconn"))
		res, err := svc.AddSession(ctx, userID)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestAuthService_Login(t *testing.T) {
	svc, userStore, _, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := domain.User{Email: "test@test.com", Password: "123456"}

	t.Run("Success", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user := domain.User{ID: 10, Email: req.Email, Password: string(hashed)}
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(&user, nil)
		id, err := svc.Login(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, int32(10), id)
	})

	t.Run("User not found", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, domain.ErrNotFound)
		id, err := svc.Login(ctx, req)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("DB error", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, errors.New("dbconn"))
		id, err := svc.Login(ctx, req)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Invalid password", func(t *testing.T) {
		user := domain.User{ID: 10, Email: req.Email, Password: "wrong-hash"}
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(&user, nil)
		id, err := svc.Login(ctx, req)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}

func TestAuthService_Logout(t *testing.T) {
	svc, _, sessionStore, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	sessionID := "abc123"

	t.Run("Success", func(t *testing.T) {
		sessionStore.EXPECT().DeleteSession(ctx, sessionID).Return(nil)
		err := svc.Logout(ctx, sessionID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		sessionStore.EXPECT().DeleteSession(ctx, sessionID).Return(errors.New("dbconn"))
		err := svc.Logout(ctx, sessionID)
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
		userStore.EXPECT().CreateUser(ctx, gomock.Any()).Return(int32(1), nil)
		id, err := svc.Register(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), id)
	})

	t.Run("Validation failed", func(t *testing.T) {
		invalid := domain.RegisterRequest{}
		id, err := svc.Register(ctx, invalid)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("User already exists", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(&domain.User{}, nil)
		id, err := svc.Register(ctx, req)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("DB error on GetUserByEmail", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, errors.New("dbconn"))
		id, err := svc.Register(ctx, req)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrDB)
	})

	t.Run("Password mismatch", func(t *testing.T) {
		req2 := req
		req2.ConfirmPassword = "wrong"
		userStore.EXPECT().GetUserByEmail(ctx, req2.Email).Return(nil, domain.ErrNotFound)
		id, err := svc.Register(ctx, req2)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("CreateUser error", func(t *testing.T) {
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, domain.ErrNotFound)
		userStore.EXPECT().CreateUser(ctx, gomock.Any()).Return(int32(0), errors.New("dbconn"))
		id, err := svc.Register(ctx, req)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestAuthService_GetUserRole(t *testing.T) {
	svc, userStore, _, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := int32(1)

	t.Run("Success", func(t *testing.T) {
		user := &domain.User{Role: "user"}
		userStore.EXPECT().GetUserByID(ctx, userID).Return(user, nil)
		role, err := svc.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, "user", role)
	})

	t.Run("DB error", func(t *testing.T) {
		userStore.EXPECT().GetUserByID(ctx, userID).Return(nil, errors.New("dbconn"))
		role, err := svc.GetUserRole(ctx, userID)
		assert.Equal(t, "", role)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

func TestAuthService_IsUserExists(t *testing.T) {
	svc, userStore, _, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := int32(1)

	t.Run("Success - user exists", func(t *testing.T) {
		userStore.EXPECT().IsUserExists(ctx, userID).Return(true, nil)
		exists, err := svc.IsUserExists(ctx, userID)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Success - user not exists", func(t *testing.T) {
		userStore.EXPECT().IsUserExists(ctx, userID).Return(false, nil)
		exists, err := svc.IsUserExists(ctx, userID)
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("DB error", func(t *testing.T) {
		userStore.EXPECT().IsUserExists(ctx, userID).Return(false, errors.New("dbconn"))
		exists, err := svc.IsUserExists(ctx, userID)
		assert.False(t, exists)
		assert.ErrorIs(t, err, domain.ErrDB)
	})
}

// Edge case тесты
func TestAuthService_EdgeCases(t *testing.T) {
	svc, userStore, _, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("Empty email login", func(t *testing.T) {
		req := domain.User{Email: "", Password: "123"}
		id, err := svc.Login(ctx, req)
		assert.Equal(t, int32(0), id)
		assert.Error(t, err)
	})

	t.Run("Empty password login", func(t *testing.T) {
		req := domain.User{Email: "test@test.com", Password: ""}
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(&domain.User{Password: "hash"}, nil)
		id, err := svc.Login(ctx, req)
		assert.Equal(t, int32(0), id)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Very long password", func(t *testing.T) {
		longPassword := "a"
		for i := 0; i < 1000; i++ {
			longPassword += "a"
		}
		req := domain.RegisterRequest{
			FirstName:       "Test",
			LastName:        "User",
			Email:           "test@test.com",
			Password:        longPassword,
			ConfirmPassword: longPassword,
		}
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, domain.ErrNotFound)
		userStore.EXPECT().CreateUser(ctx, gomock.Any()).Return(int32(1), nil)
		id, err := svc.Register(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), id)
	})

	t.Run("Special characters in email", func(t *testing.T) {
		req := domain.RegisterRequest{
			FirstName:       "Test",
			LastName:        "User",
			Email:           "test.user+tag@sub.domain.com",
			Password:        "123456",
			ConfirmPassword: "123456",
		}
		userStore.EXPECT().GetUserByEmail(ctx, req.Email).Return(nil, domain.ErrNotFound)
		userStore.EXPECT().CreateUser(ctx, gomock.Any()).Return(int32(1), nil)
		id, err := svc.Register(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), id)
	})
}

// Тесты для проверки валидации
func TestAuthService_Validation(t *testing.T) {
	svc, _, _, ctrl := newAuthServiceMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()

	testCases := []struct {
		name    string
		req     domain.RegisterRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			req: domain.RegisterRequest{
				FirstName:       "John",
				LastName:        "Doe",
				Email:           "john@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			wantErr: false,
		},
		{
			name: "Invalid email",
			req: domain.RegisterRequest{
				FirstName:       "John",
				LastName:        "Doe",
				Email:           "invalid-email",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			wantErr: true,
		},
		{
			name: "Missing first name",
			req: domain.RegisterRequest{
				FirstName:       "",
				LastName:        "Doe",
				Email:           "john@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			wantErr: true,
		},
		{
			name: "Missing last name",
			req: domain.RegisterRequest{
				FirstName:       "John",
				LastName:        "",
				Email:           "john@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			wantErr: true,
		},
		{
			name: "Short password",
			req: domain.RegisterRequest{
				FirstName:       "John",
				LastName:        "Doe",
				Email:           "john@example.com",
				Password:        "123",
				ConfirmPassword: "123",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			id, err := svc.Register(ctx, tc.req)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Equal(t, int32(0), id)
			} else {

				assert.Error(t, err)
			}
		})
	}
}
