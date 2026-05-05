# golang-api

Servidor HTTP en Go con persistencia en PostgreSQL usando GORM. Soporta también MySQL mediante un switch de engine. El ruteo y middleware se manejan con [Echo v4](https://echo.labstack.com/).

## Arquitectura

```
golang-api/
├── cmd/
│   ├── main.go              # Entrypoint: certificados + DB + Echo + server
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
│   ├── person.go            # CRUD de personas (handlers Echo)
│   ├── route.go             # Registro de rutas con e.Group + middlewares
│   └── response.go          # Envelope JSON (newResponse)
├── middleware/
│   └── middleware.go        # Authentication (JWT) — Recover/Logger los provee Echo
├── authorization/
│   ├── token.go             # GenerateToken / ValidateToken (JWT RS256)
│   └── certificates.go      # Carga claves RSA desde PEM
└── storage/
    ├── storage.go           # Store + fachadas + AutoMigrate
    ├── gorm_person.go       # CRUD Person + Community
    └── gorm_login.go        # IsLoginValid + Create User
```

### Capas

| Capa            | Paquete           | Responsabilidad                                              |
| --------------- | ----------------- | ------------------------------------------------------------ |
| Datos           | `model/`          | Definición de structs, errores y DTOs                        |
| Persistencia    | `storage/`        | Acceso a BD (GORM PostgreSQL/MySQL)                          |
| Transporte HTTP | `handler/`        | Handlers Echo, ruteo y serialización JSON                    |
| Auth            | `authorization/`  | Firma y validación de JWT con RSA                            |
| Middleware      | `middleware/`     | `Authentication` (JWT bearer); el resto lo provee Echo       |

### Relación Person → Communities

`Person` tiene muchas `Community` (has-many). GORM enlaza ambas tablas mediante la columna `person_id` en `communities`.

```
tabla persons                  tabla communities
─────────────────              ──────────────────────────
id  | name  | age              id | person_id | name
────┼───────┼────              ───┼───────────┼──────────
1   | Camilo| 30               1  |     1     | EDteam
2   | Ana   | 25               2  |     1     | Golang
                               3  |     2     | Python
```

---

## Requisitos

- Go 1.25+
- Docker y Docker Compose (opcional, para correr con Postgres en contenedor)

---

## Ejecución con Docker

```bash
docker-compose up --build
```

Levanta dos servicios:

- `postgres` — imagen `postgres:17-alpine`, puerto `5432`
- `app` — la API compilada, puerto `8080`

La app espera a que PostgreSQL esté listo (healthcheck) antes de arrancar. Las tablas se crean automáticamente vía `AutoMigrate`.

---

## Ejecución local

Requiere una instancia de PostgreSQL corriendo en `localhost:5432` y los certificados RSA generados (ver sección **Autenticación**).

```bash
cp .env.example .env
# edita .env con tus credenciales

go run ./cmd
```

---

## Variables de entorno

| Variable      | Default      | Descripción                          |
| ------------- | ------------ | ------------------------------------ |
| `DB_HOST`     | `localhost`  | Host del servidor de BD              |
| `DB_PORT`     | `5432`       | Puerto (5432 PostgreSQL, 3306 MySQL) |
| `DB_USER`     | `postgres`   | Usuario de la BD                     |
| `DB_PASSWORD` | `secret`     | Contraseña                           |
| `DB_NAME`     | `golang_api` | Nombre de la base de datos           |
| `DB_SSLMODE`  | `disable`    | Modo SSL (`disable` para desarrollo) |

---

## Cambiar de PostgreSQL a MySQL

En `cmd/main.go`, cambiá el engine:

```go
// PostgreSQL (default)
storage.New(storage.PostgreSQL)

// MySQL
storage.New(storage.MySQL)
```

Para MySQL, `DB_PORT` debe ser `3306` y `DB_SSLMODE` no aplica.

---

## Stack HTTP

El servidor usa **Echo v4** como framework de ruteo y middleware, montado sobre `http.Server` estándar de la stdlib (para conservar el control del graceful shutdown).

### Middlewares globales (registrados en `cmd/main.go`)

| Middleware       | Origen                | Función                                              |
| ---------------- | --------------------- | ---------------------------------------------------- |
| `Recover`        | `echo/v4/middleware`  | Atrapa panics, devuelve 500 limpio                   |
| `RequestLogger`  | `echo/v4/middleware`  | Loguea método, URI, status y latencia por request    |
| `BodyLimit("1M")`| `echo/v4/middleware`  | Rechaza bodies > 1 MiB (`413 Request Entity Too Large`) |

### Middleware de autenticación (selectivo)

`middleware.Authentication` (paquete propio en `middleware/middleware.go`) valida el header `Authorization: Bearer <jwt>`. Se aplica solo al grupo `/v1/persons/*`. Las rutas `/v1/login` y `/v1/signup` son públicas.

---

## Endpoints

Base URL: `http://localhost:8080`

Todas las respuestas siguen el mismo envelope:

```json
{
  "message_type": "message | error",
  "message": "Texto descriptivo",
  "data": null
}
```

### Resumen de rutas

| Método   | Ruta                       | Auth | Descripción                       |
| -------- | -------------------------- | ---- | --------------------------------- |
| `POST`   | `/v1/signup`               | ❌   | Registro de usuario               |
| `POST`   | `/v1/login`                | ❌   | Devuelve JWT                      |
| `POST`   | `/v1/persons/create`       | ✅   | Crear persona                     |
| `GET`    | `/v1/persons/get-all`      | ✅   | Listar todas las personas         |
| `GET`    | `/v1/persons/:id`          | ✅   | Obtener persona por ID            |
| `PUT`    | `/v1/persons/:id`          | ✅   | Actualizar persona por ID         |
| `DELETE` | `/v1/persons/:id`          | ✅   | Eliminar persona por ID           |

> Las rutas con `:id` usan **path param** (no query param). Ej: `GET /v1/persons/72`, no `/v1/persons/get-by-id?id=72`.

---

### `POST /v1/signup`

Crea un usuario. Endpoint público.

```bash
curl -X POST http://localhost:8080/v1/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"camilo@test.com","password":"secreto123"}'
```

Respuesta `201 Created`:

```json
{ "message_type": "message", "message": "User created", "data": { "id": 1 } }
```

---

### `POST /v1/login`

Valida credenciales y retorna un JWT firmado con RS256, válido por **1 hora**.

```bash
curl -X POST http://localhost:8080/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"camilo@test.com","password":"secreto123"}'
```

Respuesta `200 OK`:

```json
{
  "message_type": "message",
  "message": "Login OK",
  "data": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Tanto password incorrecto como email inexistente devuelven el mismo `401` con mensaje genérico, para evitar **user enumeration**.

Capturar el token automáticamente con `jq`:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"camilo@test.com","password":"secreto123"}' \
  | jq -r '.data')
```

---

### `POST /v1/persons/create`

Crea una persona con sus comunidades asociadas. Requiere token.

```bash
curl -X POST http://localhost:8080/v1/persons/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Ana García",
    "age": 28,
    "communities": [{"name":"Meli"}, {"name":"Gym"}]
  }'
```

Respuesta `201 Created`:

```json
{ "message_type": "message", "message": "Person created successfully", "data": null }
```

Errores comunes:

- `400` — JSON mal formado, validation fail (`name` vacío o > 100 chars), o campo desconocido (`DisallowUnknownFields`).
- `401` — token faltante o inválido.
- `500` — falla del storage.

---

### `GET /v1/persons/get-all`

Lista todas las personas con sus comunidades.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/v1/persons/get-all
```

Respuesta `200 OK`:

```json
{
  "message_type": "message",
  "message": "Persons fetched successfully",
  "data": [
    {
      "ID": 1,
      "CreatedAt": "2026-04-24T00:04:58Z",
      "UpdatedAt": "2026-04-24T00:04:58Z",
      "DeletedAt": null,
      "name": "Ana García",
      "age": 28,
      "communities": [
        { "ID": 1, "name": "Meli" },
        { "ID": 2, "name": "Gym" }
      ]
    }
  ]
}
```

---

### `GET /v1/persons/:id`

Obtiene una persona por su ID.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/v1/persons/1
```

Errores: `400` si el ID no es numérico, `404` si no existe.

---

### `PUT /v1/persons/:id`

Actualiza una persona por su ID.

```bash
curl -X PUT http://localhost:8080/v1/persons/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Ana María García",
    "age": 29,
    "communities": [{"name":"Meli"}, {"name":"Yoga"}]
  }'
```

Errores: `400` body inválido, `404` si la persona no existe.

---

### `DELETE /v1/persons/:id`

Elimina una persona por su ID.

```bash
curl -X DELETE http://localhost:8080/v1/persons/1 \
  -H "Authorization: Bearer $TOKEN"
```

Errores: `404` si no existe.

---

### Cargar datos de prueba

```bash
TOKEN="<jwt_obtenido_del_login>"

for payload in \
  '{"name":"Ana García","age":28,"communities":[{"name":"Meli"},{"name":"Gym"}]}' \
  '{"name":"Carlos Pérez","age":35,"communities":[{"name":"Meli"},{"name":"Running"}]}' \
  '{"name":"María López","age":22,"communities":[{"name":"Yoga"},{"name":"Lectura"}]}' \
  '{"name":"Juan Rodríguez","age":45,"communities":[{"name":"Meli"},{"name":"Ajedrez"},{"name":"Cine"}]}' \
  '{"name":"Laura Martínez","age":19,"communities":[{"name":"Nike"},{"name":"Gym"}]}' \
  '{"name":"Pedro Sánchez","age":52,"communities":[{"name":"Golf"},{"name":"Fotografía"}]}' \
  '{"name":"Sofía Ramírez","age":26,"communities":[{"name":"Meli"},{"name":"Crossfit"},{"name":"Cocina"}]}' \
  '{"name":"Diego Torres","age":31,"communities":[{"name":"Spotify"},{"name":"Netflix"},{"name":"Gamers"}]}' \
  '{"name":"Valentina Ruiz","age":24,"communities":[{"name":"Adidas"},{"name":"Pilates"}]}' \
  '{"name":"Andrés Morales","age":40,"communities":[{"name":"Meli"},{"name":"Ciclismo"}]}'
do
  curl -s -X POST http://localhost:8080/v1/persons/create \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "$payload"
  echo
done
```

---

## Autenticación

La API usa JWT firmados con RSA (algoritmo `RS256`). Flujo:

1. Generar el par de claves RSA (una sola vez por entorno).
2. Crear un usuario con `POST /v1/signup` (o vía seed binary para usuarios admin).
3. `POST /v1/login` → obtener JWT.
4. Mandar `Authorization: Bearer <jwt>` en rutas protegidas.

### Generación de claves RSA

`cmd/main.go` carga `certificates/private.pem` y `certificates/public.pem` al arrancar. Si no existen, falla con `error loading certificates`.

```bash
mkdir -p certificates
openssl genrsa -out certificates/private.pem 2048
openssl rsa -in certificates/private.pem -pubout -out certificates/public.pem
```

> Estas claves no deben commitearse. Asegurate de tener `certificates/` en `.gitignore`.

### Crear usuario admin (seed)

`cmd/seed/main.go` es un binario one-shot para crear usuarios con bcrypt:

```bash
go run ./cmd/seed -email=camilo@test.com -password=secreto123
```

Verificar la inserción:

```bash
docker exec -it <container-postgres> psql -U postgres -d golang_api \
  -c "SELECT id, email, length(password) FROM users;"
```

`length(password)` debe ser **60** (largo de un hash bcrypt). Si da menos, se guardó en claro y hay un bug.

---

## Dependencias

| Paquete                         | Versión   | Uso                              |
| ------------------------------- | --------- | -------------------------------- |
| `github.com/labstack/echo/v4`   | v4.15.x   | Framework HTTP (router + middleware) |
| `gorm.io/gorm`                  | v1.31.1   | ORM principal                    |
| `gorm.io/driver/postgres`       | v1.6.0    | Driver PostgreSQL                |
| `gorm.io/driver/mysql`          | v1.6.0    | Driver MySQL                     |
| `github.com/jackc/pgx/v5`       | v5.6.0    | Driver nativo PostgreSQL         |
| `github.com/golang-jwt/jwt/v5`  | v5.x      | Firma y verificación de JWT      |
| `github.com/go-playground/validator/v10` | v10.30.x | Validación declarativa con tags |
| `golang.org/x/crypto/bcrypt`    | latest    | Hashing de passwords             |
