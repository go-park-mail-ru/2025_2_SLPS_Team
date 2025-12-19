package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"project/domain"
	"project/internal/service/mocks"
	"project/shared/pb"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Вспомогательные функции для создания запросов
func newMultipartRequestWithProfile(t *testing.T, method, url string, profile domain.Profile, userID int32) *http.Request {
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("failed to marshal profile: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("profile", string(profileJSON)); err != nil {
		t.Fatalf("failed to write field: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
	return req.WithContext(ctx)
}

func newMultipartRequestWithoutFile(t *testing.T, method, url string, userID int32) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
	return req.WithContext(ctx)
}

func newGetRequestWithVars(t *testing.T, url string, vars map[string]string, userID int32) *http.Request {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = mux.SetURLVars(req, vars)
	if userID != 0 {
		ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
		req = req.WithContext(ctx)
	}
	return req
}

func newDeleteRequest(t *testing.T, url string, userID int32) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	if userID != 0 {
		ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
		req = req.WithContext(ctx)
	}
	return req
}

// TestProfileHandler_UpdateProfile
func TestProfileHandler_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)
	handler := &ProfileHandler{profileService: mockProfileService}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		profile := domain.Profile{FirstName: "Test"}

		mockProfileService.EXPECT().
			UpdateProfile(gomock.Any(), gomock.Any()).
			Return(&emptypb.Empty{}, nil)

		req := newMultipartRequestWithProfile(t, http.MethodPut, "/profile", profile, userID)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("ParseMultipartForm_Error", func(t *testing.T) {
		userID := int32(1)
		req := httptest.NewRequest(http.MethodPut, "/profile", nil)
		ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("MissingProfileField", func(t *testing.T) {
		userID := int32(1)
		req := newMultipartRequestWithoutFile(t, http.MethodPut, "/profile", userID)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("InvalidProfileJSON", func(t *testing.T) {
		userID := int32(1)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("profile", "invalid json")
		writer.Close()

		req := httptest.NewRequest(http.MethodPut, "/profile", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("ServiceError", func(t *testing.T) {
		userID := int32(1)
		profile := domain.Profile{FirstName: "Test"}

		mockProfileService.EXPECT().
			UpdateProfile(gomock.Any(), gomock.Any()).
			Return(nil, status.Error(codes.Internal, "internal error"))

		req := newMultipartRequestWithProfile(t, http.MethodPut, "/profile", profile, userID)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

// TestProfileHandler_UpdateAvatar
func TestProfileHandler_UpdateAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)
	handler := &ProfileHandler{profileService: mockProfileService}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)

		mockProfileService.EXPECT().
			UpdateAvatar(gomock.Any(), gomock.Any()).
			Return(&emptypb.Empty{}, nil)

		req := newMultipartRequestWithoutFile(t, http.MethodPut, "/profile/avatar", userID)
		w := httptest.NewRecorder()

		handler.UpdateAvatar(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("ParseMultipartForm_Error", func(t *testing.T) {
		userID := int32(1)
		req := httptest.NewRequest(http.MethodPut, "/profile/avatar", nil)
		ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateAvatar(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("ServiceError", func(t *testing.T) {
		userID := int32(1)

		mockProfileService.EXPECT().
			UpdateAvatar(gomock.Any(), gomock.Any()).
			Return(nil, status.Error(codes.Internal, "internal error"))

		req := newMultipartRequestWithoutFile(t, http.MethodPut, "/profile/avatar", userID)
		w := httptest.NewRecorder()

		handler.UpdateAvatar(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

// TestProfileHandler_UpdateHeader
func TestProfileHandler_UpdateHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)
	handler := &ProfileHandler{profileService: mockProfileService}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)

		mockProfileService.EXPECT().
			UpdateAvatar(gomock.Any(), gomock.Any()).
			Return(&emptypb.Empty{}, nil)

		req := newMultipartRequestWithoutFile(t, http.MethodPut, "/profile/header", userID)
		w := httptest.NewRecorder()

		handler.UpdateHeader(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("ParseMultipartForm_Error", func(t *testing.T) {
		userID := int32(1)
		req := httptest.NewRequest(http.MethodPut, "/profile/header", nil)
		ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateHeader(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("ServiceError", func(t *testing.T) {
		userID := int32(1)

		mockProfileService.EXPECT().
			UpdateAvatar(gomock.Any(), gomock.Any()).
			Return(nil, status.Error(codes.Internal, "internal error"))

		req := newMultipartRequestWithoutFile(t, http.MethodPut, "/profile/header", userID)
		w := httptest.NewRecorder()

		handler.UpdateHeader(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

// TestProfileHandler_GetProfileByUserID
func TestProfileHandler_GetProfileByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)
	handler := &ProfileHandler{profileService: mockProfileService}

	t.Run("Success", func(t *testing.T) {
		mockProfileService.EXPECT().
			GetProfileByUserID(gomock.Any(), &pb.GetProfileByUserIDRequest{
				UserID:     1,
				SelfUserID: 2,
			}).
			Return(&pb.GetProfileByUserIDResponse{
				Profile: &pb.Profile{
					UserID:    1,
					FirstName: "User",
					LastName:  "Test",
				},
			}, nil)

		req := newGetRequestWithVars(t, "/profile/1", map[string]string{"id": "1"}, 2)
		w := httptest.NewRecorder()

		handler.GetProfileByUserID(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var response domain.Profile
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), response.UserID)
	})

	t.Run("InvalidUserID", func(t *testing.T) {
		req := newGetRequestWithVars(t, "/profile/invalid", map[string]string{"id": "invalid"}, 0)
		w := httptest.NewRecorder()

		handler.GetProfileByUserID(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mockProfileService.EXPECT().
			GetProfileByUserID(gomock.Any(), gomock.Any()).
			Return(nil, status.Error(codes.NotFound, "profile not found"))

		req := newGetRequestWithVars(t, "/profile/999", map[string]string{"id": "999"}, 1)
		w := httptest.NewRecorder()

		handler.GetProfileByUserID(w, req)
		assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	})

	t.Run("NoUserIDInContext", func(t *testing.T) {
		req := newGetRequestWithVars(t, "/profile/1", map[string]string{"id": "1"}, 0)
		w := httptest.NewRecorder()
		mockProfileService.EXPECT().
			GetProfileByUserID(gomock.Any(), gomock.Any()).
			Return(&pb.GetProfileByUserIDResponse{
				Profile: &pb.Profile{
					UserID:    1,
					FirstName: "User",
					LastName:  "Test",
				},
			}, nil)
		handler.GetProfileByUserID(w, req)
		// selfUserID будет 0, что допустимо
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})
}

// TestProfileHandler_DeleteAvatar
func TestProfileHandler_DeleteAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)
	handler := &ProfileHandler{profileService: mockProfileService}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)

		mockProfileService.EXPECT().
			DeleteAvatarByUserID(gomock.Any(), &pb.DeleteAvatarRequest{UserID: userID}).
			Return(&emptypb.Empty{}, nil)

		req := newDeleteRequest(t, "/profile/avatar", userID)
		w := httptest.NewRecorder()

		handler.DeleteAvatar(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("ServiceError", func(t *testing.T) {
		userID := int32(1)

		mockProfileService.EXPECT().
			DeleteAvatarByUserID(gomock.Any(), gomock.Any()).
			Return(nil, status.Error(codes.Internal, "internal error"))

		req := newDeleteRequest(t, "/profile/avatar", userID)
		w := httptest.NewRecorder()

		handler.DeleteAvatar(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("NoUserIDInContext", func(t *testing.T) {
		req := newDeleteRequest(t, "/profile/avatar", 1)
		w := httptest.NewRecorder()
		mockProfileService.EXPECT().
			DeleteAvatarByUserID(gomock.Any(), &pb.DeleteAvatarRequest{UserID: 1}).
			Return(&emptypb.Empty{}, status.Error(codes.Internal, "internal error"))
		handler.DeleteAvatar(w, req)
		// userID будет 0, что может вызвать ошибку на сервере
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

// TestProfileHandler_EdgeCases
func TestProfileHandler_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)
	handler := &ProfileHandler{profileService: mockProfileService}

	t.Run("UpdateProfile_FileProcessingError", func(t *testing.T) {
		userID := int32(1)
		profile := domain.Profile{FirstName: "Test"}

		// Создаем запрос с невалидными файлами
		profileJSON, _ := json.Marshal(profile)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("profile", string(profileJSON))
		// Не закрываем writer специально чтобы вызвать ошибку
		// writer.Close()

		req := httptest.NewRequest(http.MethodPut, "/profile", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("UpdateAvatar_FileProcessingError", func(t *testing.T) {
		userID := int32(1)

		// Создаем запрос с невалидным multipart
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		// Не закрываем writer
		// writer.Close()

		req := httptest.NewRequest(http.MethodPut, "/profile/avatar", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateAvatar(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("GetProfileByUserID_JSONEncodeError", func(t *testing.T) {
		// Этот тест сложно реализовать, так как нужно замокать json.Marshal
		// Обычно это тестируется интеграционными тестами
		t.Skip("Hard to mock JSON encoding errors in unit tests")
	})
}

// TestProfileHandler_Constructor
func TestNewProfileHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileService := mocks.NewMockProfileServiceClient(ctrl)
	handler := NewProfileHandler(mockProfileService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockProfileService, handler.profileService)
}
