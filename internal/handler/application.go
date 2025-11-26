package handler

import (
	"encoding/json"
	"net/http"
	"project/domain"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type ApplicationHandler struct {
	applicationService domain.ApplicationService
}

func NewApplicationHandler(ApplicationService domain.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{
		applicationService: ApplicationService,
	}
}

type ApplicationIDResponse struct {
	ApplicationID int32 `json:"applicationID"`
}

// CreateApplication
// @Summary Create a new support application
// @Description Creates a new support application. Can be created by a registered user or a temp session.
// @Tags Applications
// @Accept json
// @Produce json
// @Param application body domain.Application true "Application payload"
// @Success 200 {object} handler.ApplicationIDResponse
// @Failure 400 {object} JSONResponse "Invalid JSON or missing fields"
// @Failure 500 {object} JSONResponse "Internal server error"
// @Router /applications [post]
func (api *ApplicationHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var req domain.Application

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, domain.InvalidJSON, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(req)))
		return
	}
	TempSessionInfo, _ := r.Context().Value(domain.TempSessionCtxKey).(*domain.TempSessionInfo)
	if TempSessionInfo == nil {
		TempSessionInfo = &domain.TempSessionInfo{}
	}
	if TempSessionInfo.TempSessionID == nil && TempSessionInfo.UserID == nil {
		newID := uuid.New()
		TempSessionInfo.TempSessionID = &newID
		sessionCookie := &http.Cookie{
			Name:     "temp_session_id",
			Value:    (*TempSessionInfo.TempSessionID).String(),
			Path:     "/",
			Expires:  time.Now().Add(10 * time.Hour),
			HttpOnly: true,
		}

		http.SetCookie(w, sessionCookie)
	}

	id, err := api.applicationService.CreateApplication(r.Context(), req)
	if err != nil {
		sendJSONError(w, err)
		return
	}
	err = sendJSONData(r.Context(), w, ApplicationIDResponse{ApplicationID: id})
}

// GetApplications
// @Summary List support applications
// @Description Returns paginated list of applications. Admins see all, normal users see only theirs.
// @Tags Applications
// @Accept json
// @Produce json
// @Param page query int32 false "Page number"
// @Param limit query int32 false "Items per page"
// @Success 200 {array} domain.Application
// @Failure 400 {object} JSONResponse "Invalid query parameters"
// @Failure 500 {object} JSONResponse "Internal server error"
// @Router /applications [get]
func (h *ApplicationHandler) GetApplications(w http.ResponseWriter, r *http.Request) {
	var qParams domain.PaginateQueryParams
	if err := schema.NewDecoder().Decode(&qParams, r.URL.Query()); err != nil {
		sendJSONResponse(w, domain.InvalidParams, http.StatusBadRequest)
		domain.FromContext(r.Context()).Error(domain.InvalidJSON, zap.Error(err), zap.String("struct", domain.StructName(qParams)))
		return
	}

	apps, err := h.applicationService.GetApplications(r.Context(), qParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

type updateTextRequest struct {
	Text string `json:"text"`
}

// UpdateApplicationText godoc
// @Summary Update the text of an application
// @Description Updates the text of an existing support application by ID.
// @Tags Applications
// @Accept json
// @Produce json
// @Param id path int32 true "Application ID"
// @Param text body handler.updateTextRequest true "New text"
// @Success 204 "No Content"
// @Failure 400 {object} JSONResponse "Invalid ID or request body"
// @Failure 500 {object} JSONResponse "Internal server error"
// @Router /applications/{id}/text [put]
func (h *ApplicationHandler) UpdateApplicationText(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	var req updateTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, err)
		return
	}

	if err := h.applicationService.UpdateApplicationText(r.Context(), int32(id), req.Text); err != nil {
		sendJSONError(w, err)
		return
	}

	sendJSONResponse(w, "main updated", http.StatusOK)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateApplicationStatus godoc
// @Summary Update the status of an application
// @Description Updates the status of an existing support application by ID.
// @Tags Applications
// @Accept json
// @Produce json
// @Param id path int32 true "Application ID"
// @Param status body handler.updateStatusRequest true "New status"
// @Success 204 "No Content"
// @Failure 400 {object} JSONResponse "Invalid ID or request body"
// @Failure 500 {object} JSONResponse "Internal server error"
// @Router /applications/{id}/status [put]
func (h *ApplicationHandler) UpdateApplicationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendJSONError(w, err)
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, err)
		return
	}

	if err := h.applicationService.UpdateApplicationStatus(r.Context(), int32(id), req.Status); err != nil {
		sendJSONError(w, err)
		return
	}
	sendJSONResponse(w, "main updated", http.StatusOK)
}
