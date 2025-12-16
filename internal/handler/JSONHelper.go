package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"project/domain"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type JSONResponse struct {
	Message string `json:"message"`
	Code    int32  `json:"code"`
}

const DefaultMultipartMaxSize = 50 << 20 // 50 MB
func sendJSONResponse(w http.ResponseWriter, message string, statusCode int32) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(statusCode))

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

func sendJSONData(ctx context.Context, w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		domain.FromContext(ctx).Error(domain.FailToEncode, zap.Error(err), zap.String("struct", domain.StructName(data)))
	}
}

var NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
	sendJSONResponse(w, "Not found", http.StatusNotFound)
}

func PathInt32(r *http.Request, name string) (int32, error) {
	ctx := r.Context()

	val, ok := mux.Vars(r)[name]
	id, err := strconv.Atoi(val)

	if err != nil || !ok {
		domain.FromContext(ctx).Warn(
			"Failed to parse path param",
			zap.String("param", name),
			zap.String("value", val),
			zap.Error(err),
		)
		return 0, domain.ErrInvalidParams
	}

	return int32(id), nil
}

func sendJSONSuccess(w http.ResponseWriter, r *http.Request, msg string) {
	domain.FromContext(r.Context()).Info(msg)
	sendJSONResponse(w, msg, http.StatusOK)
}

func DecodeQueryParams[T any](r *http.Request) (T, error) {
	var q T

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)

	if err := decoder.Decode(&q, r.URL.Query()); err != nil {
		domain.Warn(
			r.Context(),
			"Invalid query parameters",
			zap.String("struct", domain.StructName(q)),
			zap.Error(err),
		)
		return q, domain.ErrInvalidParams
	}

	return q, nil
}

func DecodeJSONBody[T any](r *http.Request) (T, error) {
	var req T

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.FromContext(r.Context()).Error(
			domain.InvalidJSON,
			zap.String("struct", domain.StructName(req)),
			zap.Error(err),
		)
		return req, domain.ErrInvalidParams
	}

	return req, nil
}

func ParseMultipart(r *http.Request) error {
	if err := r.ParseMultipartForm(DefaultMultipartMaxSize); err != nil {
		domain.Error(
			r.Context(),
			"Failed to parse multipart form",
			err,
		)
		return domain.ErrInvalidParams
	}
	return nil
}

func parseIntParam(r *http.Request, paramName string) (*int32, error) {
	param := r.FormValue(paramName)
	if param == "" {
		return nil, nil
	}

	id, err := strconv.Atoi(param)
	if err != nil {
		domain.FromContext(r.Context()).Error(
			fmt.Sprintf("Failed to parse %s", paramName),
			zap.Error(err),
		)

		return nil, domain.ErrInvalidParams
	}

	val := int32(id)
	return &val, nil
}
