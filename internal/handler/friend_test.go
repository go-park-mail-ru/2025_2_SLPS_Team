package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"project/domain"
	service_mocks "project/internal/service/mocks"
	pb "project/shared/pb"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// FriendTestRequestConfig - конфигурация для тестового запроса (переименовано чтобы избежать конфликта)
type FriendTestRequestConfig struct {
	Method  string
	URL     string
	Vars    map[string]string
	UserID  int32
	AddAuth bool
	Body    interface{}
}

// NewFriendTestRequest создает тестовый HTTP запрос для тестов друзей (переименовано)
func NewFriendTestRequest(t *testing.T, config FriendTestRequestConfig) *http.Request {
	req, err := http.NewRequest(config.Method, config.URL, nil)
	assert.NoError(t, err)

	// Добавляем переменные пути для gorilla/mux
	if config.Vars != nil {
		req = mux.SetURLVars(req, config.Vars)
	}

	// Добавляем userID в контекст
	if config.AddAuth {
		ctx := context.WithValue(req.Context(), domain.UserIDKey, config.UserID)
		req = req.WithContext(ctx)
	}

	return req
}

func TestFriendHandler_SendFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().SendFriendRequest(gomock.Any(), &pb.SendFriendRequestRequest{
			ActionUserID: userID,
			TargetUserID: friendID,
		}).Return(&emptypb.Empty{}, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRequestSent, response.Message)
	})

	t.Run("Invalid userID in context", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid friendID in vars", func(t *testing.T) {
		userID := int32(1)
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/invalid",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().SendFriendRequest(gomock.Any(), &pb.SendFriendRequestRequest{
			ActionUserID: userID,
			TargetUserID: friendID,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestFriendHandler_AcceptFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().AcceptFriendRequest(gomock.Any(), &pb.UserIDsPair{
			UserID:   userID,
			FriendID: friendID,
		}).Return(&emptypb.Empty{}, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/accept",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRequestAccepted, response.Message)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		userID := int32(1)
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/invalid/accept",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().AcceptFriendRequest(gomock.Any(), &pb.UserIDsPair{
			UserID:   userID,
			FriendID: friendID,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/accept",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/accept",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.AcceptFriendRequest(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestFriendHandler_RejectFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().RejectFriendRequest(gomock.Any(), &pb.UserIDsPair{
			UserID:   userID,
			FriendID: friendID,
		}).Return(&emptypb.Empty{}, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/reject",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRequestRejected, response.Message)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		userID := int32(1)
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/invalid/reject",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().RejectFriendRequest(gomock.Any(), &pb.UserIDsPair{
			UserID:   userID,
			FriendID: friendID,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/reject",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/friends/2/reject",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.RejectFriendRequest(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestFriendHandler_RemoveFriend(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().RemoveFriend(gomock.Any(), &pb.UserIDsPair{
			UserID:   userID,
			FriendID: friendID,
		}).Return(&emptypb.Empty{}, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodDelete,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendRemoved, response.Message)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		userID := int32(1)
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodDelete,
			URL:     "/friends/invalid",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().RemoveFriend(gomock.Any(), &pb.UserIDsPair{
			UserID:   userID,
			FriendID: friendID,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodDelete,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodDelete,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.RemoveFriend(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestFriendHandler_GetFriendshipStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success with status", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		statusResp := &pb.FriendshipStatusResponse{Status: "accepted"}
		mockClient.EXPECT().GetFriendshipStatus(gomock.Any(), &pb.GetFriendshipStatusRequest{
			UserID:   userID,
			FriendID: friendID,
		}).Return(statusResp, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/status",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var res FriendshipStatusResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, domain.FriendshipAccepted, res.Status)
	})

	t.Run("Invalid friendID", func(t *testing.T) {
		userID := int32(1)
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/invalid/status",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().GetFriendshipStatus(gomock.Any(), &pb.GetFriendshipStatusRequest{
			UserID:   userID,
			FriendID: friendID,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/status",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/status",
			Vars:    map[string]string{"id": "2"},
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetFriendshipStatus(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestFriendHandler_GetFriends(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{
			{UserID: 2},
			{UserID: 3},
		}}
		mockClient.EXPECT().GetFriends(gomock.Any(), &pb.GetFriendsRequest{
			UserID: userID,
			Limit:  20,
			Page:   1,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Success with pagination", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{{UserID: 2}}}
		mockClient.EXPECT().GetFriends(gomock.Any(), &pb.GetFriendsRequest{
			UserID: userID,
			Limit:  10,
			Page:   2,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends?limit=10&page=2",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		mockClient.EXPECT().GetFriends(gomock.Any(), &pb.GetFriendsRequest{
			UserID: userID,
			Limit:  20,
			Page:   1,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends",
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid query parameters", func(t *testing.T) {
		userID := int32(1)
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends?limit=invalid",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestFriendHandler_GetFriendRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{
			{UserID: 2},
		}}
		mockClient.EXPECT().GetFriendRequests(gomock.Any(), &pb.GetFriendRequestsRequest{
			UserID: userID,
			Limit:  20,
			Page:   1,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/requests",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendRequests(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		mockClient.EXPECT().GetFriendRequests(gomock.Any(), &pb.GetFriendRequestsRequest{
			UserID: userID,
			Limit:  20,
			Page:   1,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/requests",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriendRequests(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/requests",
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetFriendRequests(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestFriendHandler_GetSentRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{
			{UserID: 2},
		}}
		mockClient.EXPECT().GetSentRequests(gomock.Any(), &pb.GetSentRequestsRequest{
			UserID: userID,
			Limit:  20,
			Page:   1,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/sent",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetSentRequests(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		mockClient.EXPECT().GetSentRequests(gomock.Any(), &pb.GetSentRequestsRequest{
			UserID: userID,
			Limit:  20,
			Page:   1,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/sent",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetSentRequests(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/sent",
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetSentRequests(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestFriendHandler_GetAllUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{
			{UserID: 2},
			{UserID: 3},
		}}
		mockClient.EXPECT().GetAllUsers(gomock.Any(), &pb.GetAllUsersRequest{
			UserID: userID,
			Limit:  20,
			Page:   1,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/users/all",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		mockClient.EXPECT().GetAllUsers(gomock.Any(), &pb.GetAllUsersRequest{
			UserID: userID,
			Limit:  20,
			Page:   1,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/users/all",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/users/all",
			UserID:  0,
			AddAuth: false,
		})
		w := httptest.NewRecorder()
		handler.GetAllUsers(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestFriendHandler_CountUserRelations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(2)
		counts := &pb.UserRelationsCountsResponse{
			Accepted: 5,
			Pending:  3,
			Sent:     2,
			Blocked:  1,
		}
		mockClient.EXPECT().CountUserRelations(gomock.Any(), &pb.CountUserRelationsRequest{
			UserID: userID,
		}).Return(counts, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count",
			Vars:    map[string]string{"id": "2"},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var res domain.UserRelationsCounts
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, int32(5), res.Accepted)
		assert.Equal(t, int32(3), res.Pending)
		assert.Equal(t, int32(2), res.Sent)
		assert.Equal(t, int32(1), res.Blocked)
	})

	t.Run("Invalid userID", func(t *testing.T) {
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/invalid/count",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(2)
		mockClient.EXPECT().CountUserRelations(gomock.Any(), &pb.CountUserRelationsRequest{
			UserID: userID,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/2/count",
			Vars:    map[string]string{"id": "2"},
			UserID:  1,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.CountUserRelations(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestFriendHandler_SearchProfilesByFullName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{
			{UserID: 2},
		}}
		mockClient.EXPECT().SearchShortProfilesByFullNameAndRelationType(gomock.Any(), &pb.SearchProfilesRequest{
			FullName: "John",
			UserID:   userID,
			Type:     "accepted",
			Limit:    20,
			Page:     1,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/search?full_name=John&type=accepted",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SearchProfilesByFullName(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Missing full_name parameter", func(t *testing.T) {
		userID := int32(1)
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/search",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SearchProfilesByFullName(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := int32(1)
		mockClient.EXPECT().SearchShortProfilesByFullNameAndRelationType(gomock.Any(), &pb.SearchProfilesRequest{
			FullName: "John",
			UserID:   userID,
			Type:     "accepted",
			Limit:    20,
			Page:     1,
		}).Return(nil, status.Error(codes.Internal, "internal error"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/search?full_name=John&type=accepted",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SearchProfilesByFullName(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Default type parameter", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{{UserID: 2}}}
		// Должен использовать CountAccepted по умолчанию
		mockClient.EXPECT().SearchShortProfilesByFullNameAndRelationType(gomock.Any(), &pb.SearchProfilesRequest{
			FullName: "John",
			UserID:   userID,
			Type:     "accepted",
			Limit:    20,
			Page:     1,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/search?full_name=John",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SearchProfilesByFullName(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid query parameters", func(t *testing.T) {
		userID := int32(1)
		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends/search?full_name=John&limit=invalid",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SearchProfilesByFullName(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Edge case тесты для проверки граничных условий
func TestFriendHandler_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("Large page number", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{}}
		mockClient.EXPECT().GetFriends(gomock.Any(), &pb.GetFriendsRequest{
			UserID: userID,
			Limit:  10,
			Page:   100,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends?limit=10&page=100",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Zero limit uses default", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{}}
		mockClient.EXPECT().GetFriends(gomock.Any(), &pb.GetFriendsRequest{
			UserID: userID,
			Limit:  20, // Должен использовать дефолтное значение
			Page:   1,
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends?limit=0",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Negative page uses default", func(t *testing.T) {
		userID := int32(1)
		profiles := &pb.ShortProfileList{Profiles: []*pb.ShortProfile{}}
		mockClient.EXPECT().GetFriends(gomock.Any(), &pb.GetFriendsRequest{
			UserID: userID,
			Limit:  20,
			Page:   1, // Должен использовать дефолтное значение
		}).Return(profiles, nil)

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/friends?page=-1",
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.GetFriends(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Тесты для проверки различных gRPC ошибок
func TestFriendHandler_GrpcErrorHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service_mocks.NewMockFriendServiceClient(ctrl)
	handler := &FriendHandler{friendService: mockClient}

	t.Run("NotFound error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(999)
		mockClient.EXPECT().SendFriendRequest(gomock.Any(), &pb.SendFriendRequestRequest{
			ActionUserID: userID,
			TargetUserID: friendID,
		}).Return(nil, status.Error(codes.NotFound, "user not found"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/999",
			Vars:    map[string]string{"id": "999"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("AlreadyExists error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().SendFriendRequest(gomock.Any(), &pb.SendFriendRequestRequest{
			ActionUserID: userID,
			TargetUserID: friendID,
		}).Return(nil, status.Error(codes.AlreadyExists, "already friends"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("InvalidArgument error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(1) // попытка добавить самого себя
		mockClient.EXPECT().SendFriendRequest(gomock.Any(), &pb.SendFriendRequestRequest{
			ActionUserID: userID,
			TargetUserID: friendID,
		}).Return(nil, status.Error(codes.InvalidArgument, "cannot add yourself"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/1",
			Vars:    map[string]string{"id": "1"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("PermissionDenied error", func(t *testing.T) {
		userID := int32(1)
		friendID := int32(2)
		mockClient.EXPECT().SendFriendRequest(gomock.Any(), &pb.SendFriendRequestRequest{
			ActionUserID: userID,
			TargetUserID: friendID,
		}).Return(nil, status.Error(codes.PermissionDenied, "blocked user"))

		req := NewFriendTestRequest(t, FriendTestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/friends/2",
			Vars:    map[string]string{"id": "2"},
			UserID:  userID,
			AddAuth: true,
		})
		w := httptest.NewRecorder()
		handler.SendFriendRequest(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
