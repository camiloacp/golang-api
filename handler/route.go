package handler

import (
	"golang-api/middleware"

	"github.com/labstack/echo/v4"
)

// RoutePerson registra las rutas para las personas.
//
// Orden de middlewares (de afuera hacia adentro): Recover → Log → [Authentication] → handler.
// Recover es el más externo para atrapar panics de cualquier middleware o del handler.
func RoutePerson(e *echo.Echo, storage Storage) {
	h := newPerson(storage)
	g := e.Group("/v1/persons", middleware.Authentication)
	g.POST("", h.create)
	g.GET("", h.getAll)
	g.GET("/:id", h.getByID)
	g.PUT("/:id", h.update)
	g.DELETE("/:id", h.delete)
}

// RouteLogin registra la ruta de autenticación. Es pública (sin Authentication).
func RouteLogin(e *echo.Echo, storage Storage) {
	h := newLogin(storage)
	e.POST("/v1/login", h.login)
}

// RouteSignup registra la ruta de registro. Es pública (sin Authentication).
func RouteSignup(e *echo.Echo, storage Storage) {
	h := newSignup(storage)
	e.POST("/v1/signup", h.signup)
}
