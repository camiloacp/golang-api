package handler

import (
	"encoding/json"
	"errors"
	"golang-api/model"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type signup struct {
	storage Storage
}

func newSignup(s Storage) signup {
	return signup{storage: s}
}

func (s *signup) signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responseJSON(w, http.StatusMethodNotAllowed,
			newResponse(Error, "Method not allowed", nil))
		return
	}

	data := model.Signup{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		responseJSON(w, http.StatusBadRequest,
			newResponse(Error, "Invalid request body", nil))
		return
	}

	if err := validate.Struct(data); err != nil {
		responseJSON(w, http.StatusBadRequest,
			newResponse(Error, "Validation errors", nil))
		return
	}

	// Hashing en el handler, NO en el repo. El repo recibe el User con
	// password ya hasheado y solo se preocupa por persistir.
	hash, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("signup: hashing password: %v", err)
		responseJSON(w, http.StatusInternalServerError,
			newResponse(Error, "Internal error", nil))
		return
	}

	user := model.User{Email: data.Email, Password: string(hash)}
	if err := s.storage.CreateUser(&user); err != nil {
		if errors.Is(err, model.ErrEmailAlreadyExists) {
			responseJSON(w, http.StatusConflict,
				newResponse(Error, err.Error(), nil))
			return
		}
		log.Printf("signup: creating user: %v", err)
		responseJSON(w, http.StatusInternalServerError,
			newResponse(Error, "Internal error", nil))
		return
	}

	// Devolvemos solo el ID — NUNCA el password ni el hash.
	responseJSON(w, http.StatusCreated,
		newResponse(Message, "User created", map[string]any{"id": user.ID}))
}
