package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"project/domain"
	"project/internal/service/mocks"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Вспомогательная функция для создания запроса с контекстом и переменными маршрута
func newRequestWithVarsAndCtx(method, url string, vars map[string]string, userID int, body interface{}, t *testing.T) *http.Request {
	var req *http.Request
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequest(method, url, bytes.NewBuffer(jsonBody))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}

	// Добавляем переменные маршрута
	req = mux.SetURLVars(req, vars)

	// Добавляем userID в контекст
	ctx := context.WithValue(req.Context(), "userID", userID)
	req = req.WithContext(ctx)

	return req
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

		req := newRequestWithVarsAndCtx(http.MethodPost, "/friends/2", map[string]string{"id": strconv.Itoa(friendID)}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRequestSent, response.Message)
	})

	t.Run("Invalid userID in context", func(t *testing.T) {
		req := newRequestWithVarsAndCtx(http.MethodPost, "/friends/2", map[string]string{"id": "2"}, 0, nil, t)
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})

	t.Run("Invalid friendID in vars", func(t *testing.T) {
		userID := 1
		req := newRequestWithVarsAndCtx(http.MethodPost, "/friends/invalid", map[string]string{"id": "invalid"}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().SendFriendRequest(gomock.Any(), userID, friendID).Return(domain.ErrDB)

		req := newRequestWithVarsAndCtx(http.MethodPost, "/friends/2", map[string]string{"id": strconv.Itoa(friendID)}, userID, nil, t)
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

		req := newRequestWithVarsAndCtx(http.MethodPut, "/friends/2/accept", map[string]string{"id": strconv.Itoa(friendID)}, userID, nil, t)
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
		req := newRequestWithVarsAndCtx(http.MethodPut, "/friends/invalid/accept", map[string]string{"id": "invalid"}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
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

		req := newRequestWithVarsAndCtx(http.MethodPut, "/friends/2/reject", map[string]string{"id": strconv.Itoa(friendID)}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRequestRejected, response.Message)
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

		req := newRequestWithVarsAndCtx(http.MethodDelete, "/friends/2", map[string]string{"id": strconv.Itoa(friendID)}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRemoved, response.Message)
	})
}

func TestFriendHandler_GetFriendshipStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		friendID := 2
		status := domain.FriendshipAccepted
		mockService.EXPECT().GetFriendshipStatus(gomock.Any(), userID, friendID).Return(status, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends/2/status", map[string]string{"id": strconv.Itoa(friendID)}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		var res FriendshipStatusResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, status, res.Status)
	})

	t.Run("No friendship", func(t *testing.T) {
		userID := 1
		friendID := 2
		mockService.EXPECT().GetFriendshipStatus(gomock.Any(), userID, friendID).Return(domain.FriendshipStatus(""), nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends/2/status", map[string]string{"id": strconv.Itoa(friendID)}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		var res FriendshipStatusResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, domain.FriendshipStatus(""), res.Status)
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

		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends", nil, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res []domain.ShortProfile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})

	t.Run("With pagination params", func(t *testing.T) {
		userID := 1
		friends := []domain.ShortProfile{{UserID: 2}}
		mockService.EXPECT().GetFriends(gomock.Any(), userID, domain.PaginateQueryParams{
			Limit: 10,
			Page:  2,
		}).Return(friends, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends?limit=10&page=2", nil, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
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

		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends/requests", nil, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetFriendRequests(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res []domain.ShortProfile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
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

		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends/sent", nil, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetSentRequests(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res []domain.ShortProfile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})
}

func TestFriendHandler_CountUserRelations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockFriendService(ctrl)
	handler := &FriendHandler{friendService: mockService}

	t.Run("Success", func(t *testing.T) {
		userID := 1
		countType := domain.CountAccepted
		count := 5
		mockService.EXPECT().CountUserRelations(gomock.Any(), userID, countType).Return(count, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends/2/count?type=accepted",
			map[string]string{"id": "2"}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res FriendsCountResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, count, res.Count)
	})

	t.Run("Invalid count type", func(t *testing.T) {
		userID := 1
		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends/2/count?type=invalid",
			map[string]string{"id": "2"}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
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

		req := newRequestWithVarsAndCtx(http.MethodGet, "/friends/users/all", nil, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)

		var res []domain.ShortProfile
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})
}
