package handler

import (
	"net/http"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"

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

	friendID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	_, err = h.friendService.SendFriendRequest(r.Context(), &pb.SendFriendRequestRequest{ActionUserID: userID, TargetUserID: friendID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, domain.FriendRequestSent)
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
	friendID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	_, err = h.friendService.AcceptFriendRequest(r.Context(), &pb.UserIDsPair{UserID: userID, FriendID: friendID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, domain.FriendRequestAccepted)
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
	friendID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	_, err = h.friendService.RejectFriendRequest(r.Context(), &pb.UserIDsPair{UserID: userID, FriendID: friendID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, domain.FriendRequestRejected)
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
	friendID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	_, err = h.friendService.RemoveFriend(r.Context(), &pb.UserIDsPair{UserID: userID, FriendID: friendID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, domain.FriendRemoved)
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
	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	friends, err := h.friendService.GetFriends(r.Context(), &pb.GetFriendsRequest{UserID: userID, Page: qParams.Page, Limit: qParams.Limit})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, generated.FromPbShortProfileList(friends))
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
	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	users, err := h.friendService.GetAllUsers(r.Context(), &pb.GetAllUsersRequest{UserID: userID, Limit: qParams.Limit, Page: qParams.Page})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, generated.FromPbShortProfileList(users))
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
	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	requests, err := h.friendService.GetFriendRequests(r.Context(), &pb.GetFriendRequestsRequest{UserID: userID, Page: qParams.Page, Limit: qParams.Limit})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, generated.FromPbShortProfileList(requests))
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
	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	requests, err := h.friendService.GetSentRequests(r.Context(), &pb.GetSentRequestsRequest{UserID: userID, Limit: qParams.Limit, Page: qParams.Page})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONData(r.Context(), w, generated.FromPbShortProfileList(requests))
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
	friendID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	status, err := h.friendService.GetFriendshipStatus(r.Context(), &pb.GetFriendshipStatusRequest{UserID: userID, FriendID: friendID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	response := domain.FriendshipStatusResponse{
		Status: domain.FriendshipStatus(status.Status),
	}

	sendJSONData(r.Context(), w, response)
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
	userID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	resp, err := h.friendService.CountUserRelations(r.Context(), &pb.CountUserRelationsRequest{UserID: userID})
	if err != nil {
		sendJSONError(w, err)
		return
	}

	count := domain.UserRelationsCounts{Pending: resp.Pending, Accepted: resp.Accepted, Sent: resp.Sent, Blocked: resp.Blocked}

	sendJSONData(r.Context(), w, count)
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

	qParams, err := DecodeQueryParams[domain.PaginateQueryParams](r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	resp, err := api.friendService.SearchShortProfilesByFullNameAndRelationType(r.Context(), &pb.SearchProfilesRequest{FullName: fullName, UserID: userID, Limit: qParams.Limit, Page: qParams.Page, Type: string(fType)})
	if err != nil {
		sendJSONError(w, err)
		domain.FromContext(r.Context()).Error("Fail search profiles by full name", zap.Error(err))
		return
	}

	profiles := generated.FromPbShortProfileList(resp)

	sendJSONData(r.Context(), w, profiles)
}
