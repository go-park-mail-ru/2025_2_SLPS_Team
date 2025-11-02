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
	chatService service.ChatService
}

func NewChatHandler(chatService service.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
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
		domain.FromContext(r.Context()).Error("Failed to parse user ID", zap.Error(err))
		return
	}
	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int)

	chatID, err := api.chatService.GetOrCreateChatWithUser(r.Context(), selfUserID, userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ChatIDResponse{ChatID: chatID}); err != nil {
		domain.FromContext(r.Context()).Error(domain.FailToEncode, zap.Error(err), zap.String("struct", domain.StructName(ChatIDResponse{})))
	}

	domain.FromContext(r.Context()).Info("Chat created or retrieved", zap.Int("chatID", chatID), zap.Int("chatWithUserID", userID))
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
		domain.FromContext(r.Context()).Error("Failed to parse chat ID", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(qParams)))
		return
	}

	messagesWithAuthors, err := api.chatService.GetMessagesByChatId(r.Context(), qParams, userID, chatID)
	if err != nil {
		sendJSONError(w, err)
	}

	if err := json.NewEncoder(w).Encode(messagesWithAuthors); err != nil {
		domain.FromContext(r.Context()).Error(domain.FailToEncode, zap.Error(err), zap.String("struct", domain.StructName(messagesWithAuthors)))
		return
	}

	domain.FromContext(r.Context()).Info("Messages retrieved successfully", zap.Int("chatID", chatID), zap.Int("limit", qParams.Limit), zap.Int("offset", qParams.Offset))
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
		domain.FromContext(r.Context()).Error("Failed to parse chatID", zap.Error(err))
		return
	}

	var message domain.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(message)))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	messageID, err := api.chatService.CreateMessage(r.Context(), userID, chatID, message)
	if err != nil {
		sendJSONError(w, err)
	}

	err = sendJSONData(r.Context(), w, MessageIDResponse{MessageID: messageID})
	if err == nil {
		domain.FromContext(r.Context()).Info("Message created successfully", zap.Int("messageID", messageID), zap.Int("chatID", chatID))
	}
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

	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(qParams)))
		return
	}

	chats, err := api.chatService.GetUserChats(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
	}

	err = sendJSONData(r.Context(), w, chats)
	if err == nil {
		domain.FromContext(r.Context()).Info("Chats retrieved successfully", zap.Int("limit", qParams.Limit), zap.Int("offset", qParams.Offset))
	}
}
