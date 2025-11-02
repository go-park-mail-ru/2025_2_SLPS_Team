package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"project/domain"
	"project/internal/service"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type FriendHandler struct {
	friendStore domain.FriendStore
	userStore   domain.UserStore
}

func NewFriendHandler(friendStore domain.FriendStore, userStore domain.UserStore) *FriendHandler {
	return &FriendHandler{
		friendStore: friendStore,
		userStore:   userStore,
	}
}

// FriendsRequest - запрос для пагинации друзей
type FriendsRequest struct {
	Page  int `schema:"page"`
	Limit int `schema:"limit"`
}

// FriendsResponse - ответ со списком друзей и пагинацией
type FriendsResponse struct {
	Friends    []domain.ShortProfile `json:"friends"`
	Page       int                   `json:"page"`
	TotalPages int                   `json:"totalPages"`
	HasNext    bool                  `json:"hasNext"`
}

// FriendRequestsResponse - ответ со списком запросов и пагинацией
type FriendRequestsResponse struct {
	Requests   []domain.FriendshipWithProfile `json:"requests"`
	Page       int                            `json:"page"`
	TotalPages int                            `json:"totalPages"`
	HasNext    bool                           `json:"hasNext"`
}

// SendFriendRequest отправляет запрос в друзья
// @Summary Отправить запрос в друзья
// @Description Отправляет запрос на дружбу другому пользователю
// @Tags friends
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 409 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /friends/{id} [post]
func (h *FriendHandler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	// Нельзя отправить запрос самому себе
	if userID == friendID {
		sendJSONError(w, "Cannot send friend request to yourself", http.StatusBadRequest)
		service.Warn(r.Context(), "User tried to send friend request to themselves")
		return
	}

	service.Info(r.Context(), "Sending friend request",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

	// Проверяем существование пользователя
	_, err = h.userStore.GetUserByID(friendID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			sendJSONError(w, "User not found", http.StatusNotFound)
			service.Warn(r.Context(), "Friend user not found", zap.Int("friendID", friendID))
		} else {
			service.Error(r.Context(), "Failed to get user", err, zap.Int("friendID", friendID))
			sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Проверяем текущий статус дружбы
	currentStatus, err := h.friendStore.GetFriendshipStatus(r.Context(), userID, friendID)
	if err != nil && !errors.Is(err, domain.ErrFriendshipNotFound) {
		service.Error(r.Context(), "Failed to check friendship status", err)
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	// Обработка различных статусов
	switch currentStatus {
	case domain.FriendshipAccepted:
		sendJSONError(w, "Users are already friends", http.StatusConflict)
		service.Warn(r.Context(), "Friend request to existing friend")
		return
	case domain.FriendshipPending:
		// Определяем кто отправитель запроса
		friendship, err := h.friendStore.GetFriendship(r.Context(), userID, friendID)
		if err != nil {
			service.Error(r.Context(), "Failed to get friendship details", err)
			sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
			return
		}

		if friendship.FirstUserID == userID {
			sendJSONError(w, "Friend request already sent", http.StatusConflict)
			service.Warn(r.Context(), "Duplicate friend request")
		} else {
			sendJSONError(w, "You have incoming friend request from this user", http.StatusConflict)
			service.Warn(r.Context(), "Friend request to user who already sent request")
		}
		return
	case domain.FriendshipBlocked:
		sendJSONError(w, "Cannot send request to blocked user", http.StatusForbidden)
		service.Warn(r.Context(), "Friend request to blocked user")
		return
	}

	// Создаем запрос в друзья
	err = h.friendStore.CreateFriendship(r.Context(), userID, friendID)
	if err != nil {
		service.Error(r.Context(), "Failed to send friend request", err)
		sendJSONError(w, "Failed to send friend request", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Friend request sent successfully", http.StatusOK)
	service.Info(r.Context(), "Friend request sent successfully")
}

// AcceptFriendRequest принимает запрос в друзья
// @Summary Принять запрос в друзья
// @Description Принимает входящий запрос на дружбу
// @Tags friends
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя (отправителя запроса)"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /friends/{id}/accept [put]
func (h *FriendHandler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Accepting friend request",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

	// Проверяем существование запроса
	friendship, err := h.friendStore.GetFriendship(r.Context(), userID, friendID)
	if err != nil {
		if errors.Is(err, domain.ErrFriendshipNotFound) {
			sendJSONError(w, "Friend request not found", http.StatusNotFound)
			service.Warn(r.Context(), "Friend request not found")
		} else {
			service.Error(r.Context(), "Failed to get friendship", err)
			sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Проверяем что запрос pending и пользователь является получателем
	if friendship.Status != domain.FriendshipPending || friendship.SecondUserID != userID {
		sendJSONError(w, "No pending friend request found", http.StatusNotFound)
		service.Warn(r.Context(), "No pending friend request or user is not receiver")
		return
	}

	err = h.friendStore.UpdateFriendshipStatus(r.Context(), userID, friendID, domain.FriendshipAccepted)
	if err != nil {
		service.Error(r.Context(), "Failed to accept friend request", err)
		sendJSONError(w, "Failed to accept friend request", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Friend request accepted successfully", http.StatusOK)
	service.Info(r.Context(), "Friend request accepted successfully")
}

// RejectFriendRequest отклоняет запрос в друзья
// @Summary Отклонить запрос в друзья
// @Description Отклоняет входящий запрос на дружбу
// @Tags friends
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя (отправителя запроса)"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /friends/{id}/reject [put]
func (h *FriendHandler) RejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Rejecting friend request",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

	// Проверяем существование запроса
	friendship, err := h.friendStore.GetFriendship(r.Context(), userID, friendID)
	if err != nil {
		if errors.Is(err, domain.ErrFriendshipNotFound) {
			sendJSONError(w, "Friend request not found", http.StatusNotFound)
			service.Warn(r.Context(), "Friend request not found")
		} else {
			service.Error(r.Context(), "Failed to get friendship", err)
			sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		}
		return
	}

	// Проверяем что запрос pending и пользователь является получателем
	if friendship.Status != domain.FriendshipPending || friendship.SecondUserID != userID {
		sendJSONError(w, "No pending friend request found", http.StatusNotFound)
		service.Warn(r.Context(), "No pending friend request or user is not receiver")
		return
	}

	// Удаляем запись вместо установки статуса rejected
	err = h.friendStore.DeleteFriendship(r.Context(), userID, friendID)
	if err != nil {
		service.Error(r.Context(), "Failed to reject friend request", err)
		sendJSONError(w, "Failed to reject friend request", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Friend request rejected successfully", http.StatusOK)
	service.Info(r.Context(), "Friend request rejected successfully")
}

// RemoveFriend удаляет из друзей
// @Summary Удалить из друзей
// @Description Удаляет пользователя из списка друзей
// @Tags friends
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} JSONResponse
// @Failure 400 {object} JSONResponse
// @Failure 404 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /friends/{id} [delete]
func (h *FriendHandler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Removing friend",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

	// Проверяем что пользователи действительно друзья
	areFriends, err := h.friendStore.AreFriends(r.Context(), userID, friendID)
	if err != nil {
		service.Error(r.Context(), "Failed to check friendship", err)
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	if !areFriends {
		sendJSONError(w, "Users are not friends", http.StatusNotFound)
		service.Warn(r.Context(), "Attempt to remove non-friend")
		return
	}

	err = h.friendStore.DeleteFriendship(r.Context(), userID, friendID)
	if err != nil {
		service.Error(r.Context(), "Failed to remove friend", err)
		sendJSONError(w, "Failed to remove friend", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Friend removed successfully", http.StatusOK)
	service.Info(r.Context(), "Friend removed successfully")
}

// GetFriends получает список друзей
// @Summary Получить список друзей
// @Description Возвращает список друзей пользователя с пагинацией
// @Tags friends
// @Produce json
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество друзей на странице" default(20)
// @Success 200 {object} FriendsResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /friends [get]
func (h *FriendHandler) GetFriends(w http.ResponseWriter, r *http.Request) {
	var req FriendsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONError(w, domain.InvalidParams, http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	// Устанавливаем значения по умолчанию
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Getting user friends",
		zap.Int("userID", userID),
		zap.Int("page", req.Page),
		zap.Int("limit", req.Limit))

	friends, totalPages, err := h.friendStore.GetUserFriends(r.Context(), userID, req.Page, req.Limit)
	if err != nil {
		service.Error(r.Context(), "Failed to get user friends", err)
		sendJSONError(w, "Failed to get friends", http.StatusInternalServerError)
		return
	}

	response := FriendsResponse{
		Friends:    friends,
		Page:       req.Page,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", "FriendsResponse"))
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	service.Info(r.Context(), "User friends retrieved successfully",
		zap.Int("friendsCount", len(friends)),
		zap.Int("totalPages", totalPages))
}

// GetFriendRequests получает входящие запросы в друзья
// @Summary Получить входящие запросы в друзья
// @Description Возвращает список входящих запросов на дружбу с пагинацией
// @Tags friends
// @Produce json
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество запросов на странице" default(20)
// @Success 200 {object} FriendRequestsResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /friends/requests [get]
func (h *FriendHandler) GetFriendRequests(w http.ResponseWriter, r *http.Request) {
	var req FriendsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONError(w, domain.InvalidParams, http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	// Устанавливаем значения по умолчанию
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Getting friendship requests",
		zap.Int("userID", userID),
		zap.Int("page", req.Page),
		zap.Int("limit", req.Limit))

	requests, totalPages, err := h.friendStore.GetFriendshipRequests(r.Context(), userID, req.Page, req.Limit)
	if err != nil {
		service.Error(r.Context(), "Failed to get friendship requests", err)
		sendJSONError(w, "Failed to get friend requests", http.StatusInternalServerError)
		return
	}

	response := FriendRequestsResponse{
		Requests:   requests,
		Page:       req.Page,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", "FriendRequestsResponse"))
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	service.Info(r.Context(), "Friendship requests retrieved successfully",
		zap.Int("requestsCount", len(requests)),
		zap.Int("totalPages", totalPages))
}

// GetSentRequests получает отправленные запросы в друзья
// @Summary Получить отправленные запросы в друзья
// @Description Возвращает список отправленных запросов на дружбу с пагинацией
// @Tags friends
// @Produce json
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество запросов на странице" default(20)
// @Success 200 {object} FriendRequestsResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /friends/sent [get]
func (h *FriendHandler) GetSentRequests(w http.ResponseWriter, r *http.Request) {
	var req FriendsRequest
	if err := schema.NewDecoder().Decode(&req, r.URL.Query()); err != nil {
		sendJSONError(w, domain.InvalidParams, http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid query parameters", zap.Error(err))
		return
	}

	// Устанавливаем значения по умолчанию
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Getting sent friend requests",
		zap.Int("userID", userID),
		zap.Int("page", req.Page),
		zap.Int("limit", req.Limit))

	requests, totalPages, err := h.friendStore.GetSentRequests(r.Context(), userID, req.Page, req.Limit)
	if err != nil {
		service.Error(r.Context(), "Failed to get sent requests", err)
		sendJSONError(w, "Failed to get sent requests", http.StatusInternalServerError)
		return
	}

	response := FriendRequestsResponse{
		Requests:   requests,
		Page:       req.Page,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", "FriendRequestsResponse"))
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	service.Info(r.Context(), "Sent requests retrieved successfully",
		zap.Int("requestsCount", len(requests)),
		zap.Int("totalPages", totalPages))
}

// GetFriendshipStatus получает статус дружбы с пользователем
// @Summary Получить статус дружбы
// @Description Возвращает текущий статус дружбы с указанным пользователем
// @Tags friends
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} FriendshipStatusResponse
// @Failure 400 {object} JSONResponse
// @Failure 500 {object} JSONResponse
// @Security ApiKeyAuth
// @Router /friends/{id}/status [get]
func (h *FriendHandler) GetFriendshipStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	friendID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONError(w, "Invalid user ID", http.StatusBadRequest)
		service.Warn(r.Context(), "Invalid user ID", zap.String("friendID", vars["id"]))
		return
	}

	userID, ok := r.Context().Value(domain.UserIDKey).(int)
	if !ok {
		sendJSONError(w, domain.Unauthorized, http.StatusUnauthorized)
		service.Warn(r.Context(), "User ID not found in context")
		return
	}

	service.Info(r.Context(), "Getting friendship status",
		zap.Int("userID", userID),
		zap.Int("friendID", friendID))

	status, err := h.friendStore.GetFriendshipStatus(r.Context(), userID, friendID)
	if err != nil && !errors.Is(err, domain.ErrFriendshipNotFound) {
		service.Error(r.Context(), "Failed to get friendship status", err)
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": status,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", "FriendshipStatusResponse"))
		sendJSONError(w, domain.ServerErr, http.StatusInternalServerError)
		return
	}

	service.Info(r.Context(), "Friendship status retrieved successfully")
}

// FriendshipStatusResponse - ответ со статусом дружбы
type FriendshipStatusResponse struct {
	Status domain.FriendshipStatus `json:"status"`
}
