package handler

import (
	"golang-api/middleware"
	"net/http"
)

// RoutePerson registra las rutas para las personas.
//
// Orden de middlewares (de afuera hacia adentro): Recover → Log → [Authentication] → handler.
// Recover es el más externo para atrapar panics de cualquier middleware o del handler.
func RoutePerson(mux *http.ServeMux, storage Storage) {
	h := newPerson(storage)
	mux.HandleFunc("POST /v1/persons/create",
		middleware.Chain(h.create, middleware.Recover, middleware.Log, middleware.Authentication))
	mux.HandleFunc("GET /v1/persons/get-all",
		middleware.Chain(h.getAll, middleware.Recover, middleware.Log))
	mux.HandleFunc("PUT /v1/persons/update",
		middleware.Chain(h.update, middleware.Recover, middleware.Log))
	mux.HandleFunc("DELETE /v1/persons/delete",
		middleware.Chain(h.delete, middleware.Recover, middleware.Log))
	mux.HandleFunc("GET /v1/persons/get-by-id",
		middleware.Chain(h.getByID, middleware.Recover, middleware.Log))
}

// RouteLogin registra la ruta de autenticación.
func RouteLogin(mux *http.ServeMux, storage Storage) {
	h := newLogin(storage)
	mux.HandleFunc("POST /v1/login",
		middleware.Chain(h.login, middleware.Recover, middleware.Log))
}

// RouteSignup registra la ruta de registro de usuarios.
// Es pública: NO va envuelta en middleware.Authentication, por la misma
// razón que /v1/login — sería un huevo-y-gallina pedir token para crear cuenta.
func RouteSignup(mux *http.ServeMux, storage Storage) {
	h := newSignup(storage)
	mux.HandleFunc("POST /v1/signup",
		middleware.Chain(h.signup, middleware.Recover, middleware.Log))
}
