package handler

import (
	"errors"
	"golang-api/model"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type signup struct {
	storage Storage
}

func newSignup(s Storage) signup {
	return signup{storage: s}
}

func (s *signup) signup(c echo.Context) error {
	data := model.Signup{}
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest,
			newResponse(Error, "Invalid request body", nil))
	}

	if err := validate.Struct(data); err != nil {
		return c.JSON(http.StatusBadRequest,
			newResponse(Error, "Validation errors", nil))
	}

	// Hashing en el handler, NO en el repo. El repo recibe el User con
	// password ya hasheado y solo se preocupa por persistir.
	hash, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("signup: hashing password: %v", err)
		return c.JSON(http.StatusInternalServerError,
			newResponse(Error, "Internal error", nil))
	}

	user := model.User{Email: data.Email, Password: string(hash)}
	if err := s.storage.CreateUser(&user); err != nil {
		if errors.Is(err, model.ErrEmailAlreadyExists) {
			return c.JSON(http.StatusConflict,
				newResponse(Error, err.Error(), nil))
		}
		log.Printf("signup: creating user: %v", err)
		return c.JSON(http.StatusInternalServerError,
			newResponse(Error, "Internal error", nil))
	}

	// Devolvemos solo el ID — NUNCA el password ni el hash.
	return c.JSON(http.StatusCreated,
		newResponse(Message, "User created", map[string]any{"id": user.ID}))
}
