package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"project/domain"
)

type JSONResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func sendJSONResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(JSONResponse{
		Message: message,
		Code:    statusCode,
	}); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func sendJSONError(w http.ResponseWriter, err error) {
	code, msg := domain.MapErrorToHTTP(err)
	sendJSONResponse(w, msg, code)
}

var NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
	sendJSONResponse(w, "Not found", http.StatusNotFound)
}
