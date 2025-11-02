package handler

import (
	"encoding/json"
	"net/http"
	"project/domain"
	"project/internal/service"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type ProfileHandler struct {
	profileService service.ProfileService
}

func NewProfileHandler(profileService service.ProfileService) *ProfileHandler {
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
	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse multipart form", zap.Error(err))
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

	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	files := r.MultipartForm.File["avatar"]

	err = api.profileService.UpdateProfile(r.Context(), req, userID, files)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Profile updated", http.StatusOK)
	domain.FromContext(r.Context()).Info("Profile updated successfully")
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

	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse multipart form", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	files := r.MultipartForm.File["avatar"]

	err = api.profileService.UpdateAvatar(r.Context(), userID, files)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Avatar updated", http.StatusOK)
	domain.FromContext(r.Context()).Info("Avatar updated successfully")
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
	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Can't parse multipart form", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse multipart form", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	files := r.MultipartForm.File["header"]

	err = api.profileService.UpdateHeader(r.Context(), userID, files)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "Header updated", http.StatusOK)
	domain.FromContext(r.Context()).Info("Header updated successfully")
}

// GetProfileByUserID получает профиль пользователя по ID.
//
// @Summary Получение профиля по ID
// @Description Возвращает профиль пользователя по его ID.
// @Tags profile
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} domain.Profile "Профиль пользователя"
// @Failure 400 {string} string "Invalid user ID / User does not exist"
// @Failure 500 {string} string "Server error"
// @Security ApiKeyAuth
// @Router /profile/{id} [get]
func (api *ProfileHandler) GetProfileByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		sendJSONResponse(w, "Invalid user ID", http.StatusBadRequest)
		domain.FromContext(r.Context()).Error("Failed to parse user ID", zap.Error(err))
		return
	}

	profile, err := api.profileService.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		sendJSONError(w, err)
	}

	err = sendJSONData(r.Context(), w, profile)
	if err == nil {
		domain.FromContext(r.Context()).Info("Profile return successfully")
	}
}
