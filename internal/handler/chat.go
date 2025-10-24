package handler

import (
	"encoding/json"
	"net/http"
	"project/domain"
	"project/internal/service"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type ChatHandler struct {
	userStore    domain.UserStore
	profileStore domain.ProfileStore
	chatStore    domain.ChatStore
	messageStore domain.MessageStore
}

func NewChatHandler(userStore domain.UserStore, profileStore domain.ProfileStore, chatStore domain.ChatStore, messageStore domain.MessageStore) *ChatHandler {
	return &ChatHandler{
		userStore:    userStore,
		profileStore: profileStore,
		chatStore:    chatStore,
		messageStore: messageStore,
	}
}

type ChatIDResponse struct {
	ChatID int `json:"chatID"`
}

// GetOrCreateChatWithUser получает или создает чат с указанным пользователем
// @Summary Получить или создать чат с пользователем
// @Description Возвращает ID чата между текущим пользователем и указанным userID, создавая чат при отсутствии
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} ChatIDResponse
// @Failure 400 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /chats/user/{id} [get]
func (api *ChatHandler) GetOrCreateChatWithUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		sendJSONSuccess(w, "Invalid user ID", http.StatusBadRequest)
		service.Error(r.Context(), "Failed to parse user ID", err)
		return
	}

	isUserExist, err := api.userStore.IsUserExists(userID)
	if err != nil {
		sendJSONSuccess(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to check user existence", err)
		return
	}
	if !isUserExist {
		sendJSONSuccess(w, "User doesn't exist", http.StatusBadRequest)
		service.Warn(r.Context(), "User not found")
		return
	}
	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int)
	if userID == selfUserID {
		sendJSONSuccess(w, "Cant create chat with yourself", http.StatusBadRequest)
		service.Warn(r.Context(), "Failed to create or get chat with same self user")
		return
	}
	chatID, err := api.chatStore.GetOrCreateChatWithUser(selfUserID, userID)
	if err != nil {
		sendJSONSuccess(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to create or get chat with user", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ChatIDResponse{ChatID: chatID}); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", service.StructName(ChatIDResponse{})))
	}

	service.Info(r.Context(), "Chat created or retrieved", zap.Int("chatID", chatID), zap.Int("chatWithUserID", userID))
}

// стоит это вынести на уровень домена
type PaginateQueryParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
type MessagesWithAuthorsResp struct {
	Messages []domain.Message      `json:"messages"`
	Authors  []domain.ShortProfile `json:"authors"`
}

// GetMessagesByChatId возвращает список сообщений и краткие профили авторов в чате
// @Summary Получить сообщения чата с авторами
// @Description Возвращает сообщения из чата с пагинацией и краткую информацию об авторах
// @Tags messages
// @Produce json
// @Param id path int true "ID чата"
// @Param limit query int false "Лимит количества сообщений" default(20)
// @Param offset query int false "Смещение для пагинации" default(0)
// @Success 200 {object} handler.MessagesWithAuthorsResp
// @Failure 400 {object} JSONResponse
// @Failure 403 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /chats/{id}/messages [get]
func (api *ChatHandler) GetMessagesByChatId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatIDStr := vars["id"]
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		sendJSONSuccess(w, "Invalid chat ID", http.StatusBadRequest)
		service.Error(r.Context(), "Failed to parse chat ID", err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	var qParams PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONError(w, domain.InvalidParams, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", service.StructName(qParams)))
		return
	}

	isMember, err := api.chatStore.IsMemberOfChat(userID, chatID)
	if err != nil {
		sendJSONSuccess(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to check membership", err, zap.Int("chatID", chatID))
		return
	}
	if !isMember {
		sendJSONSuccess(w, domain.Forbidden, http.StatusForbidden)
		service.Warn(r.Context(), "User not a member of chat", zap.Int("chatID", chatID))
		return
	}

	messages, err := api.messageStore.GetMessagesByChatId(chatID, qParams.Limit, qParams.Offset)
	if err != nil {
		sendJSONSuccess(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to get messages", err, zap.Int("chatID", chatID))
		return
	}

	mapIDs := make(map[int]struct{})
	for _, msg := range messages {
		mapIDs[msg.AuthorID] = struct{}{}
	}
	authorIDs := make([]int, 0, len(mapIDs))
	for id := range mapIDs {
		authorIDs = append(authorIDs, id)
	}

	authors, err := api.profileStore.GetShortProfileByUserIDs(authorIDs)
	if err != nil {
		sendJSONSuccess(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to get authors", err, zap.Ints("authorIDs", authorIDs))
		return
	}

	response := MessagesWithAuthorsResp{
		Messages: messages,
		Authors:  authors,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", service.StructName(response)))
	}
	service.Info(r.Context(), "Messages retrieved successfully", zap.Int("chatID", chatID), zap.Int("messagesCount", len(messages)))
}

type MessageIDResponse struct {
	MessageID int `json:"messageID"`
}

// CreateMessage создает новое сообщение в чате
// @Summary Создать сообщение
// @Description Создает новое сообщение в указанном чате
// @Tags messages
// @Accept json
// @Produce json
// @Param chatID path int true "ID чата"
// @Param message body domain.Message true "Тело сообщения"
// @Success 200 {object} MessageIDResponse
// @Failure 400 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Router /chats/{chatID}/message [post]
func (api *ChatHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatIDStr := vars["id"]
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		http.Error(w, "invalid chatID", http.StatusBadRequest)
		service.Error(r.Context(), "Failed to parse chatID", err)
		return
	}
	exits, err := api.chatStore.IsChatExist(chatID)
	if err != nil {
		sendJSONSuccess(w, domain.ServerErr, http.StatusBadRequest)
		service.Warn(r.Context(), "Failed to get chat", zap.Int("chatID", chatID))
		return
	}

	if !exits {
		sendJSONSuccess(w, "Chat does not exist", http.StatusBadRequest)
		service.Warn(r.Context(), "Chat not found", zap.Int("chatID", chatID))
		return
	}
	var message domain.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		sendJSONError(w, domain.InvalidJSON, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", service.StructName(message)))
		return
	}
	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	message.AuthorID = userID
	message.ChatID = chatID
	messageID, err := api.messageStore.CreateMessage(message)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to create message", err, zap.Int("chatID", chatID))
		return
	}
	if err := json.NewEncoder(w).Encode(MessageIDResponse{MessageID: messageID}); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", service.StructName(MessageIDResponse{})))
	}
	service.Info(r.Context(), "Message created successfully", zap.Int("messageID", messageID), zap.Int("chatID", chatID))
}
