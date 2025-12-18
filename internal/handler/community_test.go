package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"project/domain"
	"project/internal/service/mocks"
)

// newRequestWithVarsShort создает запрос с параметрами пути
func newRequestWithVarsShort(t *testing.T, method, url string, vars map[string]string) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	req = mux.SetURLVars(req, vars)
	return req
}

func TestCommunityHandler_CreateCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		community := &domain.Community{
			ID:          1,
			Name:        "Test Community",
			Description: "Test Description",
			CreatorID:   1,
		}

		mockService.EXPECT().
			CreateCommunity(
				gomock.Any(),
				int32(1),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(community, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("name", "Test Community")
		writer.WriteField("description", "Test Description")
		writer.Close()

		req := httptest.NewRequest("POST", "/communities", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.CreateCommunity(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Message string `json:"message"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Community created successfully", response.Message)
	})

	t.Run("Missing name - service should be called and return error", func(t *testing.T) {
		mockService.EXPECT().
			CreateCommunity(
				gomock.Any(),
				int32(1),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(nil, errors.New("name is required"))

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("description", "Test Description")
		writer.Close()

		req := httptest.NewRequest("POST", "/communities", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.CreateCommunity(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCommunityHandler_UpdateCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		mockService.EXPECT().
			UpdateCommunity(
				gomock.Any(),
				int32(1),
				int32(1),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("name", "Updated Community")
		writer.Close()

		// Используем функцию с параметрами пути
		req := newRequestWithVarsShort(t, "PUT", "/communities/1", map[string]string{"id": "1"})
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Body = io.NopCloser(body)
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.UpdateCommunity(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Community updated successfully", response["message"])
	})

	t.Run("Invalid community ID", func(t *testing.T) {
		// Для нечислового ID
		req := newRequestWithVarsShort(t, "PUT", "/communities/invalid", map[string]string{"id": "invalid"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.UpdateCommunity(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCommunityHandler_DeleteCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		mockService.EXPECT().
			DeleteCommunity(gomock.Any(), int32(1), int32(1)).
			Return(nil)

		req := newRequestWithVarsShort(t, "DELETE", "/communities/1", map[string]string{"id": "1"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.DeleteCommunity(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Community deleted successfully", response["message"])
	})

	t.Run("Service error", func(t *testing.T) {
		mockService.EXPECT().
			DeleteCommunity(gomock.Any(), int32(1), int32(1)).
			Return(errors.New("service error"))

		req := newRequestWithVarsShort(t, "DELETE", "/communities/1", map[string]string{"id": "1"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.DeleteCommunity(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCommunityHandler_GetCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		expectedCommunity := &domain.CommunityForView{
			ID:          1,
			Name:        "Test Community",
			Description: "Test Description",
			CreatorID:   1,
		}

		mockService.EXPECT().
			GetCommunity(gomock.Any(), int32(1), int32(1)).
			Return(expectedCommunity, nil)

		req := newRequestWithVarsShort(t, "GET", "/communities/1", map[string]string{"id": "1"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetCommunity(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.CommunityForView
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expectedCommunity.ID, response.ID)
		assert.Equal(t, expectedCommunity.Name, response.Name)
	})
}

func TestCommunityHandler_Subscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		mockService.EXPECT().
			Subscribe(gomock.Any(), int32(1), int32(1)).
			Return(nil)

		req := newRequestWithVarsShort(t, "POST", "/communities/1/subscribe", map[string]string{"id": "1"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.Subscribe(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Subscribed successfully", response["message"])
	})
}

func TestCommunityHandler_Unsubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		mockService.EXPECT().
			Unsubscribe(gomock.Any(), int32(1), int32(1)).
			Return(nil)

		req := newRequestWithVarsShort(t, "POST", "/communities/1/unsubscribe", map[string]string{"id": "1"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.Unsubscribe(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Unsubscribed successfully", response["message"])
	})
}

func TestCommunityHandler_GetUserCommunitiesByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		expectedCommunities := []domain.ShortCommunity{
			{ID: 1, Name: "Community 1"},
		}

		mockService.EXPECT().
			GetUserCommunitiesByID(gomock.Any(), int32(2), gomock.Any()).
			Return(expectedCommunities, nil)

		req := newRequestWithVarsShort(t, "GET", "/communities/users/2", map[string]string{"id": "2"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetUserCommunitiesByID(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []domain.ShortCommunity
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
	})
}

func TestCommunityHandler_GetUserSubscribedCommunityIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		expectedIDs := []int32{1, 2, 3}

		mockService.EXPECT().
			GetUserSubscribedCommunityIDs(gomock.Any(), int32(2)).
			Return(expectedIDs, nil)

		req := newRequestWithVarsShort(t, "GET", "/communities/users/2/subscribed-ids", map[string]string{"userID": "2"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetUserSubscribedCommunityIDs(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Проверяем что ответ - просто массив
		var response []int32
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expectedIDs, response)
	})
}

func TestCommunityHandler_GetCommunitySubscribers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockCommunityService(ctrl)
	handler := NewCommunityHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		expectedSubscribers := []domain.CommunitySubscriber{
			{UserID: 1, FullName: "user1"},
		}

		mockService.EXPECT().
			GetCommunitySubscribers(gomock.Any(), int32(1), gomock.Any()).
			Return(expectedSubscribers, nil)

		req := newRequestWithVarsShort(t, "GET", "/communities/1/subscribers", map[string]string{"id": "1"})
		ctx := context.WithValue(req.Context(), domain.UserIDKey, int32(1))
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		handler.GetCommunitySubscribers(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Проверяем что ответ - просто массив
		var response []domain.CommunitySubscriber
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, expectedSubscribers[0].UserID, response[0].UserID)
		assert.Equal(t, expectedSubscribers[0].FullName, response[0].FullName)
	})
}
