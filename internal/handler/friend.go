package handler

import (
	"net/http"
	"project/domain"

	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type FriendHandler struct {
	friendService domain.FriendService
}

func NewFriendHandler(friendService domain.FriendService) *FriendHandler {
	return &FriendHandler{
		friendService: friendService,
	}
}

// FriendshipStatusResponse - ответ со статусом дружбы
// @Description Ответ с текущим статусом дружбы между пользователями
type FriendshipStatusResponse struct {
	Status domain.FriendshipStatus `json:"status" example:"pending" enums:"pending,accepted,rejected,blocked"` // Статус дружбы
}

// SendFriendRequest отправляет запрос в друзья
// @Summary Отправить запрос в друзья
// @Description Отправляет запрос на дружбу другому пользователю
// @Tags friends
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя, которому отправляется запрос" minimum(1)
// @Success 200 {object} JSONResponse "Запрос успешно отправлен"
// @Failure 400 {object} JSONResponse "Неверный ID пользователя или попытка добавить самого себя"
// @Failure 404 {object} JSONResponse "Пользователь не найден"
// @Failure 409 {object} JSONResponse "Запрос уже существует или пользователи уже друзья"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/{id} [post]
func (h *FriendHandler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.friendService.SendFriendRequest(r.Context(), userID, friendID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, domain.FriendRequestSent, http.StatusOK)
}

// AcceptFriendRequest принимает запрос в друзья
// @Summary Принять запрос в друзья
// @Description Принимает входящий запрос на дружбу
// @Tags friends
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя (отправителя запроса)" minimum(1)
// @Success 200 {object} JSONResponse "Запрос успешно принят"
// @Failure 400 {object} JSONResponse "Неверный ID пользователя"
// @Failure 404 {object} JSONResponse "Запрос не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/{id}/accept [put]
func (h *FriendHandler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.friendService.AcceptFriendRequest(r.Context(), userID, friendID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, domain.FriendRequestAccepted, http.StatusOK)
}

// RejectFriendRequest отклоняет запрос в друзья
// @Summary Отклонить запрос в друзья
// @Description Отклоняет входящий запрос на дружбу
// @Tags friends
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя (отправителя запроса)" minimum(1)
// @Success 200 {object} JSONResponse "Запрос успешно отклонен"
// @Failure 400 {object} JSONResponse "Неверный ID пользователя"
// @Failure 404 {object} JSONResponse "Запрос не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/{id}/reject [put]
func (h *FriendHandler) RejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.friendService.RejectFriendRequest(r.Context(), userID, friendID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, domain.FriendRequestRejected, http.StatusOK)
}

// RemoveFriend удаляет из друзей
// @Summary Удалить из друзей
// @Description Удаляет пользователя из списка друзей
// @Tags friends
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя" minimum(1)
// @Success 200 {object} JSONResponse "Пользователь успешно удален из друзей"
// @Failure 400 {object} JSONResponse "Неверный ID пользователя"
// @Failure 404 {object} JSONResponse "Пользователи не являются друзьями"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/{id} [delete]
func (h *FriendHandler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	err = h.friendService.RemoveFriend(r.Context(), userID, friendID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, domain.FriendRemoved, http.StatusOK)
}
// GetFriends получает список друзей
// @Summary Получить список друзей
// @Description Возвращает список друзей пользователя с пагинацией
// @Tags friends
// @Produce json
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество друзей на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.ShortProfile "Успешный ответ со списком друзей"
// @Failure 400 {object} JSONResponse "Неверные параметры пагинации"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends [get]
func (h *FriendHandler) GetFriends(w http.ResponseWriter, r *http.Request) {
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

	friends, err := h.friendService.GetFriends(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, friends); err != nil {
		return
	}
}

// GetAllUsers получает всех пользователей кроме текущего
// @Summary Получить всех пользователей (кроме себя)
// @Description Возвращает список всех пользователей кроме текущего пользователя с пагинацией
// @Tags friends
// @Produce json
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество пользователей на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.ShortProfile "Успешный ответ со списком пользователей"
// @Failure 400 {object} JSONResponse "Неверные параметры пагинации"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/users/all [get]
func (h *FriendHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
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

    users, err := h.friendService.GetAllUsers(r.Context(), userID, qParams)
    if err != nil {
        sendJSONError(w, err)
        return
    }

    if err := sendJSONData(r.Context(), w, users); err != nil {
        return
    }
}

// GetFriendRequests получает входящие запросы в друзья
// @Summary Получить входящие запросы в друзья
// @Description Возвращает список входящих запросов на дружбу с пагинацией
// @Tags friends
// @Produce json
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество запросов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.FriendshipWithProfile "Успешный ответ со списком запросов"
// @Failure 400 {object} JSONResponse "Неверные параметры пагинации"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/requests [get]
func (h *FriendHandler) GetFriendRequests(w http.ResponseWriter, r *http.Request) {
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

	requests, err := h.friendService.GetFriendRequests(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, requests); err != nil {
		return
	}
}

// GetSentRequests получает отправленные запросы в друзья
// @Summary Получить отправленные запросы в друзья
// @Description Возвращает список отправленных запросов на дружбу с пагинацией
// @Tags friends
// @Produce json
// @Param page query int false "Номер страницы" default(1) minimum(1)
// @Param limit query int false "Количество запросов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.FriendshipWithProfile "Успешный ответ со списком отправленных запросов"
// @Failure 400 {object} JSONResponse "Неверные параметры пагинации"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/sent [get]
func (h *FriendHandler) GetSentRequests(w http.ResponseWriter, r *http.Request) {
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

	requests, err := h.friendService.GetSentRequests(r.Context(), userID, qParams)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, requests); err != nil {
		return
	}
}

// GetFriendshipStatus получает статус дружбы с пользователем
// @Summary Получить статус дружбы
// @Description Возвращает текущий статус дружбы с указанным пользователем
// @Tags friends
// @Produce json
// @Param id path int true "ID пользователя" minimum(1)
// @Success 200 {object} FriendshipStatusResponse "Успешный ответ со статусом дружбы"
// @Failure 400 {object} JSONResponse "Неверный ID пользователя"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/{id}/status [get]
func (h *FriendHandler) GetFriendshipStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	status, err := h.friendService.GetFriendshipStatus(r.Context(), userID, friendID)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	response := FriendshipStatusResponse{
		Status: status,
	}

	if err := sendJSONData(r.Context(), w, response); err != nil {
		return
	}
}
