package middleware

import (
	"golang-api/authorization"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
)

func Log(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("peticion: %q, método: %q", r.URL.Path, r.Method)
		f(w, r)
	}
}

func Authentication(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// El estándar (RFC 6750) define el formato: "Authorization: Bearer <token>".
		// Hay que stripear el prefijo "Bearer " antes de validar; jwt.ParseWithClaims
		// espera el JWT crudo, sin prefijo.
		const bearerPrefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			log.Printf("auth: missing or malformed Authorization header (path=%q)", r.URL.Path)
			forbidden(w, r)
			return
		}
		token := strings.TrimPrefix(header, bearerPrefix)

		_, err := authorization.ValidateToken(token)
		if err != nil {
			log.Printf("auth: validating token (path=%q): %v", r.URL.Path, err)
			forbidden(w, r)
			return
		}

		f(w, r)
	}
}

func forbidden(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"message": "Token inválido"}`))
}

func Recover(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v\n%s", rec, debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

// Chain compone middlewares aplicándolos de afuera hacia adentro.
//
// Chain(h, A, B, C) es equivalente a A(B(C(h))): cuando llega un request,
// primero corre A, que delega en B, que delega en C, que delega en h.
// Por eso el primer middleware (A) es el "más externo" y atrapa cualquier
// cosa que pase en los siguientes — usalo para Recover.
func Chain(handler http.HandlerFunc, mws ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}
