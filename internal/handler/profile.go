package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"project/domain"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
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

func (api *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req domain.Profile
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONSuccess(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(userIDKey).(int)
	log.Println(userID)
	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		sendJSONSuccess(w, "Invalid data", http.StatusBadRequest)
		return
	}

	err = api.profileStore.UpdateProfile(req, userID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Profile updated", http.StatusOK)
}

type UpdateAvatarRequest struct {
	AvatarPath string `json:"avatarPath"`
}

func (api *ProfileHandler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	var req UpdateAvatarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONSuccess(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(userIDKey).(int)
	err := api.profileStore.UpdateAvatar(req.AvatarPath, userID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Avatar updated", http.StatusOK)
}

type UpdateHeaderRequest struct {
	HeaderPath string `json:"headerPath"`
}

func (api *ProfileHandler) UpdateHeader(w http.ResponseWriter, r *http.Request) {
	var req UpdateHeaderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONSuccess(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(userIDKey).(int)
	err := api.profileStore.UpdateHeader(req.HeaderPath, userID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Header updated", http.StatusOK)

}

func (api *ProfileHandler) GetProfileByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		sendJSONSuccess(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var profile domain.Profile
	profile, err = api.profileStore.GetProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			sendJSONSuccess(w, "User does`not exist", http.StatusBadRequest)
			return
		}
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)

		return
	}
}
