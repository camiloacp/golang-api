# TODOs pendientes

Mejoras identificadas durante la implementación de la feature de autenticación, ordenadas por impacto. Cada ítem incluye contexto suficiente para retomarlo en una sesión futura sin necesidad de releer el chat.

---

## 1. Reglas de complejidad de password

**Estado actual**: `model/signup.go` valida solo `required,min=8,max=72` con `go-playground/validator`. No hay requisito de mayúsculas, dígitos, símbolos, etc.

**Objetivo**: enforciar al menos 1 dígito + 1 mayúscula + 1 minúscula (o el set que se decida).

**Cómo encararlo**:

`go-playground/validator` no trae estas reglas built-in. Hay que registrar un validator custom:

```go
// en algún init() del paquete handler o en un helper:
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

**Decisiones a tomar antes**:
- ¿Se aplica también al `Login`? Probablemente no (login solo valida formato; las reglas de complejidad solo se chequean al crear/cambiar password).
- ¿Se requiere símbolo especial? Subjetivo — NIST SP 800-63B desaconseja reglas de complejidad excesivas y prefiere longitud + blacklist de passwords comunes. Considerar.
- Mensaje de error al cliente: "password must contain at least one uppercase, one lowercase, and one digit".

---

## 2. Refresh tokens para sesiones largas

**Estado actual**: `authorization/token.go:15` setea `ExpiresAt` a `time.Now().Add(time.Hour * 1)`. La duración del access token quedó en 1h (subida desde los 5 min iniciales). Falta el mecanismo de refresh para evitar que el usuario re-introduzca el password cada hora.

**Objetivo**: agregar refresh tokens para extender sesiones sin pedir credenciales nuevamente.

**Cómo encararlo**:

Patrón estándar de access + refresh:
- **Access token**: corto (15min - 1h), va en cada request, se descarta si expira.
- **Refresh token**: largo (7-30 días), se guarda persistido (en BD o cookie HttpOnly), permite obtener nuevos access tokens sin pedir password.

Estructura aproximada:
1. Nuevo modelo `RefreshToken` en `model/` con `UserID`, `Token` (random secure), `ExpiresAt`, `RevokedAt`.
2. Endpoint `POST /v1/auth/refresh` que recibe el refresh token, valida que no esté expirado/revocado, y emite un nuevo access token.
3. Endpoint `POST /v1/auth/logout` que revoca el refresh token (set `RevokedAt = NOW()`).
4. En `/v1/login`, devolver ambos tokens en el response.

**Decisiones**:
- ¿Refresh token va en body, header, o cookie HttpOnly? Cookie HttpOnly es más seguro contra XSS pero menos cómodo para clientes no-browser.
- ¿Se permite reutilización de refresh token o token rotation? Rotation (emitir un refresh token nuevo cada uso) es más seguro pero requiere más bookkeeping.

---

## 3. Auto-login post-signup

**Estado actual**: `POST /v1/signup` retorna `201 {"message_type":"message","message":"User created","data":{"id": <uint>}}`. El cliente debe llamar a `/v1/login` después.

**Objetivo**: que el signup retorne directamente el JWT como hace login, ahorrando un round-trip.

**Cómo encararlo**:

En `handler/signup.go`, después del `s.storage.CreateUser(&user)` exitoso:

```go
token, err := authorization.GenerateToken(&model.Login{Email: data.Email, Password: data.Password})
if err != nil {
    log.Printf("signup: generating token: %v", err)
    // Decisión: ¿200 con error o 201 sin token? Probablemente 201 con campo
    // vacío para indicar que el usuario sí se creó pero hay que loguear.
}

responseJSON(w, http.StatusCreated,
    newResponse(Message, "User created", map[string]any{"id": user.ID, "token": token}))
```

**Trade-off**: signup ahora depende del subsistema de tokens, así que un fallo de certificados rompe signup también. Hoy son endpoints independientes — esto los acopla.

---

## 4. Rate limiting en `/signup` y `/login`

**Estado actual**: ningún endpoint tiene rate limiting. `/signup` permite registrar miles de cuentas en segundos; `/login` permite fuerza bruta de passwords.

**Objetivo**: limitar requests por IP (y/o por email en el caso de `/login`).

**Cómo encararlo**:

Opciones de menor a mayor complejidad:

1. **Middleware in-memory por IP** con `golang.org/x/time/rate.Limiter`. Suficiente para una sola instancia. Se pierde el estado al reiniciar.
2. **Redis-backed** con un counter por `IP+endpoint+window`. Funciona con múltiples instancias detrás de un load balancer.
3. **Servicio externo** (Cloudflare, AWS WAF) — out-of-process.

Para empezar, opción 1:

```go
// middleware/ratelimit.go
func RateLimit(rps float64, burst int) http.HandlerFunc {
    limiters := sync.Map{} // map[string]*rate.Limiter por IP
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            ip, _, _ := net.SplitHostPort(r.RemoteAddr)
            l, _ := limiters.LoadOrStore(ip, rate.NewLimiter(rate.Limit(rps), burst))
            if !l.(*rate.Limiter).Allow() {
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            next(w, r)
        }
    }
}
```

Y aplicarlo en `route.go` con `Chain(... RateLimit(1, 5), ...)` para `/login` y `/signup`.

**Decisiones**:
- ¿Por IP o por email? Por IP es más simple, por email previene mejor fuerza bruta dirigida.
- ¿Qué thresholds? Para `/login`: ~5 intentos por minuto por IP. Para `/signup`: ~3 por hora por IP.

---

## 5. Limpieza de la sección "Arquitectura" del README

**Estado actual**: `README.md` líneas 5-29 describen la arquitectura con paths que ya no existen:
- Menciona `main.go` en raíz → ahora está en `cmd/main.go`.
- Menciona `pkg/person/` → ese paquete ya no existe.
- Menciona `memory.go` (repositorio en memoria) → no existe.
- No menciona `handler/`, `middleware/`, `authorization/`, `cmd/seed/`.

**Objetivo**: que la documentación refleje el estado real del proyecto.

**Cómo encararlo**:

Reescribir la sección "Arquitectura" con la estructura real:

```
golang-api/
├── cmd/
│   ├── main.go              # Entrypoint: certificados + DB + routers + server
│   └── seed/
│       └── main.go          # Binario one-shot para crear usuarios admin
├── model/
│   ├── login.go             # DTO Login + Claim + sentinels de auth
│   ├── signup.go            # DTO Signup
│   ├── user.go              # Entidad User (BD)
│   ├── person.go            # Entidades Person + Community
│   └── model.go             # Errores de dominio
├── handler/
│   ├── handler.go           # Interfaz Storage
│   ├── login.go             # POST /v1/login
│   ├── signup.go            # POST /v1/signup
│   ├── person.go            # CRUD de personas
│   ├── route.go             # Registro de rutas con middlewares
│   └── response.go          # Envelope JSON
├── middleware/
│   └── middleware.go        # Log, Authentication, Recover, Chain
├── authorization/
│   ├── token.go             # GenerateToken / ValidateToken (JWT RS256)
│   └── certificates.go      # Carga claves RSA desde PEM
└── storage/
    ├── storage.go           # Store + fachadas + AutoMigrate
    ├── gorm_person.go       # CRUD Person + Community
    └── gorm_login.go        # IsLoginValid + Create User
```

Y actualizar también la tabla de "Capas" — sacar `pkg/person/` que no existe.

---

## Notas generales

- El bug del **espacio en `validate:"min=8, max=72"`** que costó tiempo de debug (panic → empty reply) está documentado implícitamente en la existencia del middleware `Recover`. Vale la pena agregar un comentario en `model/signup.go` recordando que los tags no toleran espacios entre validators.
- El handler de `Login` y `Signup` comparten estructura casi idéntica (decode → validate → bcrypt/IsLoginValid → response). Si crece a 3+ endpoints similares, considerar un **service layer** que encapsule la lógica común y deje los handlers como adaptadores HTTP delgados.
- Tests unitarios y de integración están **completamente ausentes**. Antes de escalar features, vale la pena introducir al menos:
  - Tests de `storage/` con un Postgres en `testcontainers`.
  - Tests de handler con `httptest` y un `Storage` mock.
