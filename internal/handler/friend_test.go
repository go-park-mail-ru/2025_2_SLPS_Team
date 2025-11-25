package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"project/domain"
	"project/internal/service/mocks"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// FriendsCountResponse - ответ с количеством отношений
type FriendsCountResponse struct {
	UserID    int32                      `json:"userID"`
	Count     int32                      `json:"count"`
	CountType domain.FriendshipCountType `json:"countType,omitempty"`
}

func TestFriendHandler_SendFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().SendFriendRequest(gomock.Any(), userID, friendID).Return(nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRequestSent, response.Message)
	})

	t.Run("Invalid userID in context", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})

	t.Run("Invalid friendID in vars", func(t *testing.T) {
		userID := 1
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/invalid",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().SendFriendRequest(gomock.Any(), userID, friendID).Return(domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

func TestFriendHandler_AcceptFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().AcceptFriendRequest(gomock.Any(), userID, friendID).Return(nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/accept",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRequestAccepted, response.Message)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		userID := 1
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/invalid/accept",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().AcceptFriendRequest(gomock.Any(), userID, friendID).Return(domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/accept",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/accept",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}

func TestFriendHandler_RejectFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().RejectFriendRequest(gomock.Any(), userID, friendID).Return(nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/reject",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRequestRejected, response.Message)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		userID := 1
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/invalid/reject",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().RejectFriendRequest(gomock.Any(), userID, friendID).Return(domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/reject",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/reject",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}

func TestFriendHandler_RemoveFriend(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().RemoveFriend(gomock.Any(), userID, friendID).Return(nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodDelete,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRemoved, response.Message)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		userID := 1
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodDelete,
			URL:     "/friends/invalid",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().RemoveFriend(gomock.Any(), userID, friendID).Return(domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodDelete,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodDelete,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}

func TestFriendHandler_GetFriendshipStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success with status", func(t *testing.T) {
		userID := 1
		friendID := 2
		status := domain.FriendshipAccepted
		mockService.EXPECT().GetFriendshipStatus(gomock.Any(), userID, friendID).Return(status, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/status",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res FriendshipStatusResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, status, res.Status)
	})

	t.Run("Success no friendship", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().GetFriendshipStatus(gomock.Any(), userID, friendID).Return(domain.FriendshipStatus(""), nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/status",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res FriendshipStatusResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipStatus(""), res.Status)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		userID := 1
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/invalid/status",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().GetFriendshipStatus(gomock.Any(), userID, friendID).Return(domain.FriendshipStatus(""), domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/status",
			Vars:    map[string]string{"id": strconv.Itoa(friendID)},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/status",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}

func TestFriendHandler_GetFriends(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		friends := []domain.ShortProfile{
			{UserID: 2, FullName: "John Doe"},
			{UserID: 3, FullName: "Jane Smith"},
		}
		mockService.EXPECT().GetFriends(gomock.Any(), userID, gomock.Any()).Return(friends, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res []domain.ShortProfile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})

	t.Run("Success with pagination", func(t *testing.T) {
		userID := 1
		friends := []domain.ShortProfile{{UserID: 2}}
		mockService.EXPECT().GetFriends(gomock.Any(), userID, domain.PaginateQueryParams{
			Limit: 10,
			Page:  2,
		}).Return(friends, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends?limit=10&page=2",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		mockService.EXPECT().GetFriends(gomock.Any(), userID, gomock.Any()).Return(nil, domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends",
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}

func TestFriendHandler_GetFriendRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		requests := []domain.ShortProfile{
			{UserID: 2, FullName: "Requester 1"},
		}
		mockService.EXPECT().GetFriendRequests(gomock.Any(), userID, gomock.Any()).Return(requests, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/requests",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendRequests(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res []domain.ShortProfile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		mockService.EXPECT().GetFriendRequests(gomock.Any(), userID, gomock.Any()).Return(nil, domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/requests",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendRequests(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/requests",
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetFriendRequests(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}

func TestFriendHandler_GetSentRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		requests := []domain.ShortProfile{
			{UserID: 2, FullName: "Receiver 1"},
		}
		mockService.EXPECT().GetSentRequests(gomock.Any(), userID, gomock.Any()).Return(requests, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/sent",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetSentRequests(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res []domain.ShortProfile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		mockService.EXPECT().GetSentRequests(gomock.Any(), userID, gomock.Any()).Return(nil, domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/sent",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetSentRequests(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/sent",
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetSentRequests(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}

func TestFriendHandler_CountUserRelations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		targetUserID := 2
		countType := domain.CountAccepted
		count := 5

		mockService.EXPECT().CountUserRelations(gomock.Any(), targetUserID, countType).Return(count, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count?type=accepted",
			Vars:    map[string]string{"id": strconv.Itoa(targetUserID)},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res FriendsCountResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, count, res.Count)
		assert.Equal(t, targetUserID, res.UserID)
		assert.Equal(t, countType, res.CountType)
	})

	t.Run("Invalid count type", func(t *testing.T) {
		targetUserID := 2
		countType := domain.FriendshipCountType("invalid")

		// ВАЖНО: ожидаем вызов сервиса, он вернет ошибку валидации
		mockService.EXPECT().CountUserRelations(gomock.Any(), targetUserID, countType).
			Return(0, domain.ErrInvalidInput)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count?type=invalid",
			Vars:    map[string]string{"id": strconv.Itoa(targetUserID)},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		// Сервис вернет ErrInvalidInput, который мапится в 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Missing count type - uses default", func(t *testing.T) {
		targetUserID := 2
		count := 3
		// Должен использовать CountAccepted по умолчанию
		mockService.EXPECT().CountUserRelations(gomock.Any(), targetUserID, domain.CountAccepted).Return(count, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count", // без параметра type
			Vars:    map[string]string{"id": strconv.Itoa(targetUserID)},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res FriendsCountResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, count, res.Count)
		assert.Equal(t, domain.CountAccepted, res.CountType)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		// НЕ ожидаем вызов сервиса - ошибка парсинга ID ДО вызова сервиса
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/invalid/count?type=accepted",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		targetUserID := 2
		countType := domain.CountAccepted
		mockService.EXPECT().CountUserRelations(gomock.Any(), targetUserID, countType).Return(0, domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count?type=accepted",
			Vars:    map[string]string{"id": strconv.Itoa(targetUserID)},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Count pending", func(t *testing.T) {
		targetUserID := 2
		countType := domain.CountPending
		count := 3

		mockService.EXPECT().CountUserRelations(gomock.Any(), targetUserID, countType).Return(count, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count?type=pending",
			Vars:    map[string]string{"id": strconv.Itoa(targetUserID)},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Count sent", func(t *testing.T) {
		targetUserID := 2
		countType := domain.CountSent
		count := 2

		mockService.EXPECT().CountUserRelations(gomock.Any(), targetUserID, countType).Return(count, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count?type=sent",
			Vars:    map[string]string{"id": strconv.Itoa(targetUserID)},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Count blocked", func(t *testing.T) {
		targetUserID := 2
		countType := domain.CountBlocked
		count := 1

		mockService.EXPECT().CountUserRelations(gomock.Any(), targetUserID, countType).Return(count, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count?type=blocked",
			Vars:    map[string]string{"id": strconv.Itoa(targetUserID)},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Count rejected", func(t *testing.T) {
		targetUserID := 2
		countType := domain.CountRejected
		count := 0

		mockService.EXPECT().CountUserRelations(gomock.Any(), targetUserID, countType).Return(count, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count?type=rejected",
			Vars:    map[string]string{"id": strconv.Itoa(targetUserID)},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})
}

func TestFriendHandler_GetAllUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		users := []domain.ShortProfile{
			{UserID: 2, FullName: "User 2"},
			{UserID: 3, FullName: "User 3"},
		}
		mockService.EXPECT().GetAllUsers(gomock.Any(), userID, gomock.Any()).Return(users, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/users/all",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res []domain.ShortProfile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})

	t.Run("Success with pagination", func(t *testing.T) {
		userID := 1
		users := []domain.ShortProfile{{UserID: 2}}
		mockService.EXPECT().GetAllUsers(gomock.Any(), userID, domain.PaginateQueryParams{
			Limit: 5,
			Page:  1,
		}).Return(users, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/users/all?limit=5&page=1",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		mockService.EXPECT().GetAllUsers(gomock.Any(), userID, gomock.Any()).Return(nil, domain.ErrDB)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/users/all",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/users/all",
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})
}

func TestFriendHandler_DefaultPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Default pagination for friends", func(t *testing.T) {
		userID := 1
		friends := []domain.ShortProfile{{UserID: 2}}

		// Используем gomock.Any() так как schema.NewDecoder() устанавливает дефолтные значения
		mockService.EXPECT().GetFriends(gomock.Any(), userID, gomock.Any()).Return(friends, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends", // без параметров пагинации
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Default pagination for requests", func(t *testing.T) {
		userID := 1
		requests := []domain.ShortProfile{{UserID: 2}}

		mockService.EXPECT().GetFriendRequests(gomock.Any(), userID, gomock.Any()).Return(requests, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/requests", // без параметров пагинации
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendRequests(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Default pagination for all users", func(t *testing.T) {
		userID := 1
		users := []domain.ShortProfile{{UserID: 2}}

		mockService.EXPECT().GetAllUsers(gomock.Any(), userID, gomock.Any()).Return(users, nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/users/all", // без параметров пагинации
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})

	t.Run("Invalid pagination parameters", func(t *testing.T) {
		userID := 1

		// Не нужно мокать сервис, так как будет ошибка валидации до вызова сервиса
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends?limit=invalid", // невалидный limit
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}
