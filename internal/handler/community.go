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

// GetCommunityPosts возвращает посты сообщества
// @Summary Получить посты сообщества
// @Description Возвращает посты указанного сообщества с пагинацией
// @Tags communities
// @Produce json
// @Param id path int true "ID сообщества"
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество постов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.Post "Список постов сообщества"
// @Failure 400 {object} JSONResponse "Неверные параметры запроса"
// @Failure 404 {object} JSONResponse "Сообщество не найдено"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /communities/{id}/posts [get]
func (h *CommunityHandler) GetCommunityPosts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	communityID, err := strconv.ParseUint(vars["id"], 10, 32)
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

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	posts, err := h.communityService.GetCommunityPosts(r.Context(), userID, int(communityID), qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, posts); err != nil {
		return
	}
}
