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
	wsHub        domain.WSHub
}

func NewChatHandler(userStore domain.UserStore, profileStore domain.ProfileStore, chatStore domain.ChatStore, messageStore domain.MessageStore, wsHub domain.WSHub) *ChatHandler {
	return &ChatHandler{
		userStore:    userStore,
		profileStore: profileStore,
		chatStore:    chatStore,
		messageStore: messageStore,
		wsHub:        wsHub,
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
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		service.Error(r.Context(), "Failed to parse user ID", err)
		return
	}

	isUserExist, err := api.userStore.IsUserExists(r.Context(), userID)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to check user existence", err)
		return
	}
	if !isUserExist {
		sendJSONResponse(w, "User doesn't exist", http.StatusBadRequest)
		service.Warn(r.Context(), "User not found")
		return
	}
	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int)
	if userID == selfUserID {
		sendJSONResponse(w, "Cant create chat with yourself", http.StatusBadRequest)
		service.Warn(r.Context(), "Failed to create or get chat with same self user")
		return
	}
	chatID, err := api.chatStore.GetOrCreateChatWithUser(r.Context(), selfUserID, userID)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
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
		sendJSONResponse(w, "Invalid chat ID", http.StatusBadRequest)
		service.Error(r.Context(), "Failed to parse chat ID", err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	var qParams PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", service.StructName(qParams)))
		return
	}

	isMember, err := api.chatStore.IsMemberOfChat(r.Context(), userID, chatID)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to check membership", err, zap.Int("chatID", chatID))
		return
	}
	if !isMember {
		sendJSONResponse(w, domain.Forbidden, http.StatusForbidden)
		service.Warn(r.Context(), "User not a member of chat", zap.Int("chatID", chatID))
		return
	}

	messages, err := api.messageStore.GetMessagesByChatId(r.Context(), chatID, qParams.Limit, qParams.Offset)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
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

	authors, err := api.profileStore.GetShortProfileByUserIDs(r.Context(), authorIDs)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
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
	service.Info(r.Context(), "Messages retrieved successfully", zap.Int("chatID", chatID), zap.Int("limit", qParams.Limit), zap.Int("offset", qParams.Offset))
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
	exits, err := api.chatStore.IsChatExist(r.Context(), chatID)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusBadRequest)
		service.Warn(r.Context(), "Failed to get chat", zap.Int("chatID", chatID))
		return
	}

	if !exits {
		sendJSONResponse(w, "Chat does not exist", http.StatusBadRequest)
		service.Warn(r.Context(), "Chat not found", zap.Int("chatID", chatID))
		return
	}
	var message domain.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", service.StructName(message)))
		return
	}
	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	message.AuthorID = userID
	message.ChatID = chatID
	messageID, err := api.messageStore.CreateMessage(r.Context(), message)
	if err != nil {
		sendJSONResponse(w, "Internal server error", http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to create message", err, zap.Int("chatID", chatID))
		return
	}
	if err := json.NewEncoder(w).Encode(MessageIDResponse{MessageID: messageID}); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", service.StructName(MessageIDResponse{})))
		return
	}
	chat, err := api.chatStore.GetFullChatByIDAndSenderID(r.Context(), userID, chatID)
	if err != nil {
		service.Error(r.Context(), "Fail to get chat", err)
		return
	}
	recipients, err := api.chatStore.GetOtherChatMembersIdByAuthorId(r.Context(), userID, chatID)
	if err != nil {
		service.Error(r.Context(), "Fail to get recipients", err)
		return
	}
	data, err := json.Marshal(chat)
	if err != nil {
		service.Error(r.Context(), "Fail to marshal chat", err)
		return
	}
	response := domain.Envelope{
		Type: "new_message",
		Data: data,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		service.Error(r.Context(), "Fail to marshal chat", err)
		return
	}

	for _, recipient := range recipients {
		api.wsHub.SendToUser(recipient, jsonResponse)
	}
	service.Info(r.Context(), "Message created successfully", zap.Int("messageID", messageID), zap.Int("chatID", chatID))
}

// GetUserChats получает список чатов для текущего пользователя, включая
// последнее сообщение и его автора.
//
// @Summary Получение чатов пользователя
// @Description Возвращает постраничный список чатов для аутентифицированного пользователя.
// Каждый чат содержит его ID, имя, аватар, тип (групповой/приватный), последнее сообщение и автора последнего сообщения.
// @Tags chats
// @Accept json
// @Produce json
// @Param limit query int false "Number of chats to return" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} domain.FullChat "List of chats"
// @Failure 400 {object} JSONResponse "Invalid query parameters"
// @Failure 500 {object} JSONResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /chats [get]
func (api *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	var qParams PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		service.Error(r.Context(), domain.InvalidJSON, err, zap.String("struct", service.StructName(qParams)))
		return
	}

	chats, err := api.chatStore.GetUserFullChats(r.Context(), userID, qParams.Limit, qParams.Offset)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to get chats", err)
		return
	}

	if err := json.NewEncoder(w).Encode(chats); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", service.StructName(chats)))
	}
	service.Info(r.Context(), "Chats retrieved successfully", zap.Int("limit", qParams.Limit), zap.Int("offset", qParams.Offset))
}
