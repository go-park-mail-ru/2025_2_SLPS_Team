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
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

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

func newGetRequestWithVars(t *testing.T, url string, vars map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	return mux.SetURLVars(req, vars)
}

func TestProfileHandler_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockProfileService(ctrl)
	handler := &ProfileHandler{profileService: mockService}

	t.Run("Success test", func(t *testing.T) {
		userID := 1
		profile := domain.Profile{FirstName: "Test"}

		mockService.EXPECT().UpdateProfile(gomock.Any(), profile, userID, nil).Return(nil)

		req := newMultipartRequestWithProfile(t, http.MethodPut, "/profile", profile, userID)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Missing profile", func(t *testing.T) {
		req := newMultipartRequestWithoutFile(t, http.MethodPut, "/profile", 1)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}

func TestProfileHandler_UpdateAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockProfileService(ctrl)
	handler := &ProfileHandler{profileService: mockService}

	t.Run("Success test", func(t *testing.T) {
		userID := 1
		mockService.EXPECT().UpdateAvatar(gomock.Any(), userID, nil).Return(nil)

		req := newMultipartRequestWithoutFile(t, http.MethodPut, "/profile/avatar", userID)
		w := httptest.NewRecorder()

		handler.UpdateAvatar(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})
}

func TestProfileHandler_UpdateHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockProfileService(ctrl)
	handler := &ProfileHandler{profileService: mockService}

	t.Run("Success test", func(t *testing.T) {
		userID := 1
		mockService.EXPECT().UpdateHeader(gomock.Any(), userID, nil).Return(nil)

		req := newMultipartRequestWithoutFile(t, http.MethodPut, "/profile/header", userID)
		w := httptest.NewRecorder()

		handler.UpdateHeader(w, req)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})
}

func TestProfileHandler_GetProfileByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockProfileService(ctrl)
	handler := &ProfileHandler{profileService: mockService}

	t.Run("Success test", func(t *testing.T) {
		userID := 1
		profile := domain.Profile{FirstName: "User"}
		mockService.EXPECT().GetProfileByUserID(gomock.Any(), userID).Return(&profile, nil)

		req := newGetRequestWithVars(t, "/profile/1", map[string]string{"id": strconv.Itoa(userID)})
		w := httptest.NewRecorder()

		handler.GetProfileByUserID(w, req)

		var res domain.Profile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, profile, res)
	})

	t.Run("Invalid userID", func(t *testing.T) {
		req := newGetRequestWithVars(t, "/profile/invalid", map[string]string{"id": "invalid"})
		w := httptest.NewRecorder()

		handler.GetProfileByUserID(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}
