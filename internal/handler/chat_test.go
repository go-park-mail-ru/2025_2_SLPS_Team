package handler

import (
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
)

func contextWithUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, domain.UserIDKey, userID)
}

func newRequestWithVarsAndCtx(method, url string, vars map[string]string, userID int, body interface{}, t *testing.T) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, JSONReader(t, body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	if vars != nil {
		req = mux.SetURLVars(req, vars)
	}
	if userID != 0 {
		ctx := contextWithUserID(req.Context(), userID)
		req = req.WithContext(ctx)
	}
	return req
}

func TestChatHandler_GetOrCreateChatWithUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatService := mocks.NewMockChatService(ctrl)
	handler := &ChatHandler{chatService: mockChatService}

	t.Run("Success test", func(t *testing.T) {
		selfUserID := 1
		otherUserID := 2
		chatID := 10
		mockChatService.EXPECT().GetOrCreateChatWithUser(gomock.Any(), selfUserID, otherUserID).Return(chatID, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats/user/2", map[string]string{"id": strconv.Itoa(otherUserID)}, selfUserID, nil, t)
		w := httptest.NewRecorder()
		handler.GetOrCreateChatWithUser(w, req)

		var res ChatIDResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, chatID, res.ChatID)
	})

	t.Run("Invalid userID", func(t *testing.T) {
		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats/user/invalid", map[string]string{"id": "invalid"}, 0, nil, t)
		w := httptest.NewRecorder()
		handler.GetOrCreateChatWithUser(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}

func TestChatHandler_GetMessagesByChatId(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatService := mocks.NewMockChatService(ctrl)
	handler := &ChatHandler{chatService: mockChatService}

	t.Run("Success test", func(t *testing.T) {
		chatID := 1
		userID := 2
		messagesResp := domain.MessagesWithAuthors{
			Messages: []domain.Message{{ID: 1, Text: "hello"}},
			Authors:  map[int]domain.ShortProfile{2: {UserID: 2, FullName: "user2"}},
		}
		mockChatService.EXPECT().GetMessagesByChatId(gomock.Any(), gomock.Any(), userID, chatID).Return(&messagesResp, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats/1/messages", map[string]string{"id": strconv.Itoa(chatID)}, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetMessagesByChatId(w, req)

		var res domain.MessagesWithAuthors
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, messagesResp, res)
	})

	t.Run("Invalid chatID", func(t *testing.T) {
		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats/invalid/messages", map[string]string{"id": "invalid"}, 0, nil, t)
		w := httptest.NewRecorder()
		handler.GetMessagesByChatId(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}

func TestChatHandler_CreateMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatService := mocks.NewMockChatService(ctrl)
	handler := &ChatHandler{chatService: mockChatService}

	t.Run("Success test", func(t *testing.T) {
		chatID := 1
		userID := 2
		message := domain.Message{Text: "Hello"}
		messageID := 10
		mockChatService.EXPECT().CreateMessage(gomock.Any(), userID, chatID, message).Return(messageID, nil)

		req := newRequestWithVarsAndCtx(http.MethodPost, "/chats/1/message", map[string]string{"id": strconv.Itoa(chatID)}, userID, message, t)
		w := httptest.NewRecorder()
		handler.CreateMessage(w, req)

		var res MessageIDResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, messageID, res.MessageID)
	})

	t.Run("Invalid chatID", func(t *testing.T) {
		req := newRequestWithVarsAndCtx(http.MethodPost, "/chats/invalid/message", map[string]string{"id": "invalid"}, 0, nil, t)
		w := httptest.NewRecorder()
		handler.CreateMessage(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}

func TestChatHandler_GetUserChats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatService := mocks.NewMockChatService(ctrl)
	handler := &ChatHandler{chatService: mockChatService}

	t.Run("Success test", func(t *testing.T) {
		userID := 1
		chats := []domain.FullChat{{ID: 1, Name: "Chat1"}}
		mockChatService.EXPECT().GetUserChats(gomock.Any(), userID, gomock.Any()).Return(chats, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats?limit=10&offset=0", nil, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetUserChats(w, req)

		var res []domain.FullChat
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, chats, res)
	})
}
