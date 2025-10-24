package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"project/domain"
	"project/internal/service"
	"strconv"
	"strings"

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
	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Can't parse multipart form", http.StatusBadRequest)
		return
	}

	jsonProfile := r.FormValue("profile")
	if jsonProfile == "" {
		sendJSONSuccess(w, "Missing profile field", http.StatusBadRequest)
		return
	}

	var req domain.Profile
	if err := json.NewDecoder(strings.NewReader(jsonProfile)).Decode(&req); err != nil {
		sendJSONSuccess(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ok, err := govalidator.ValidateStruct(req)
	if !ok || err != nil {
		sendJSONSuccess(w, "Invalid data", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)
	// для надежности можно проверять что пользователь существует, но по идее если есть сессия с id пользвателем он точно должен существовать
	//isUserExist, err := api.userStore.IsUserExists(userID)
	//if err != nil {
	//    sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
	//    return
	//}
	//if !isUserExist {
	//    sendJSONSuccess(w, "User does not exist", http.StatusBadRequest)
	//    return
	//}

	files := r.MultipartForm.File["avatar"]
	if len(files) == 1 {
		avatarOldPath, err := api.profileStore.GetAvatarByUserID(userID)
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		newfilePath, err := service.HandleFileUpload(files, []*string{avatarOldPath})
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = api.profileStore.UpdateAvatar(newfilePath[0], userID)
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	} else {
		log.Println("Missing avatar field")
	}

	err = api.profileStore.UpdateProfile(req, userID)
	if err != nil {
		sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sendJSONSuccess(w, "Profile updated", http.StatusOK)
}

func (api *ProfileHandler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {

	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Can't parse multipart form", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	files := r.MultipartForm.File["avatar"]
	if len(files) == 1 {
		avatarOldPath, err := api.profileStore.GetAvatarByUserID(userID)
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		newfilePath, err := service.HandleFileUpload(files, []*string{avatarOldPath})
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = api.profileStore.UpdateAvatar(newfilePath[0], userID)
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	} else {
		sendJSONSuccess(w, "Missing avatar field", http.StatusBadRequest)
		log.Println("Missing avatar field")
		return
	}

	sendJSONSuccess(w, "Avatar updated", http.StatusOK)
}

func (api *ProfileHandler) UpdateHeader(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Can't parse multipart form", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(domain.UserIDKey).(int)

	files := r.MultipartForm.File["header"]
	if len(files) == 1 {
		headerOldPath, err := api.profileStore.GetHeaderByUserID(userID)
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		newfilePath, err := service.HandleFileUpload(files, []*string{headerOldPath})
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = api.profileStore.UpdateHeader(newfilePath[0], userID)
		if err != nil {
			sendJSONSuccess(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	} else {
		sendJSONSuccess(w, "Missing header field", http.StatusBadRequest)
		log.Println("Missing header field")
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
