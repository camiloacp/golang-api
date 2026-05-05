package handler

import (
	"errors"
	"golang-api/authorization"
	"golang-api/model"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

type login struct {
	storage Storage
}

func newLogin(s Storage) login {
	return login{s}
}

func (l *login) login(c echo.Context) error {
	data := model.Login{}
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, newResponse(Error, "Invalid request body", nil))
	}

	if err := validate.Struct(data); err != nil {
		return c.JSON(http.StatusBadRequest, newResponse(Error, "Validation errors", nil))
	}

	// Validación de credenciales contra BD.
	// ErrInvalidCredentials = email o password mal → 401.
	// Cualquier otro error = falla de infraestructura → 500.
	if err := l.storage.IsLoginValid(data); err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, newResponse(Error, err.Error(), nil))
		}
		log.Printf("login: validating credentials: %v", err)
		return c.JSON(http.StatusInternalServerError, newResponse(Error, "Internal error", nil))
	}

	token, err := authorization.GenerateToken(&data)
	if err != nil {
		log.Printf("login: generating token: %v", err)
		return c.JSON(http.StatusInternalServerError, newResponse(Error, "Could not generate token", nil))
	}

	return c.JSON(http.StatusOK, newResponse(Message, "Login OK", model.LoginToken{Token: token}))
}
