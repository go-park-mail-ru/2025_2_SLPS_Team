package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

type JSONResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func sendJSONSuccess(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(JSONResponse{
		Message: message,
		Code:    statusCode,
	}); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

var NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
	sendJSONSuccess(w, "Not found", http.StatusNotFound)
}
