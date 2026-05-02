package handler

import (
	"encoding/json"
	"net/http"
)

const (
	Error   = "error"
	Message = "message"
)

type response struct {
	MessageType string      `json:"message_type"`
	Message     string      `json:"message"`
	Data        interface{} `json:"data,omitempty"`
}

func newResponse(messageType string, message string, data interface{}) response {
	return response{
		MessageType: messageType,
		Message:     message,
		Data:        data,
	}
}

func responseJSON(w http.ResponseWriter, statusCodde int, response response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCodde)
	err := json.NewEncoder(w).Encode(&response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
