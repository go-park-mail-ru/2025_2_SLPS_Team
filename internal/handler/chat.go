package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"project/domain"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
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
		return
	}

	isUserExist, err := api.userStore.IsUserExists(userID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !isUserExist {
		sendJSONSuccess(w, "User doesn't exist", http.StatusBadRequest)
		return
	}
	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int)
	chatID, err := api.chatStore.GetOrCreateChatWithUser(selfUserID, userID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ChatIDResponse{ChatID: chatID}); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
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
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	var qParams PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONError(w, "Invalid query parameters", http.StatusBadRequest)
		return
	}

	isMember, err := api.chatStore.IsMemberOfChat(userID, chatID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		sendJSONSuccess(w, "Forbidden", http.StatusForbidden)
		return
	}

	messages, err := api.messageStore.GetMessagesByChatId(chatID, qParams.Limit, qParams.Offset)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
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
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := MessagesWithAuthorsResp{
		Messages: messages,
		Authors:  authors,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
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
		return
	}
	exits, err := api.chatStore.IsChatExist(chatID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !exits {
		sendJSONSuccess(w, "Chat does not exist", http.StatusBadRequest)
		return
	}
	var message domain.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	message.AuthorID = userID
	message.ChatID = chatID
	messageID, err := api.messageStore.CreateMessage(message)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(MessageIDResponse{MessageID: messageID}); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
