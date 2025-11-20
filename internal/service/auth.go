package service

import (
	"context"
	"errors"
	"net/http"
	"project/domain"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	sessionStore        domain.SessionStore
	userStore           domain.UserStore
	elasticProfileStore domain.ElasticProfileStore
}

func NewAuthService(userStore domain.UserStore, sessionStore domain.SessionStore, elasticProfileStore domain.ElasticProfileStore) domain.AuthService {
	return &AuthService{
		sessionStore:        sessionStore,
		userStore:           userStore,
		elasticProfileStore: elasticProfileStore,
	}
}

func (api *AuthService) IsLoggedIn(ctx context.Context, sessionCookie *http.Cookie) (*domain.Session, error) {

	session, err := api.sessionStore.GetSessionBySessionID(ctx, sessionCookie.Value)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.FromContext(ctx).Warn("Session not found")
			return nil, err
		}
		domain.FromContext(ctx).Error("Session found error:", zap.Error(err))
		return nil, err
	}

	domain.FromContext(ctx).Info("Session loaded")
	return session, nil
}

func (api *AuthService) AddSession(ctx context.Context, userID int) (*domain.SIDAndSCRFToken, error) {
	tokens, err := api.sessionStore.AddSession(ctx, userID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to add session", zap.Error(err))
		return nil, domain.ErrDB
	}

	return tokens, nil
}

func (api *AuthService) Login(ctx context.Context, req domain.User) (int, error) {

	user, err := api.userStore.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			domain.FromContext(ctx).Error("User by email does not exist", zap.Error(err))
			return 0, domain.ErrNotFound
		} else {
			domain.FromContext(ctx).Error("Failed to get user by email", zap.Error(err))
			return 0, domain.ErrDB
		}

	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		domain.FromContext(ctx).Warn(domain.IncorrectPassword)
		return 0, domain.ErrInvalidInput
	}

	domain.FromContext(ctx).Info("User logged in", zap.Int("userID", user.ID))
	return user.ID, nil
}

func (api *AuthService) Logout(ctx context.Context, session *http.Cookie) error {

	err := api.sessionStore.DeleteSession(ctx, session.Value)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to logout", zap.Error(err))
		return domain.ErrDB
	}

	domain.FromContext(ctx).Info("User logged out")
	return nil
}

func (api *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (int, error) {

	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		domain.FromContext(ctx).Error("Register validate failed", zap.Error(err))
		return 0, domain.ErrInvalidInput
	}

	_, err = api.userStore.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			domain.FromContext(ctx).Error("Failed to get user by email", zap.Error(err))
			return 0, domain.ErrDB
		}
	} else {
		domain.FromContext(ctx).Warn("User already exist")
		return 0, domain.ErrAlreadyExists
	}

	if req.Password != req.ConfirmPassword {
		domain.FromContext(ctx).Info("Register validate failed: password filed doesn't match")
		return 0, domain.ErrInvalidInput
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to generate hashed password", zap.Error(err))
		return 0, domain.ErrService
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
	userID, err := api.userStore.CreateUser(ctx, user, profile)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to create user", zap.Error(err))
		return 0, domain.ErrDB
	}

	fullName := profile.FirstName + " " + profile.LastName
	err = api.elasticProfileStore.CreateProfile(ctx, fullName, userID)
	if err != nil {
		domain.FromContext(ctx).Error("Failed to update profile index in es", zap.Error(err))
		return 0, domain.ErrDB
	}

	domain.FromContext(ctx).Info("User created, registration complete", zap.Int("userID", userID))
	return userID, nil
}

func (api *AuthService) GetUserRole(ctx context.Context, userID int) (string, error) {
	user, err := api.userStore.GetUserByID(ctx, userID)
	return user.Role, err
}
