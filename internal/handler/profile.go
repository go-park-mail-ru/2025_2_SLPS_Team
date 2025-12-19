package handler

import (
	"encoding/json"
	"net/http"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"
	"strings"

	"go.uber.org/zap"
)

type ProfileHandler struct {
	profileService pb.ProfileServiceClient
}

func NewProfileHandler(profileService pb.ProfileServiceClient) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// UpdateProfile обновляет данные профиля пользователя.
//
// @Summary Обновление профиля
// @Description Обновляет данные пользователя и загружает новый аватар, если он передан.
// @Tags profile
// @Accept multipart/form-data
// @Produce json
// @Param profile formData string true "JSON с данными профиля"
// @Param avatar formData file false "Новый аватар пользователя"
// @Success 200 {string} string "Profile updated"
// @Failure 400 {string} string "Invalid data / missing fields"
// @Failure 500 {string} string "Server error"
// @Security ApiKeyAuth
// @Router /profile [put]
func (api *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	err := ParseMultipart(r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	jsonProfile := r.FormValue("profile")
	if jsonProfile == "" {
		sendJSONResponse(w, "Missing profile field", http.StatusBadRequest)
		domain.FromContext(r.Context()).Warn("Missing profile field in request")
		return
	}

	var req domain.Profile
	if err := json.NewDecoder(strings.NewReader(jsonProfile)).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to decode profile JSON", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	files, err := domain.MultipartFiles(r, "avatar")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	_, err = api.profileService.UpdateProfile(r.Context(), &pb.UpdateProfileRequest{Profile: generated.ToProtoProfile(req), UserID: userID, Files: generated.FilesToProto(files)})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "Profile updated")
}

// UpdateAvatar обновляет аватар пользователя.
//
// @Summary Обновление аватара
// @Description Загружает новый аватар пользователя.
// @Tags profile
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Новый аватар пользователя"
// @Success 200 {string} string "Avatar updated"
// @Failure 400 {string} string "Missing avatar field"
// @Failure 500 {string} string "Server error"
// @Security ApiKeyAuth
// @Router /profile/avatar [put]
func (api *ProfileHandler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	err := ParseMultipart(r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	files, err := domain.MultipartFiles(r, "avatar")
	if err != nil {
		sendJSONError(w, err)
		return
	}
	_, err = api.profileService.UpdateAvatar(r.Context(), &pb.UpdateAvatarRequest{Avatar: generated.FilesToProto(files), UserID: userID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "Avatar updated")
}

// UpdateHeader обновляет header пользователя.
//
// @Summary Обновление header
// @Description Загружает новый header пользователя.
// @Tags profile
// @Accept multipart/form-data
// @Produce json
// @Param header formData file true "Новый header пользователя"
// @Success 200 {string} string "Header updated"
// @Failure 400 {string} string "Missing header field"
// @Failure 500 {string} string "Server error"
// @Security ApiKeyAuth
// @Router /profile/header [put]
func (api *ProfileHandler) UpdateHeader(w http.ResponseWriter, r *http.Request) {
	err := ParseMultipart(r)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	files, err := domain.MultipartFiles(r, "header")
	if err != nil {
		sendJSONError(w, err)
		return
	}
	_, err = api.profileService.UpdateAvatar(r.Context(), &pb.UpdateAvatarRequest{Avatar: generated.FilesToProto(files), UserID: userID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "Header updated")
}

// GetProfileByUserID получает профиль пользователя по ID.
//
// @Summary Получение профиля по ID
// @Description Возвращает профиль пользователя по его ID.
// @Tags profile
// @Produce json
// @Param id path int32 true "ID пользователя"
// @Success 200 {object} domain.Profile "Профиль пользователя"
// @Failure 400 {string} string "Invalid user ID / User does not exist"
// @Failure 500 {string} string "Server error"
// @Security ApiKeyAuth
// @Router /profile/{id} [get]
func (api *ProfileHandler) GetProfileByUserID(w http.ResponseWriter, r *http.Request) {
	userID, err := PathInt32(r, "id")
	if err != nil {
		sendJSONError(w, err)
		return
	}

	selfUserID, _ := r.Context().Value(domain.UserIDKey).(int32)

	resp, err := api.profileService.GetProfileByUserID(r.Context(), &pb.GetProfileByUserIDRequest{UserID: userID, SelfUserID: selfUserID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	profile := generated.FromProtoProfile(resp.Profile)

	sendJSONData(r.Context(), w, profile)
}

// DeleteAvatar Удаление аватара
// @Summary      Удалить аватар пользователя
// @Description  Очищает поле avatar_path в профиле текущего пользователя и возвращает сообщение об успешном удалении.
// @Tags         profile
// @Accept       json
// @Produce      json
// @Success      200 {object} JSONResponse "Avatar deleted"
// @Failure      500 {object} JSONResponse "Internal Server Error"
// @Security     BearerAuth
// @Router       /profile/avatar [delete]
func (api *ProfileHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(domain.UserIDKey).(int32)

	_, err := api.profileService.DeleteAvatarByUserID(r.Context(), &pb.DeleteAvatarRequest{UserID: userID})
	if err != nil {
		err = domain.FromGrpcError(err)
		sendJSONError(w, err)
		return
	}

	sendJSONSuccess(w, r, "Avatar deleted")
}
