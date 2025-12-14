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

	if err := sendJSONData(r.Context(), w, ChatIDResponse{ChatID: chatID}); err != nil {
		return
	}
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
		return
	}

	if err := sendJSONData(r.Context(), w, messagesWithAuthors); err != nil {
		return
	}
}

type MessageIDResponse struct {
	MessageID int32 `json:"messageID"`
}

// CreateMessage создает новое сообщение в чате
// @Summary Создать сообщение (текст + файлы)
// @Description Создает новое сообщение в указанном чате с возможностью прикрепления файлов
// @Tags messages
// @Accept multipart/form-data
// @Produce json
// @Param id path int32 true "ID чата"
// @Param text formData string true "Текст сообщения"
// @Param attachments formData []file false "Вложения к сообщению" collectionFormat(multi)
// @Success 200 {object} MessageIDResponse "Сообщение успешно создано"
// @Failure 400 {object} JSONResponse "Неверные данные запроса"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 403 {object} JSONResponse "Доступ запрещен (не участник чата)"
// @Failure 404 {object} JSONResponse "Чат не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /chats/{id}/message [post]
func (api *ChatHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatIDStr := vars["id"]
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		sendJSONResponse(w, "Invalid chat ID", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse chat ID", zap.Error(err))
		return
	}

	// Парсим multipart форму (максимум 50MB)
	err = r.ParseMultipartForm(50 << 20)
	if err != nil {
		sendJSONResponse(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	text := r.FormValue("text")
	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	// Обрабатываем вложения
	var attachmentFiles []*domain.File
	if attachments, ok := r.MultipartForm.File["attachments"]; ok {
		attachmentFiles, err = domain.MultipartListToFiles(attachments)
		if err != nil {
			sendJSONResponse(w, "Can't parse multipart form to files", http.StatusBadRequest)
			domain.Error(r.Context(), "Failed to parse multipart form to files", err)
			return
		}
	}

	// Создаем сообщение
	messageID, err := api.chatService.CreateMessage(r.Context(), userID, int32(chatID), text, attachmentFiles)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, MessageIDResponse{MessageID: messageID}); err != nil {
		return
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
		return
	}

	if err := sendJSONData(r.Context(), w, chats); err != nil {
		return
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
		sendJSONResponse(w, "Invalid chat ID", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse chat ID", zap.Error(err))
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
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Last read message updated", http.StatusOK)
}
