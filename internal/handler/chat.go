package handler

import (
	"encoding/json"
	"net/http"
	"project/domain"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type ChatHandler struct {
	chatService domain.ChatService
}

func NewChatHandler(chatService domain.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

type ChatIDResponse struct {
	ChatID int32 `json:"chatID"`
}

// GetOrCreateChatWithUser получает или создает чат с указанным пользователем
// @Summary Получить или создать чат с пользователем
// @Description Возвращает ID чата между текущим пользователем и указанным userID, создавая чат при отсутствии
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int32 true "ID пользователя"
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
	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int32)

	chatID, err := api.chatService.GetOrCreateChatWithUser(r.Context(), selfUserID, int32(userID))
	if err != nil {
		sendJSONError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ChatIDResponse{ChatID: chatID}); err != nil {
		domain.FromContext(r.Context()).Error(domain.FailToEncode, zap.Error(err), zap.String("struct", domain.StructName(ChatIDResponse{})))
	}

	domain.FromContext(r.Context()).Info("Chat created or retrieved", zap.Int32("chatID", chatID), zap.Int("chatWithUserID", userID))
}

// GetMessagesByChatId возвращает список сообщений и краткие профили авторов в чате
// @Summary Получить сообщения чата с авторами
// @Description Возвращает сообщения из чата с пагинацией и краткую информацию об авторах
// @Tags messages
// @Produce json
// @Param id path int32 true "ID чата"
// @Param limit query int32 false "Лимит количества сообщений" default(20)
// @Param page query int32 false "страница для пагинации" default(0)
// @Success 200 {object} domain.MessagesWithAuthors
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

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(qParams)))
		return
	}

	messagesWithAuthors, err := api.chatService.GetMessagesByChatId(r.Context(), qParams, userID, int32(chatID))
	if err != nil {
		sendJSONError(w, err)
	}

	if err := json.NewEncoder(w).Encode(messagesWithAuthors); err != nil {
		domain.FromContext(r.Context()).Error(domain.FailToEncode, zap.Error(err), zap.String("struct", domain.StructName(messagesWithAuthors)))
		return
	}

	domain.FromContext(r.Context()).Info("Messages retrieved successfully", zap.Int("chatID", chatID), zap.Int32("limit", qParams.Limit), zap.Int32("page", qParams.Page))
}

type MessageIDResponse struct {
	MessageID int32 `json:"messageID"`
}

// CreateMessage создает новое сообщение в чате
// @Summary Создать сообщение
// @Description Создает новое сообщение в указанном чате
// @Tags messages
// @Accept json
// @Produce json
// @Param chatID path int32 true "ID чата"
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
		sendJSONResponse(w, "invalid chatID", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse chatID", zap.Error(err))
		return
	}

	var message domain.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(message)))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	messageID, err := api.chatService.CreateMessage(r.Context(), userID, int32(chatID), message)
	if err != nil {
		sendJSONError(w, err)
	}

	err = sendJSONData(r.Context(), w, MessageIDResponse{MessageID: messageID})
	if err == nil {
		domain.FromContext(r.Context()).Info("Message created successfully", zap.Int32("messageID", messageID), zap.Int("chatID", chatID))
	}
}

// GetUserChats получает список чатов для текущего пользователя,
// включая последнее сообщение и его автора.
//
// @Summary Получить список чатов пользователя
// @Description Возвращает постраничный список чатов для текущего (аутентифицированного) пользователя.
// Каждый чат содержит свой ID, имя, аватар, тип (групповой или приватный),
// последнее сообщение и информацию об авторе этого сообщения.
// @Tags chats
// @Accept json
// @Produce json
// @Param limit query int32 false "Количество чатов для возврата" default(20)
// @Param page query int32 false "Номер страницы для пагинации" default(0)
// @Success 200 {array} domain.FullChat "Список чатов"
// @Failure 400 {object} JSONResponse "Некорректные параметры запроса"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /chats [get]
func (api *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

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
		domain.FromContext(r.Context()).Info("Chats retrieved successfully", zap.Int32("limit", qParams.Limit), zap.Int32("page", qParams.Page))
	}
}

type UpdateLastReadRequest struct {
	LastReadMessageID int32 `json:"lastReadMessageID"`
}

// UpdateLastReadMessage обновляет ID последнего прочитанного сообщения пользователя в чате.
//
// @Summary Обновить последнее прочитанное сообщение
// @Description Обновляет значение lastReadMessageID для текущего (аутентифицированного) пользователя в указанном чате.
// Обновление произойдёт только если новое значение больше текущего, чтобы предотвратить откат счётчика непрочитанных сообщений.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int32 true "ID чата"
// @Param body body UpdateLastReadRequest true "Новый ID последнего прочитанного сообщения"
// @Success 200 {object} JSONResponse "Информация об успешном обновлении или отсутствии изменений"
// @Failure 400 {object} JSONResponse "Некорректные параметры запроса"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Router /chats/{id}/last-read [put]
func (api *ChatHandler) UpdateLastReadMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatIDStr := vars["id"]
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		sendJSONResponse(w, "invalid chatID", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse chatID", zap.Error(err))
		return
	}

	var req UpdateLastReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(req)))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)
	err = api.chatService.UpdateLastReadMessage(r.Context(), userID, int32(chatID), req.LastReadMessageID)
	if err != nil {
		domain.FromContext(r.Context()).Error("Failed update last read message", zap.Error(err))
		return
	}

	domain.FromContext(r.Context()).Info("last read message updated")
	sendJSONResponse(w, "last read message updated", http.StatusOK)
}
