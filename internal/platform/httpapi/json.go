package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func DecodeJSON(r *http.Request, target any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func WriteError(w http.ResponseWriter, status int, code string, err error) {
	message := "request failed"
	if err != nil {
		message = err.Error()
	}
	WriteJSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}
func ParseLabels(values []string) map[string]string {
	result := map[string]string{}
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}
