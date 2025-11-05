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
	})

	t.Run("Invalid userID", func(t *testing.T) {
		req := newRequestWithVarsAndCtx(http.MethodPost, "/friends/invalid", map[string]string{"id": "invalid"}, 0, nil, t)
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
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
}
