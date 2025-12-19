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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contextWithUserID(ctx context.Context, userID int32) context.Context {
	return context.WithValue(ctx, domain.UserIDKey, userID)
}

type MultipartData struct {
	Fields map[string]string // Текстовые поля
	Files  map[string][]byte // имя_файла -> содержимое
}

func newRequestWithVarsAndCtx(method, url string, vars map[string]string, userID int32, body interface{}, t *testing.T) *http.Request {
	var req *http.Request

	switch v := body.(type) {
	case nil:
		req = httptest.NewRequest(method, url, nil)

	case *MultipartData:
		// Создаем multipart тело
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)

		// Добавляем текстовые поля
		for key, value := range v.Fields {
			writer.WriteField(key, value)
		}

		// Добавляем файлы
		for filename, content := range v.Files {
			part, _ := writer.CreateFormFile("file", filename)
			part.Write(content)
		}

		writer.Close()

		req = httptest.NewRequest(method, url, bodyBuf)
		// ВАЖНО: Установить правильный Content-Type!
		req.Header.Set("Content-Type", writer.FormDataContentType())

	default: // Для JSON
		req = httptest.NewRequest(method, url, JSONReader(t, body))
		req.Header.Set("Content-Type", "application/json")
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
func MulpipartBody(formData map[string]string) *bytes.Buffer {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, value := range formData {
		writer.WriteField(key, value)
	}

	writer.Close()
	return body
}

func TestChatHandler_GetOrCreateChatWithUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatService := mocks.NewMockChatService(ctrl)
	handler := &ChatHandler{chatService: mockChatService}

	t.Run("Success test", func(t *testing.T) {
		selfUserID := int32(1)
		otherUserID := int32(2)
		chatID := int32(10)
		mockChatService.EXPECT().GetOrCreateChatWithUser(gomock.Any(), selfUserID, otherUserID).Return(chatID, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats/user/2", map[string]string{"id": "2"}, selfUserID, nil, t)
		w := httptest.NewRecorder()
		handler.GetOrCreateChatWithUser(w, req)

		var res domain.ChatIDResponse
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, chatID, res.ChatID)
	})

	t.Run("Invalid userID", func(t *testing.T) {
		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats/user/invalid", map[string]string{"id": "invalid"}, int32(0), nil, t)
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
		chatID := int32(1)
		userID := int32(2)
		messagesResp := domain.MessagesWithAuthors{
			Messages: []domain.Message{{ID: 1, Text: "hello"}},
			Authors:  map[int32]domain.ShortProfile{2: {UserID: 2, FullName: "user2"}},
		}
		mockChatService.EXPECT().GetMessagesByChatId(gomock.Any(), gomock.Any(), userID, chatID).Return(&messagesResp, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats/1/messages", map[string]string{"id": "1"}, userID, nil, t)
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
		chatID := int32(1)
		userID := int32(2)
		messageID := int32(10)
		text := "Hello world"
		mockChatService.EXPECT().CreateMessage(gomock.Any(), userID, chatID, text, nil, nil).Return(messageID, nil)
		body := &MultipartData{
			Fields: map[string]string{
				"text": text,
			},
			Files: map[string][]byte{},
		}
		req := newRequestWithVarsAndCtx(http.MethodPost, "/chats/1/message", map[string]string{"id": "1"}, userID, body, t)
		w := httptest.NewRecorder()
		handler.CreateMessage(w, req)

		var res domain.MessageIDResponse
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
		userID := int32(1)
		chatName := "Chat1"
		chats := []domain.FullChat{{ID: 1, Name: &chatName}}
		mockChatService.EXPECT().GetUserChats(gomock.Any(), userID, gomock.Any()).Return(chats, nil)

		req := newRequestWithVarsAndCtx(http.MethodGet, "/chats?limit=10&page=1", nil, userID, nil, t)
		w := httptest.NewRecorder()
		handler.GetUserChats(w, req)

		var res []domain.FullChat
		err := json.NewDecoder(w.Body).Decode(&res)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, chats, res)
	})
}

func TestChatHandler_UpdateLastReadMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatService := mocks.NewMockChatService(ctrl)
	handler := &ChatHandler{chatService: mockChatService}

	t.Run("Success", func(t *testing.T) {
		userID := int32(1)
		chatID := int32(10)
		lastReadMessageID := int32(100)

		mockChatService.EXPECT().UpdateLastReadMessage(gomock.Any(), userID, chatID, lastReadMessageID).Return(nil)

		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/chats/10/last-read",
			Vars:    map[string]string{"id": "10"},
			UserID:  userID,
			Body:    map[string]interface{}{"lastReadMessageID": lastReadMessageID},
			AddAuth: true,
		})

		w := httptest.NewRecorder()
		handler.UpdateLastReadMessage(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response domain.JSONResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Last read message updated", response.Message)
	})

	t.Run("Invalid chat ID", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/chats/invalid/last-read",
			Vars:    map[string]string{"id": "invalid"},
			UserID:  1,
			Body:    map[string]interface{}{"lastReadMessageID": 100},
			AddAuth: true,
		})

		w := httptest.NewRecorder()
		handler.UpdateLastReadMessage(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid request body", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPut,
			URL:     "/chats/10/last-read",
			Vars:    map[string]string{"id": "10"},
			UserID:  1,
			Body:    "invalid json",
			AddAuth: true,
		})

		w := httptest.NewRecorder()
		handler.UpdateLastReadMessage(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestChatHandler_GetUserChats_InvalidParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatService := mocks.NewMockChatService(ctrl)
	handler := &ChatHandler{chatService: mockChatService}

	t.Run("Invalid query params", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodGet,
			URL:     "/chats?limit=invalid&page=invalid",
			UserID:  1,
			AddAuth: true,
		})

		w := httptest.NewRecorder()
		handler.GetUserChats(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestChatHandler_CreateMessage_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatService := mocks.NewMockChatService(ctrl)
	handler := &ChatHandler{chatService: mockChatService}

	t.Run("Invalid message body", func(t *testing.T) {
		req := NewTestRequest(t, TestRequestConfig{
			Method:  http.MethodPost,
			URL:     "/chats/10/message",
			Vars:    map[string]string{"id": "10"},
			UserID:  1,
			Body:    "invalid json",
			AddAuth: true,
		})

		w := httptest.NewRecorder()
		handler.CreateMessage(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
