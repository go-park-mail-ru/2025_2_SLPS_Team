package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"project/domain"
	"project/internal/service"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type ProfileHandler struct {
	profileStore domain.ProfileStore
	userStore    domain.UserStore
}

func NewProfileHandler(profileStore domain.ProfileStore, userStore domain.UserStore) *ProfileHandler {
	return &ProfileHandler{
		profileStore: profileStore,
		userStore:    userStore,
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
		service.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	jsonProfile := r.FormValue("profile")
	if jsonProfile == "" {
		sendJSONResponse(w, "Missing profile field", http.StatusBadRequest)
		service.Warn(r.Context(), "Missing profile field in request")
		return
	}

	var req domain.Profile
	if err := json.NewDecoder(strings.NewReader(jsonProfile)).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		service.Error(r.Context(), "Failed to decode profile JSON", err)
		return
	}

	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		sendJSONResponse(w, domain.InvalidData, http.StatusBadRequest)
		service.Warn(r.Context(), "Profile validation failed")
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	// для надежности можно проверять что пользователь существует, но по идее если есть сессия с id пользвателем он точно должен существовать
	//isUserExist, err := api.userStore.IsUserExists(userID)
	//if err != nil {
	//    sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
	//    return
	//}
	//if !isUserExist {
	//    sendJSONResponse(w, "User does not exist", http.StatusBadRequest)
	//    return
	//}

	files := r.MultipartForm.File["avatar"]
	if len(files) == 1 {
		avatarOldPath, err := api.profileStore.GetAvatarByUserID(r.Context(), userID)
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to get old avatar path", err)
			return
		}

		newfilePath, err := service.HandleFileUpload(files, []*string{avatarOldPath})
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to upload new avatar", err)
			return
		}

		err = api.profileStore.UpdateAvatar(r.Context(), newfilePath[0], userID)
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to update avatar", err)
			return
		}

	} else {
		service.Warn(r.Context(), "Missing avatar field")
	}

	err = api.profileStore.UpdateProfile(r.Context(), req, userID)
	if err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to update profile", err)
		return
	}
	service.Info(r.Context(), "Profile updated successfully")
	sendJSONResponse(w, "Profile updated", http.StatusOK)
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
		service.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	files := r.MultipartForm.File["avatar"]
	if len(files) == 1 {
		avatarOldPath, err := api.profileStore.GetAvatarByUserID(r.Context(), userID)
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to get old avatar path", err)
			return
		}

		newfilePath, err := service.HandleFileUpload(files, []*string{avatarOldPath})
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to upload avatar", err)
			return
		}

		err = api.profileStore.UpdateAvatar(r.Context(), newfilePath[0], userID)
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to update avatar", err)
			return
		}

	} else {
		sendJSONResponse(w, "Missing avatar field", http.StatusBadRequest)
		service.Warn(r.Context(), "Missing avatar field in request")
		return
	}

	sendJSONResponse(w, "Avatar updated", http.StatusOK)
	service.Info(r.Context(), "Avatar updated successfully")
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
		service.Error(r.Context(), "Failed to parse multipart form", err)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	files := r.MultipartForm.File["header"]
	if len(files) == 1 {
		headerOldPath, err := api.profileStore.GetHeaderByUserID(r.Context(), userID)
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to get old header path", err)
			return
		}

		newfilePath, err := service.HandleFileUpload(files, []*string{headerOldPath})
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to upload header", err)
			return
		}

		err = api.profileStore.UpdateHeader(r.Context(), newfilePath[0], userID)
		if err != nil {
			sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
			service.Error(r.Context(), "Failed to update header", err)
			return
		}

	} else {
		sendJSONResponse(w, "Missing header field", http.StatusBadRequest)
		service.Warn(r.Context(), "Missing header field in request")
		return
	}

	sendJSONResponse(w, "Header updated", http.StatusOK)
	service.Info(r.Context(), "Header updated successfully")
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
		service.Error(r.Context(), "Failed to parse user ID", err)
		return
	}

	var profile domain.Profile
	profile, err = api.profileStore.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			sendJSONResponse(w, "User does`not exist", http.StatusBadRequest)
			service.Warn(r.Context(), "User not found", zap.Int("userID", userID))
			return
		}
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), "Failed to get profile", err, zap.Int("userID", userID))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		service.Error(r.Context(), domain.FailToEncode, err, zap.String("struct", service.StructName(profile)))
		return
	}
}
