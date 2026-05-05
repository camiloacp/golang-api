# TODOs pendientes

Mejoras identificadas durante la implementación de auth y la migración a Echo. Ordenadas por impacto. Cada ítem incluye contexto suficiente para retomarlo en una sesión futura sin releer el chat.

---

## 1. Reglas de complejidad de password

**Estado actual**: `model/signup.go` valida solo `required,min=8,max=72`. No hay requisito de mayúsculas, dígitos, símbolos.

**Objetivo**: enforciar al menos 1 dígito + 1 mayúscula + 1 minúscula.

**Cómo encararlo**:

`go-playground/validator` no trae estas reglas built-in. Hay que registrar un validator custom una sola vez (idealmente en el `init()` del paquete handler, sobre la instancia compartida `validate`):

```go
validate.RegisterValidation("strongpwd", func(fl validator.FieldLevel) bool {
    s := fl.Field().String()
    var hasUpper, hasLower, hasDigit bool
    for _, r := range s {
        switch {
        case unicode.IsUpper(r): hasUpper = true
        case unicode.IsLower(r): hasLower = true
        case unicode.IsDigit(r): hasDigit = true
        }
    }
    return hasUpper && hasLower && hasDigit
})
```

Y en `model/signup.go`:

```go
Password string `json:"password" validate:"required,min=8,max=72,strongpwd"`
```

**Decisiones a tomar**:
- Solo aplica a signup (login solo valida formato).
- ¿Símbolo especial obligatorio? NIST SP 800-63B desaconseja reglas excesivas y prefiere longitud + blacklist de passwords comunes.
- Mensaje al cliente: extender `fieldErrorMessage` en `handler/person.go` para mapear el tag `strongpwd` a un texto amigable.

---

## 2. Refresh tokens para sesiones largas

**Estado actual**: `authorization/token.go` setea expiración en 1 hora. No hay refresh token, así que el usuario re-introduce el password cada hora.

**Objetivo**: agregar refresh tokens.

**Cómo encararlo**:

Patrón estándar access + refresh:
- **Access token**: corto (15min – 1h), va en cada request.
- **Refresh token**: largo (7–30 días), persistido, permite obtener nuevos access tokens.

Estructura:
1. Modelo `RefreshToken` con `UserID`, `Token` (random secure), `ExpiresAt`, `RevokedAt`.
2. `POST /v1/auth/refresh` recibe el refresh token, valida, emite access token nuevo.
3. `POST /v1/auth/logout` revoca el refresh token.
4. `/v1/login` devuelve ambos en el response.

**Decisiones**:
- ¿Body, header o cookie HttpOnly? Cookie HttpOnly es más segura contra XSS pero menos cómoda para clientes no-browser.
- ¿Token rotation (emitir refresh nuevo en cada uso)? Más seguro, requiere bookkeeping.

---

## 3. Auto-login post-signup

**Estado actual**: `POST /v1/signup` retorna `201 {"data":{"id": <uint>}}`. El cliente debe llamar a `/v1/login` después.

**Objetivo**: que signup retorne directamente el JWT.

**Cómo encararlo**:

En `handler/signup.go`, después del `s.storage.CreateUser(&user)`:

```go
token, err := authorization.GenerateToken(&model.Login{Email: data.Email, Password: data.Password})
if err != nil {
    log.Printf("signup: generating token: %v", err)
    // Devolver 201 igual: el usuario sí se creó, pero sin token.
    return c.JSON(http.StatusCreated,
        newResponse(Message, "User created (login required)", map[string]any{"id": user.ID}))
}

return c.JSON(http.StatusCreated,
    newResponse(Message, "User created", map[string]any{"id": user.ID, "token": token}))
```

**Trade-off**: signup pasa a depender del subsistema de tokens (un fallo de certificados rompe ambos endpoints). Hoy son independientes.

---

## 4. Rate limiting en `/signup` y `/login`

**Estado actual**: ningún endpoint tiene rate limiting. `/signup` permite registrar miles de cuentas en segundos; `/login` permite fuerza bruta.

**Objetivo**: limitar requests por IP (y/o email para `/login`).

**Cómo encararlo (con Echo)**:

Echo trae `middleware.RateLimiter` built-in. Implementación in-memory por IP:

```go
import echomw "github.com/labstack/echo/v4/middleware"

// En cmd/main.go, ANTES de registrar las rutas, o aplicado solo a signup/login:
loginRateLimiter := echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
    Store: echomw.NewRateLimiterMemoryStoreWithConfig(
        echomw.RateLimiterMemoryStoreConfig{
            Rate:      rate.Limit(5.0/60.0), // 5 por minuto
            Burst:     5,
            ExpiresIn: 3 * time.Minute,
        },
    ),
})
```

Y en `handler/route.go`, aplicarlo selectivamente:

```go
func RouteLogin(e *echo.Echo, storage Storage, rateLimiter echo.MiddlewareFunc) {
    h := newLogin(storage)
    e.POST("/v1/login", h.login, rateLimiter)
}
```

**Opciones por escala**:
1. **In-memory** (Echo built-in): suficiente para una sola instancia. Se pierde el estado al reiniciar.
2. **Redis-backed**: para múltiples instancias detrás de un load balancer. Hay que implementar un `Store` custom o usar paquetes como `go-redis-rate`.
3. **Servicio externo** (Cloudflare, AWS WAF): out-of-process.

**Decisiones**:
- ¿IP, email o ambos? IP es simple, email previene mejor brute force dirigido. Combinarlos es lo más seguro.
- Thresholds sugeridos:
  - `/login`: 5 intentos/min por IP, 3 fallidos/hora por email.
  - `/signup`: 3/hora por IP.

---

## 5. Tests automatizados

**Estado actual**: tests unitarios y de integración están **completamente ausentes**. Ningún archivo `*_test.go` en el repo.

**Objetivo**: cobertura mínima antes de escalar features.

**Cómo encararlo**:

Echo tiene un patrón muy testeable porque expone `e.ServeHTTP(rec, req)` directamente sin levantar un servidor:

```go
// handler/person_test.go
func TestPerson_Create(t *testing.T) {
    e := echo.New()
    mockStorage := &mockStorage{}  // implementa Storage
    h := newPerson(mockStorage)

    body := `{"name":"Test","age":30}`
    req := httptest.NewRequest(http.MethodPost, "/v1/persons/create", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    err := h.create(c)
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, rec.Code)
}
```

Estrategia por capa:
- **`handler/`**: con `httptest` + mock de `Storage`. Rápido, sin DB.
- **`storage/`**: con `testcontainers-go` levantando Postgres real. Más lento pero válida queries reales.
- **`authorization/`**: tests puros de `GenerateToken` / `ValidateToken` sin red.

---

## 6. Migrar `Logger` a `RequestLogger` con `slog` estructurado

**Estado actual**: `cmd/main.go` usa `echomw.RequestLoggerWithConfig` que loguea con `log.Printf` en formato `method=X uri=Y status=Z latency=W`. Es texto plano.

**Objetivo**: logs estructurados (JSON) que se puedan parsear con Datadog / Loki / Grafana.

**Cómo encararlo**:

Cambiar el `LogValuesFunc` actual por uno que use `log/slog` (stdlib desde Go 1.21):

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

e.Use(echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
    LogStatus:  true,
    LogURI:     true,
    LogMethod:  true,
    LogLatency: true,
    LogValuesFunc: func(c echo.Context, v echomw.RequestLoggerValues) error {
        logger.Info("http_request",
            slog.String("method", v.Method),
            slog.String("uri", v.URI),
            slog.Int("status", v.Status),
            slog.Duration("latency", v.Latency),
        )
        return nil
    },
}))
```

**Decisiones**:
- ¿`slog` con handler JSON o un wrapper como `zerolog`? `slog` es stdlib y suficiente para empezar; `zerolog` es más rápido en hot paths.
- ¿Filtrar health checks? Si en el futuro se agrega `/health`, agregar `if v.URI == "/health" { return nil }` para no inundar el log.

---

## 7. Eliminar el paquete `middleware/` (consolidación)

**Estado actual**: tras la migración a Echo, el paquete `middleware/` solo contiene `Authentication`. Es un paquete con una sola función.

**Objetivo**: mover `Authentication` a `authorization/middleware.go` y eliminar el paquete `middleware/`.

**Justificación**: la auth ya depende del paquete `authorization` (`authorization.ValidateToken`). Tener un paquete separado solo para envolverla es ruido organizacional.

**Cambios concretos**:
1. Mover `func Authentication(...)` de `middleware/middleware.go` a un nuevo `authorization/middleware.go`.
2. Actualizar imports en `handler/route.go`: `"golang-api/middleware"` → `"golang-api/authorization"` (con uso `authorization.Authentication`).
3. Borrar `middleware/middleware.go` y la carpeta vacía.

Bajo riesgo, alta limpieza.

---

## 8. URLs RESTful puras (`/v1/persons` vs `/v1/persons/create`)

**Estado actual**: el grupo `/v1/persons/*` mezcla estilo "action" (`/create`, `/get-all`) con path params idiomáticos (`/:id`).

**Objetivo**: full REST.

| Hoy                          | REST puro              |
| ---------------------------- | ---------------------- |
| `POST /v1/persons/create`    | `POST /v1/persons`     |
| `GET /v1/persons/get-all`    | `GET /v1/persons`      |
| `GET /v1/persons/:id`        | (igual)                |
| `PUT /v1/persons/:id`        | (igual)                |
| `DELETE /v1/persons/:id`     | (igual)                |

Echo distingue por método, así que `POST /v1/persons` y `GET /v1/persons` conviven sin conflicto.

**Cambio en `handler/route.go`**:

```go
g := e.Group("/v1/persons", middleware.Authentication)
g.POST("", h.create)         // POST /v1/persons
g.GET("", h.getAll)          // GET /v1/persons
g.GET("/:id", h.getByID)
g.PUT("/:id", h.update)
g.DELETE("/:id", h.delete)
```

**Es breaking change**: cualquier cliente actual (incluyendo la Postman collection) hay que actualizarlo.

---

## Notas generales

- **Bug del espacio en `validate:"min=8, max=72"`**: los tags de `go-playground/validator` no toleran espacios entre validators. Detectado por `Recover` middleware durante un panic. Vale agregar un comentario en `model/signup.go` recordándolo.
- **Service layer ausente**: `Login` y `Signup` comparten estructura casi idéntica (decode → validate → bcrypt/IsLoginValid → response). Si crece a 3+ endpoints de auth, considerar extraer un `auth/service.go` y dejar los handlers como adaptadores HTTP delgados.
- **DTOs vs entidades de BD**: `model.Person` y `model.User` se serializan tal cual incluyendo `gorm.Model` (`ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`). Cuando crezca la API, separar entidades de BD de DTOs de respuesta (ej. `PersonResponse`) para no filtrar internals del schema.
