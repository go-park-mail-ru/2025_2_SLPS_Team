package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"project/domain"

	"go.uber.org/zap"
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

func sendJSONData(ctx context.Context, w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		sendJSONResponse(w, domain.ServerErr, http.StatusInternalServerError)
		domain.FromContext(ctx).Error(domain.FailToEncode, zap.Error(err), zap.String("struct", domain.StructName(data)))
		return err
	}
	return nil
}

var NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
	sendJSONResponse(w, "Not found", http.StatusNotFound)
}
