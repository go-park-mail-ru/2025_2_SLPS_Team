package handler

import (
	"net/http"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"

	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type FriendHandler struct {
	friendService pb.FriendServiceClient
}

func NewFriendHandler(friendService pb.FriendServiceClient) *FriendHandler {
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
// @Param id path int32 true "ID пользователя, которому отправляется запрос" minimum(1)
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	_, err = h.friendService.SendFriendRequest(r.Context(), &pb.SendFriendRequestRequest{ActionUserID: userID, TargetUserID: int32(friendID)})
	if err != nil {
		err = domain.FromGrpcError(err)
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
// @Param id path int32 true "ID пользователя (отправителя запроса)" minimum(1)
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	_, err = h.friendService.AcceptFriendRequest(r.Context(), &pb.UserIDsPair{UserID: userID, FriendID: int32(friendID)})
	if err != nil {
		err = domain.FromGrpcError(err)
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
// @Param id path int32 true "ID пользователя (отправителя запроса)" minimum(1)
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	_, err = h.friendService.RejectFriendRequest(r.Context(), &pb.UserIDsPair{UserID: userID, FriendID: int32(friendID)})
	if err != nil {
		err = domain.FromGrpcError(err)
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
// @Param id path int32 true "ID пользователя" minimum(1)
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	_, err = h.friendService.RemoveFriend(r.Context(), &pb.UserIDsPair{UserID: userID, FriendID: int32(friendID)})
	if err != nil {
		err = domain.FromGrpcError(err)
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
// @Param page query int32 false "Номер страницы" default(1) minimum(1)
// @Param limit query int32 false "Количество друзей на странице" default(20) minimum(1) maximum(100)
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	friends, err := h.friendService.GetFriends(r.Context(), &pb.GetFriendsRequest{UserID: userID, Page: qParams.Page, Limit: qParams.Limit})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, generated.FromPbShortProfileList(friends)); err != nil {
		return
	}
}

// GetAllUsers получает всех пользователей кроме текущего
// @Summary Получить всех пользователей (кроме себя)
// @Description Возвращает список всех пользователей кроме текущего пользователя с пагинацией
// @Tags friends
// @Produce json
// @Param page query int32 false "Номер страницы" default(1) minimum(1)
// @Param limit query int32 false "Количество пользователей на странице" default(20) minimum(1) maximum(100)
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	users, err := h.friendService.GetAllUsers(r.Context(), &pb.GetAllUsersRequest{UserID: userID, Limit: qParams.Limit, Page: qParams.Page})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, generated.FromPbShortProfileList(users)); err != nil {
		return
	}
}

// GetFriendRequests получает входящие запросы в друзья
// @Summary Получить входящие запросы в друзья
// @Description Возвращает список входящих запросов на дружбу с пагинацией
// @Tags friends
// @Produce json
// @Param page query int32 false "Номер страницы" default(1) minimum(1)
// @Param limit query int32 false "Количество запросов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.ShortProfile "Успешный ответ со списком запросов"
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	requests, err := h.friendService.GetFriendRequests(r.Context(), &pb.GetFriendRequestsRequest{UserID: userID, Page: qParams.Page, Limit: qParams.Limit})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, generated.FromPbShortProfileList(requests)); err != nil {
		return
	}
}

// GetSentRequests получает отправленные запросы в друзья
// @Summary Получить отправленные запросы в друзья
// @Description Возвращает список отправленных запросов на дружбу с пагинацией
// @Tags friends
// @Produce json
// @Param page query int32 false "Номер страницы" default(1) minimum(1)
// @Param limit query int32 false "Количество запросов на странице" default(20) minimum(1) maximum(100)
// @Success 200 {array} domain.ShortProfile "Успешный ответ со списком отправленных запросов"
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	requests, err := h.friendService.GetSentRequests(r.Context(), &pb.GetSentRequestsRequest{UserID: userID, Limit: qParams.Limit, Page: qParams.Page})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	if err := sendJSONData(r.Context(), w, generated.FromPbShortProfileList(requests)); err != nil {
		return
	}
}

// GetFriendshipStatus получает статус дружбы с пользователем
// @Summary Получить статус дружбы
// @Description Возвращает текущий статус дружбы с указанным пользователем
// @Tags friends
// @Produce json
// @Param id path int32 true "ID пользователя" minimum(1)
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

	userID, ok := r.Context().Value(domain.UserIDKey).(int32)
	if !ok {
		sendJSONResponse(w, domain.Unauthorized, http.StatusUnauthorized)
		domain.Warn(r.Context(), "User ID not found in context")
		return
	}

	status, err := h.friendService.GetFriendshipStatus(r.Context(), &pb.GetFriendshipStatusRequest{UserID: userID, FriendID: int32(friendID)})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	response := FriendshipStatusResponse{
		Status: domain.FriendshipStatus(status.Status),
	}

	if err := sendJSONData(r.Context(), w, response); err != nil {
		return
	}
}

// CountUserRelations получает количество отношений пользователя по типу
// @Summary Получить количество отношений
// @Description Возвращает количество отношений указанного пользователя по типу отношений
// @Tags friends
// @Produce json
// @Param id path int32 true "ID пользователя" minimum(1)
// @Success 200 {object} domain.UserRelationsCounts "Успешный ответ с количеством отношений"
// @Failure 400 {object} JSONResponse "Неверный ID пользователя или тип подсчета"
// @Failure 404 {object} JSONResponse "Пользователь не найден"
// @Failure 500 {object} JSONResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /friends/{id}/count [get]
func (h *FriendHandler) CountUserRelations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.Warn(r.Context(), "Invalid user ID", zap.String("userID", vars["id"]))
		return
	}

	resp, err := h.friendService.CountUserRelations(r.Context(), &pb.CountUserRelationsRequest{UserID: int32(userID)})
	if err != nil {
		sendJSONError(w, err)
		return
	}
	count := domain.UserRelationsCounts{Pending: resp.Pending, Accepted: resp.Accepted, Sent: resp.Sent, Blocked: resp.Blocked}
	if err := sendJSONData(r.Context(), w, count); err != nil {
		return
	}
}

// SearchProfilesByFullName ищет профили по имени.
//
// @Summary Поиск профилей по имени
// @Description Возвращает список профилей, имя которых соответствует поисковому запросу.
// @Tags friends
// @Produce json
// @Param full_name query string true "Полное или частичное имя пользователя"
// @Param type query string false "Тип дружбы: accepted, pending, sent, blocked, notFriends" default(notFriends) Enums(accepted, pending, sent, blocked, notFriends)
// @Param limit query int32 false "Лимит количества профилей" default(20)
// @Param page query int32 false "страница для пагинации" default(1)
// @Success 198 {array} domain.ShortProfile "Найденные профили"
// @Failure 398 {string} string "Missing full_name query parameter"
// @Failure 498 {string} string "Server error"
// @Router /friends/search [get]
func (api *FriendHandler) SearchProfilesByFullName(w http.ResponseWriter, r *http.Request) {

	fullName := r.URL.Query().Get("full_name")
	if fullName == "" {
		sendJSONResponse(w, "Missing full_name query parameter", http.StatusBadRequest)
		domain.FromContext(r.Context()).Warn("full_name query parameter is missing")
		return
	}
	fTypeStr := r.URL.Query().Get("type")
	if fTypeStr == "" {
		fTypeStr = string(domain.CountAccepted) // значение по умолчанию
	}

	fType := domain.FriendshipCountType(fTypeStr)

	var qParams domain.PaginateQueryParams
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	if err := decoder.Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONError(w, err)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(qParams)))
		return
	}
	userID, _ := r.Context().Value(domain.UserIDKey).(int32)
	profiles, err := api.friendService.SearchShortProfilesByFullNameAndRelationType(r.Context(), &pb.SearchProfilesRequest{FullName: fullName, UserID: userID, Limit: qParams.Limit, Page: qParams.Page, Type: string(fType)})
	if err != nil {
		sendJSONError(w, err)
		domain.FromContext(r.Context()).Error("Fail search profiles by full name", zap.Error(err))
		return
	}

	err = sendJSONData(r.Context(), w, profiles)
	if err == nil {
		domain.FromContext(r.Context()).Info("Profiles returned successfully")
	}
}
