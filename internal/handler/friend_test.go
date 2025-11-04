// package handler

// import (
// 	"context"
// 	"encoding/json"
// 	"net/http"
// 	"net/http/httptest"
// 	"project/domain"
// 	"project/internal/service/mocks"
// 	"testing"

// 	"github.com/golang/mock/gomock"
// 	"github.com/gorilla/mux"
// 	"github.com/stretchr/testify/assert"
// )

// func TestFriendHandler_SendFriendRequest(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockFriendService := mocks.NewMockFriendService(ctrl)
// 	handler := NewFriendHandler(mockFriendService)

// 	t.Run("Success", func(t *testing.T) {
// 		mockFriendService.EXPECT().
// 			SendFriendRequest(gomock.Any(), 1, 2).
// 			Return(nil)

// 		req := newRequestWithVarsAndCtx(t, "/friends/2", map[string]string{"id": "2"}, 1)
// 		w := httptest.NewRecorder()

// 		handler.SendFriendRequest(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response JSONResponse
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, domain.FriendRequestSent, response.Message)
// 	})

// 	t.Run("Invalid user ID", func(t *testing.T) {
// 		req := newRequestWithVarsAndCtx(t, "/friends/invalid", map[string]string{"id": "invalid"}, 1)
// 		w := httptest.NewRecorder()

// 		handler.SendFriendRequest(w, req)

// 		assert.Equal(t, http.StatusBadRequest, w.Code)
// 	})

// 	t.Run("Unauthorized", func(t *testing.T) {
// 		req := newRequestWithVars(t, "/friends/2", map[string]string{"id": "2"})
// 		w := httptest.NewRecorder()

// 		handler.SendFriendRequest(w, req)

// 		assert.Equal(t, http.StatusUnauthorized, w.Code)
// 	})
// }

// func TestFriendHandler_AcceptFriendRequest(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockFriendService := mocks.NewMockFriendService(ctrl)
// 	handler := NewFriendHandler(mockFriendService)

// 	t.Run("Success", func(t *testing.T) {
// 		mockFriendService.EXPECT().
// 			AcceptFriendRequest(gomock.Any(), 1, 2).
// 			Return(nil)

// 		req := newRequestWithVarsAndCtx(t, "/friends/2/accept", map[string]string{"id": "2"}, 1)
// 		w := httptest.NewRecorder()

// 		handler.AcceptFriendRequest(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response JSONResponse
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, domain.FriendRequestAccepted, response.Message)
// 	})

// 	t.Run("Service returns error", func(t *testing.T) {
// 		mockFriendService.EXPECT().
// 			AcceptFriendRequest(gomock.Any(), 1, 2).
// 			Return(domain.ErrNotFound)

// 		req := newRequestWithVarsAndCtx(t, "/friends/2/accept", map[string]string{"id": "2"}, 1)
// 		w := httptest.NewRecorder()

// 		handler.AcceptFriendRequest(w, req)

// 		assert.Equal(t, http.StatusNotFound, w.Code)
// 	})
// }

// func TestFriendHandler_RejectFriendRequest(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockFriendService := mocks.NewMockFriendService(ctrl)
// 	handler := NewFriendHandler(mockFriendService)

// 	t.Run("Success", func(t *testing.T) {
// 		mockFriendService.EXPECT().
// 			RejectFriendRequest(gomock.Any(), 1, 2).
// 			Return(nil)

// 		req := newRequestWithVarsAndCtx(t, "/friends/2/reject", map[string]string{"id": "2"}, 1)
// 		w := httptest.NewRecorder()

// 		handler.RejectFriendRequest(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response JSONResponse
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, domain.FriendRequestRejected, response.Message)
// 	})
// }

// func TestFriendHandler_RemoveFriend(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockFriendService := mocks.NewMockFriendService(ctrl)
// 	handler := NewFriendHandler(mockFriendService)

// 	t.Run("Success", func(t *testing.T) {
// 		mockFriendService.EXPECT().
// 			RemoveFriend(gomock.Any(), 1, 2).
// 			Return(nil)

// 		req := newRequestWithVarsAndCtx(t, "/friends/2", map[string]string{"id": "2"}, 1)
// 		w := httptest.NewRecorder()

// 		handler.RemoveFriend(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response JSONResponse
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, domain.FriendRemoved, response.Message)
// 	})
// }

// func TestFriendHandler_GetFriends(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockFriendService := mocks.NewMockFriendService(ctrl)
// 	handler := NewFriendHandler(mockFriendService)

// 	t.Run("Success", func(t *testing.T) {
// 		friends := []domain.ShortProfile{
// 			{UserID: 2, FullName: "Friend One"},
// 			{UserID: 3, FullName: "Friend Two"},
// 		}
// 		totalPages := 1

// 		mockFriendService.EXPECT().
// 			GetFriends(gomock.Any(), 1, 1, 20).
// 			Return(friends, totalPages, nil)

// 		req := httptest.NewRequest("GET", "/friends?page=1&limit=20", nil)
// 		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
// 		req = req.WithContext(ctx)
// 		w := httptest.NewRecorder()

// 		handler.GetFriends(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response FriendsResponse
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Len(t, response.Friends, 2)
// 		assert.Equal(t, totalPages, response.TotalPages)
// 	})

// 	t.Run("Invalid query parameters", func(t *testing.T) {
// 		req := httptest.NewRequest("GET", "/friends?page=invalid", nil)
// 		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
// 		req = req.WithContext(ctx)
// 		w := httptest.NewRecorder()

// 		handler.GetFriends(w, req)

// 		assert.Equal(t, http.StatusBadRequest, w.Code)
// 	})
// }

// func TestFriendHandler_GetFriendRequests(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockFriendService := mocks.NewMockFriendService(ctrl)
// 	handler := NewFriendHandler(mockFriendService)

// 	t.Run("Success", func(t *testing.T) {
// 		requests := []domain.FriendshipWithProfile{
// 			{
// 				Friendship: domain.Friendship{
// 					ID:           1,
// 					FirstUserID:  1,
// 					SecondUserID: 2,
// 					Status:       domain.FriendshipPending,
// 				},
// 				Friend: domain.ShortProfile{UserID: 2, FullName: "Requester"},
// 			},
// 		}
// 		totalPages := 1

// 		mockFriendService.EXPECT().
// 			GetFriendRequests(gomock.Any(), 1, 1, 20).
// 			Return(requests, totalPages, nil)

// 		req := httptest.NewRequest("GET", "/friends/requests?page=1&limit=20", nil)
// 		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
// 		req = req.WithContext(ctx)
// 		w := httptest.NewRecorder()

// 		handler.GetFriendRequests(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response FriendRequestsResponse
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Len(t, response.Requests, 1)
// 		assert.Equal(t, totalPages, response.TotalPages)
// 	})
// }

// func TestFriendHandler_GetSentRequests(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockFriendService := mocks.NewMockFriendService(ctrl)
// 	handler := NewFriendHandler(mockFriendService)

// 	t.Run("Success", func(t *testing.T) {
// 		requests := []domain.FriendshipWithProfile{
// 			{
// 				Friendship: domain.Friendship{
// 					ID:           1,
// 					FirstUserID:  1,
// 					SecondUserID: 2,
// 					Status:       domain.FriendshipPending,
// 				},
// 				Friend: domain.ShortProfile{UserID: 2, FullName: "Receiver"},
// 			},
// 		}
// 		totalPages := 1

// 		mockFriendService.EXPECT().
// 			GetSentRequests(gomock.Any(), 1, 1, 20).
// 			Return(requests, totalPages, nil)

// 		req := httptest.NewRequest("GET", "/friends/sent?page=1&limit=20", nil)
// 		ctx := context.WithValue(req.Context(), domain.UserIDKey, 1)
// 		req = req.WithContext(ctx)
// 		w := httptest.NewRecorder()

// 		handler.GetSentRequests(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response FriendRequestsResponse
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Len(t, response.Requests, 1)
// 		assert.Equal(t, totalPages, response.TotalPages)
// 	})
// }

// func TestFriendHandler_GetFriendshipStatus(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockFriendService := mocks.NewMockFriendService(ctrl)
// 	handler := NewFriendHandler(mockFriendService)

// 	t.Run("Success", func(t *testing.T) {
// 		mockFriendService.EXPECT().
// 			GetFriendshipStatus(gomock.Any(), 1, 2).
// 			Return(domain.FriendshipAccepted, nil)

// 		req := newRequestWithVarsAndCtx(t, "/friends/2/status", map[string]string{"id": "2"}, 1)
// 		w := httptest.NewRecorder()

// 		handler.GetFriendshipStatus(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response FriendshipStatusResponse
// 		err := json.Unmarshal(w.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, domain.FriendshipAccepted, response.Status)
// 	})

// 	t.Run("Service returns error", func(t *testing.T) {
// 		mockFriendService.EXPECT().
// 			GetFriendshipStatus(gomock.Any(), 1, 2).
// 			Return("", domain.ErrDB)

// 		req := newRequestWithVarsAndCtx(t, "/friends/2/status", map[string]string{"id": "2"}, 1)
// 		w := httptest.NewRecorder()

// 		handler.GetFriendshipStatus(w, req)

// 		assert.Equal(t, http.StatusInternalServerError, w.Code)
// 	})
// }

// // Вспомогательные функции
// func newRequestWithVars(t *testing.T, url string, vars map[string]string) *http.Request {
// 	req := httptest.NewRequest("GET", url, nil)
// 	return mux.SetURLVars(req, vars)
// }

// func newRequestWithVarsAndCtx(t *testing.T, url string, vars map[string]string, userID int) *http.Request {
// 	req := newRequestWithVars(t, url, vars)
// 	ctx := context.WithValue(req.Context(), domain.UserIDKey, userID)
// 	return req.WithContext(ctx)
// }