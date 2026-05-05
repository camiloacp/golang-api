package middleware

import (
	"golang-api/authorization"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/labstack/echo/v4"
)

// Authentication valida el JWT del header "Authorization: Bearer <token>".
// Si falta, está malformado o es inválido, responde 401 sin invocar al siguiente handler.
func Authentication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// El estándar (RFC 6750) define el formato: "Authorization: Bearer <token>".
		// Hay que stripear el prefijo "Bearer " antes de validar; jwt.ParseWithClaims
		// espera el JWT crudo, sin prefijo.
		const bearerPrefix = "Bearer "
		header := c.Request().Header.Get("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			log.Printf("auth: missing or malformed Authorization header (path=%q)", c.Path())
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Token inválido"})
		}
		token := strings.TrimPrefix(header, bearerPrefix)

		if _, err := authorization.ValidateToken(token); err != nil {
			log.Printf("auth: validating token (path=%q): %v", c.Path(), err)
			return echo.ErrForbidden
		}

		return next(c)
	}
}

// Recover atrapa panics en handlers o middlewares posteriores, loguea el
// stack trace y responde 500. Debe ser el middleware más externo de la cadena.
//
// Nota: echo/v4/middleware.Recover() cumple la misma función con más opciones.
func Recover(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v\n%s", rec, debug.Stack())
				c.JSON(http.StatusInternalServerError,
					echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error"))
			}
		}()
		return next(c)
	}
}
