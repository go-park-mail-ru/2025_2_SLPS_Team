package handler

import (
	"mime/multipart"
	"net/http"
	"project/domain"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type CommunityHandler struct {
	communityService domain.CommunityService
}

func NewCommunityHandler(communityService domain.CommunityService) *CommunityHandler {
	return &CommunityHandler{
		communityService: communityService,
	}
}

// CreateCommunity создает новое сообщество
// @Summary Создать сообщество
// @Description Создает новое сообщество с возможностью загрузки аватара и обложки
// @Tags communities
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Название сообщества (3-48 символов)"
// @Param description formData string false "Описание сообщества (до 512 символов)"
// @Param avatar formData file false "Аватар сообщества"
// @Param cover formData file false "Обложка сообщества"
// @Success 201 {object} map[string]interface{} "Сообщество успешно создано"
// @Failure 400 {object} JSONResponse "Неверные данные запроса"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities [post]
func (h *CommunityHandler) CreateCommunity(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		sendJSONResponse(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	var avatarFile, coverFile *multipart.FileHeader
	if avatars, ok := r.MultipartForm.File["avatar"]; ok && len(avatars) > 0 {
		avatarFile = avatars[0]
	}
	if covers, ok := r.MultipartForm.File["cover"]; ok && len(covers) > 0 {
		coverFile = covers[0]
	}

	req := domain.CommunityRequest{
		Name:        name,
		Description: description,
	}

	community, err := h.communityService.CreateCommunity(r.Context(), userID, req, avatarFile, coverFile)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	// Используем sendJSONData для возврата структуры с сообщением
	response := map[string]interface{}{
		"message":   "Community created successfully",
		"community": community,
	}

	if err := sendJSONData(r.Context(), w, response); err != nil {
		return
	}
}

// UpdateCommunity обновляет сообщество
// @Summary Обновить сообщество
// @Description Обновляет информацию о сообществе (только создатель)
// @Tags communities
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID сообщества"
// @Param name formData string false "Название сообщества (3-48 символов)"
// @Param description formData string false "Описание сообщества (до 512 символов)"
// @Param avatar formData file false "Новый аватар сообщества"
// @Param cover formData file false "Новая обложка сообщества"
// @Success 200 {object} JSONResponse "Сообщество успешно обновлено"
// @Failure 400 {object} JSONResponse "Неверные данные запроса"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 403 {object} JSONResponse "Доступ запрещен (не создатель)"
// @Failure 404 {object} JSONResponse "Сообщество не найдено"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/{id} [put]
func (h *CommunityHandler) UpdateCommunity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid community ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid community ID", zap.String("communityID", vars["id"]))
		return
	}

	err = r.ParseMultipartForm(50 << 20)
	if err != nil {
		sendJSONResponse(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	var avatarFile, coverFile *multipart.FileHeader
	if avatars, ok := r.MultipartForm.File["avatar"]; ok && len(avatars) > 0 {
		avatarFile = avatars[0]
	}
	if covers, ok := r.MultipartForm.File["cover"]; ok && len(covers) > 0 {
		coverFile = covers[0]
	}

	req := domain.CommunityRequest{
		Name:        name,
		Description: description,
	}

	err = h.communityService.UpdateCommunity(r.Context(), communityID, userID, req, avatarFile, coverFile)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Community updated successfully", http.StatusOK)
}

// DeleteCommunity удаляет сообщество
// @Summary Удалить сообщество
// @Description Удаляет сообщество (только создатель)
// @Tags communities
// @Produce json
// @Param id path int true "ID сообщества"
// @Success 200 {object} JSONResponse "Сообщество успешно удалено"
// @Failure 400 {object} JSONResponse "Неверный ID сообщества"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 403 {object} JSONResponse "Доступ запрещен (не создатель)"
// @Failure 404 {object} JSONResponse "Сообщество не найдено"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/{id} [delete]
func (h *CommunityHandler) DeleteCommunity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid community ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid community ID", zap.String("communityID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.communityService.DeleteCommunity(r.Context(), communityID, userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Community deleted successfully", http.StatusOK)
}

// GetCommunity возвращает информацию о сообществе
// @Summary Получить информацию о сообществе
// @Description Возвращает информацию о сообществе включая количество подписчиков, статус подписки текущего пользователя, создателя
// @Tags communities
// @Produce json
// @Param id path int true "ID сообщества"
// @Success 200 {object} domain.CommunityForView "Информация о сообществе"
// @Failure 400 {object} JSONResponse "Неверный ID сообщества"
// @Failure 404 {object} JSONResponse "Сообщество не найдено"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/{id} [get]
func (h *CommunityHandler) GetCommunity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid community ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid community ID", zap.String("communityID", vars["id"]))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	community, err := h.communityService.GetCommunity(r.Context(), userID, communityID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, community); err != nil {
		return
	}
}

// GetUserCommunities возвращает сообщества пользователя
// @Summary Получить сообщества пользователя
// @Description Возвращает список сообществ, на которые подписан пользователь
// @Tags communities
// @Produce json
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество сообществ на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.ShortCommunity "Список сообществ пользователя"
// @Failure 400 {object} JSONResponse "Неверные параметры пагинации"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/my [get]
func (h *CommunityHandler) GetUserCommunities(w http.ResponseWriter, r *http.Request) {
	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	communities, err := h.communityService.GetUserCommunities(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, communities); err != nil {
		return
	}
}

// GetOtherCommunities возвращает сообщества, на которые пользователь не подписан
// @Summary Получить другие сообщества
// @Description Возвращает список сообществ, на которые пользователь не подписан (рекомендации)
// @Tags communities
// @Produce json
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество сообществ на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.ShortCommunity "Список рекомендуемых сообществ"
// @Failure 400 {object} JSONResponse "Неверные параметры пагинации"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/other [get]
func (h *CommunityHandler) GetOtherCommunities(w http.ResponseWriter, r *http.Request) {
	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	communities, err := h.communityService.GetOtherCommunities(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, communities); err != nil {
		return
	}
}

// GetUserCommunitiesByID возвращает сообщества, на которые подписан указанный пользователь
// @Summary Получить сообщества пользователя по ID
// @Description Возвращает список сообществ, на которые подписан указанный пользователь
// @Tags communities
// @Produce json
// @Param id path int true "ID пользователя"
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество сообществ на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.ShortCommunity "Список сообществ пользователя"
// @Failure 400 {object} JSONResponse "Неверные параметры запроса"
// @Failure 404 {object} JSONResponse "Пользователь не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/users/{id} [get]
func (h *CommunityHandler) GetUserCommunitiesByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("userID", vars["id"]))
		return
	}

	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	communities, err := h.communityService.GetUserCommunitiesByID(r.Context(), targetUserID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, communities); err != nil {
		return
	}
}

// GetUserSubscribedCommunityIDs возвращает ID сообществ, на которые подписан указанный пользователь
// @Summary Получить ID подписанных сообществ пользователя
// @Description Возвращает список ID сообществ, на которые подписан указанный пользователь
// @Tags communities
// @Produce json
// @Param userID path int true "ID пользователя"
// @Success 200 {array} int "Список ID подписанных сообществ"
// @Failure 400 {object} JSONResponse "Неверный ID пользователя"
// @Failure 404 {object} JSONResponse "Пользователь не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/users/{userID}/subscribed-ids [get]
func (h *CommunityHandler) GetUserSubscribedCommunityIDs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("userID", vars["userID"]))
		return
	}

	communityIDs, err := h.communityService.GetUserSubscribedCommunityIDs(r.Context(), targetUserID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, communityIDs); err != nil {
		return
	}
}

// GetCreatedCommunities возвращает сообщества, созданные пользователем
// @Summary Получить созданные сообщества
// @Description Возвращает список сообществ, созданных текущим пользователем (только ID, название и аватар)
// @Tags communities
// @Produce json
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество сообществ на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.CommunityForMyCommunity "Список созданных сообществ"
// @Failure 400 {object} JSONResponse "Неверные параметры пагинации"
// @Failure 401 {object} JSONResponse "Пользователь не авторизован"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/created [get]
func (h *CommunityHandler) GetCreatedCommunities(w http.ResponseWriter, r *http.Request) {
	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	communities, err := h.communityService.GetCreatedCommunities(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, communities); err != nil {
		return
	}
}

// GetCommunitySubscribers возвращает список подписчиков сообщества
// @Summary Получить подписчиков сообщества
// @Description Возвращает список подписчиков указанного сообщества
// @Tags communities
// @Produce json
// @Param id path int true "ID сообщества"
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество подписчиков на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.CommunitySubscriber "Список подписчиков"
// @Failure 400 {object} JSONResponse "Неверные параметры запроса"
// @Failure 404 {object} JSONResponse "Сообщество не найдено"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/{id}/subscribers [get]
func (h *CommunityHandler) GetCommunitySubscribers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid community ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid community ID", zap.String("communityID", vars["id"]))
		return
	}

	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	subscribers, err := h.communityService.GetCommunitySubscribers(r.Context(), communityID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, subscribers); err != nil {
		return
	}
}

// Subscribe подписывает пользователя на сообщество
// @Summary Подписаться на сообщество
// @Description Подписывает текущего пользователя на указанное сообщество
// @Tags communities
// @Produce json
// @Param id path int true "ID сообщества"
// @Success 200 {object} JSONResponse "Успешная подписка"
// @Failure 400 {object} JSONResponse "Неверный ID сообщества"
// @Failure 404 {object} JSONResponse "Сообщество не найдено"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/{id}/subscribe [post]
func (h *CommunityHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid community ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid community ID", zap.String("communityID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.communityService.Subscribe(r.Context(), communityID, userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Subscribed successfully", http.StatusOK)
}

// Unsubscribe отписывает пользователя от сообщества
// @Summary Отписаться от сообщества
// @Description Отписывает текущего пользователя от указанного сообщества
// @Tags communities
// @Produce json
// @Param id path int true "ID сообщества"
// @Success 200 {object} JSONResponse "Успешная отписка"
// @Failure 400 {object} JSONResponse "Неверный ID сообщества"
// @Failure 404 {object} JSONResponse "Подписка не найдена"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/{id}/unsubscribe [post]
func (h *CommunityHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid community ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid community ID", zap.String("communityID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.communityService.Unsubscribe(r.Context(), communityID, userID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Unsubscribed successfully", http.StatusOK)
}

// SearchCommunityByName ищет сообщества по имени.
//
// @Summary Поиск сообществ по имени
// @Description Возвращает список сообществ, имя которых соответствует поисковому запросу.
//
//	Можно фильтровать по типу подписки (например, подписан или нет).
//
// @Tags community
// @Produce json
// @Param name query string true "Полное или частичное имя сообщества"
// @Param type query string false "Тип подписки: subscriber, notSubscriber" default(recommended) Enums(subscriber, recommended)
// @Param limit query int false "Лимит количества сообществ" default(20)
// @Param page query int false "Номер страницы для пагинации" default(1)
// @Success 200 {array} domain.ShortCommunity "Найденные сообщества"
// @Failure 400 {string} string "Missing name query parameter"
// @Failure 500 {string} string "Server error"
// @Router /community/search [get]
func (api *CommunityHandler) SearchCommunityByName(w http.ResponseWriter, r *http.Request) {

	name := r.URL.Query().Get("name")
	if name == "" {
		sendJSONResponse(w, "Missing name query parameter", http.StatusBadRequest)
		domain.FromContext(r.Context()).Warn("name query parameter is missing")
		return
	}
	cTypeStr := r.URL.Query().Get("type")
	if cTypeStr == "" {
		cTypeStr = string(domain.Subscriber) // значение по умолчанию
	}

	cType := domain.CommunityType(cTypeStr)

	var qParams domain.PaginateQueryParams
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	if err := decoder.Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONError(w, err)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(qParams)))
		return
	}
	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	com, err := api.communityService.SearchShortCommunityByNameAndType(r.Context(), userID, qParams, name, cType)
	if err != nil {
		sendJSONError(w, err)
		domain.FromContext(r.Context()).Error("Fail search profiles by full name", zap.Error(err))
		return
	}

	err = sendJSONData(r.Context(), w, com)
	if err == nil {
		domain.FromContext(r.Context()).Info("Profiles returned successfully")
	}
}
