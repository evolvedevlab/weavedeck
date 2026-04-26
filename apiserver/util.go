package apiserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type ApiHandlerFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewApiError(status int, msg string, data any) *ApiError {
	return &ApiError{
		Status:  status,
		Message: msg,
		Data:    data,
	}
}

func (e ApiError) Error() string {
	return fmt.Sprintf("[%d]: %+v", e.Status, e.Message)
}

func NewBadRequestError(msg string, data any) *ApiError {
	return &ApiError{
		Status:  http.StatusBadRequest,
		Message: msg,
		Data:    data,
	}
}

func NewInternalServerError(msg string, data any) *ApiError {
	return &ApiError{
		Status:  http.StatusInternalServerError,
		Message: msg,
		Data:    data,
	}
}

func NewUnauthoziedError(msg string, data any) *ApiError {
	return &ApiError{
		Status:  http.StatusUnauthorized,
		Message: msg,
		Data:    data,
	}
}

func handler(fn ApiHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err == nil {
			return
		}

		slog.Error("http_request", "method", r.Method, "path", r.URL.Path, "err", err.Error())

		var apiErr *ApiError
		if errors.As(err, &apiErr) {
			writeJSON(w, apiErr.Status, apiErr)
		} else {
			writeJSON(w, http.StatusInternalServerError, ApiError{
				Status:  http.StatusInternalServerError,
				Message: "internal server error",
			})
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
