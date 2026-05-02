package handler

import (
	"encoding/json"
	"errors"
	"golang-api/authorization"
	"golang-api/model"
	"log"
	"net/http"
)

type login struct {
	storage Storage
}

func newLogin(s Storage) login {
	return login{s}
}

func (l *login) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responseJSON(w, http.StatusMethodNotAllowed,
			newResponse(Error, "Method not allowed", nil))
		return
	}

	data := model.Login{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		responseJSON(w, http.StatusBadRequest,
			newResponse(Error, "Invalid request body", nil))
		return
	}

	if err := validate.Struct(data); err != nil {
		responseJSON(w, http.StatusBadRequest,
			newResponse(Error, "Validation errors", nil))
		return // ← faltaba acá
	}

	// Validación de credenciales contra BD.
	// ErrInvalidCredentials = email o password mal → 401.
	// Cualquier otro error = falla de infraestructura → 500.
	if err := l.storage.IsLoginValid(data); err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			responseJSON(w, http.StatusUnauthorized,
				newResponse(Error, err.Error(), nil))
			return
		}
		log.Printf("login: validating credentials: %v", err)
		responseJSON(w, http.StatusInternalServerError,
			newResponse(Error, "Internal error", nil))
		return
	}

	token, err := authorization.GenerateToken(&data)
	if err != nil {
		log.Printf("login: generating token: %v", err)
		responseJSON(w, http.StatusInternalServerError,
			newResponse(Error, "Could not generate token", nil))
		return
	}

	responseJSON(w, http.StatusOK,
		newResponse(Message, "Login OK", token))
}
