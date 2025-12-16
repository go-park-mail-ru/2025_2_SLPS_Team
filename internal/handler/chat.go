package handler

import (
	"net/http"
	"project/domain"
)

type ChatHandler struct {
	chatService domain.ChatService
}

func NewChatHandler(chatService domain.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
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
	userID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int32)

	chatID, err := api.chatService.GetOrCreateChatWithUser(r.Context(), selfUserID, int32(userID))
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, domain.ChatIDResponse{ChatID: chatID})
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

	chatID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	messagesWithAuthors, err := api.chatService.GetMessagesByChatId(r.Context(), qParams, userID, int32(chatID))
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, messagesWithAuthors)
}

// CreateMessage создает новое сообщение в чате
// @Summary Создать сообщение (текст, файлы или стикер)
// @Description Создает новое сообщение в указанном чате. Можно отправить:
// 1. Текст (можно с вложениями)
// 2. Только вложения (без текста)
// 3. Только стикер (без текста и вложений)
// НЕЛЬЗЯ: текст со стикером или вложения со стикером
// @Tags messages
// @Accept multipart/form-data
// @Produce json
// @Param id path int32 true "ID чата"
// @Param text formData string false "Текст сообщения (обязателен, если нет вложений и стикера)"
// @Param attachments formData []file false "Вложения к сообщению" collectionFormat(multi)
// @Param sticker_id formData int32 false "ID стикера (если отправляется стикер, то нельзя отправлять текст и вложения)"
// @Success 200 {object} MessageIDResponse "Сообщение успешно создано"
// @Failure 400 {object} JSONResponse "Неверные данные запроса"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 403 {object} JSONResponse "Доступ запрещен (не участник чата)"
// @Failure 404 {object} JSONResponse "Чат или стикер не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /chats/{id}/message [post]
func (api *ChatHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	chatID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	// Парсим multipart форму (максимум 50MB)
	err = r.ParseMultipartForm(50 << 20)
	if err != nil {
		sendJSONResponse(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	err = ParseMultipart(r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	text := r.FormValue("text")
	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	stickerID, err := parseIntParam(r, "sticker_id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	attachmentFiles, err := domain.MultipartFiles(r, "attachments")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	// Создаем сообщение
	messageID, err := api.chatService.CreateMessage(r.Context(), userID, int32(chatID), text, attachmentFiles, stickerID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, domain.MessageIDResponse{MessageID: messageID})
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
	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	chats, err := api.chatService.GetUserChats(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, chats)
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
	chatID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	req, err := DecodeJSONBody[domain.UpdateLastReadRequest](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	err = api.chatService.UpdateLastReadMessage(r.Context(), userID, chatID, req.LastReadMessageID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "Last read message updated")
}
