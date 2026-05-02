# golang-api

Servidor HTTP en Go con persistencia en PostgreSQL usando GORM. Soporta también MySQL mediante un switch de engine.

## Arquitectura

```
golang-api/
├── main.go                  # Inicialización DB + servidor HTTP
├── model/
│   ├── model.go             # Errores del dominio
│   └── person.go            # Structs Person y Community (con tags GORM)
├── pkg/
│   └── person/
│       └── person.go        # Interfaz Storage + Service layer
└── storage/
    ├── storage.go           # Singleton GORM (PostgreSQL / MySQL)
    ├── gorm_person.go       # Repositorio CRUD con GORM
    └── memory.go            # Repositorio en memoria (referencia / tests)
```

### Capas

| Capa         | Paquete       | Responsabilidad                                 |
| ------------ | ------------- | ----------------------------------------------- |
| Datos        | `model/`      | Definición de structs y errores                 |
| Persistencia | `storage/`    | Acceso a BD (GORM) o memoria                    |
| Negocio      | `pkg/person/` | Interfaz `Storage` + `Service` con validaciones |

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
- Docker y Docker Compose

---

## Ejecución con Docker

```bash
docker-compose up --build
```

Levanta dos servicios:

- `postgres` — imagen `postgres:17-alpine`, puerto `5432`
- `app` — la API compilada, puerto `8080`

La app espera a que PostgreSQL esté listo (healthcheck) antes de arrancar. Las tablas `persons` y `communities` se crean automáticamente via `AutoMigrate`.

---

## Ejecución local

Requiere una instancia de PostgreSQL corriendo en `localhost:5432`.

```bash
cp .env.example .env
# edita .env con tus credenciales
go run main.go
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

En `main.go`, cambia el engine:

```go
// PostgreSQL (default)
storage.New(storage.PostgreSQL)

// MySQL
storage.New(storage.MySQL)
```

Para MySQL, `DB_PORT` debe ser `3306` y `DB_SSLMODE` no aplica.

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

### `POST /v1/persons/create`

Crea una persona con sus comunidades asociadas.

**Request body:**

```json
{
  "name": "Ana García",
  "age": 28,
  "communities": [{ "name": "Meli" }, { "name": "Gym" }]
}
```

**curl:**

```bash
curl -X POST http://localhost:8080/v1/persons/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ana García",
    "age": 28,
    "communities": [{"name": "Meli"}, {"name": "Gym"}]
  }'
```

**Respuesta (201 Created):**

```json
{
  "message_type": "message",
  "message": "Person created successfully",
  "data": null
}
```

**Errores comunes:**

- `400 Bad Request` — JSON mal formado o `age` fuera de rango (0–255 por ser `uint8`).
- `500 Internal Server Error` — Falla en la persistencia.

---

### `GET /v1/persons/get-all`

Retorna todas las personas con sus comunidades.

**curl:**

```bash
curl http://localhost:8080/v1/persons/get-all
```

**Respuesta (200 OK):**

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

### `PUT /v1/persons/update?id={id}`

Actualiza una persona por su ID (pasado como query param).

**curl:**

```bash
curl -X PUT 'http://localhost:8080/v1/persons/update?id=1' \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ana María García",
    "age": 29,
    "communities": [{"name": "Meli"}, {"name": "Yoga"}]
  }'
```

**Respuesta (200 OK):**

```json
{
  "message_type": "message",
  "message": "Person updated successfully",
  "data": null
}
```

**Errores comunes:**

- `400 Bad Request` — `id` no numérico o body inválido.
- `500 Internal Server Error` — La persona no existe o falla la BD.

---

### Comportamiento por método HTTP

Cada ruta acepta únicamente un método específico. Cualquier otro retorna `405 Method Not Allowed` con el header `Allow` indicando el método válido:

| Ruta                  | Método |
| --------------------- | ------ |
| `/v1/persons/create`  | `POST` |
| `/v1/persons/get-all` | `GET`  |
| `/v1/persons/update`  | `PUT`  |

Ejemplo:

```bash
$ curl -v http://localhost:8080/v1/persons/create
< HTTP/1.1 405 Method Not Allowed
< Allow: POST
```

---

### Cargar datos de prueba (20 personas)

Para poblar la BD rápidamente:

```bash
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
  '{"name":"Andrés Morales","age":40,"communities":[{"name":"Meli"},{"name":"Ciclismo"}]}' \
  '{"name":"Camila Herrera","age":17,"communities":[{"name":"K-pop"},{"name":"Anime"},{"name":"TikTok"}]}' \
  '{"name":"Jorge Castillo","age":60,"communities":[{"name":"Jardinería"},{"name":"Café"}]}' \
  '{"name":"Isabella Díaz","age":29,"communities":[{"name":"Meli"},{"name":"Viajes"},{"name":"Idiomas"}]}' \
  '{"name":"Miguel Ángel","age":33,"communities":[{"name":"Parrilla"},{"name":"Fútbol"},{"name":"Cerveza Artesanal"}]}' \
  '{"name":"Gabriela Vargas","age":27,"communities":[{"name":"Meli"},{"name":"Startup"},{"name":"Café"}]}' \
  '{"name":"Ricardo Mendoza","age":48,"communities":[{"name":"Tenis"},{"name":"Whisky"}]}' \
  '{"name":"Daniela Rojas","age":21,"communities":[{"name":"Gym"},{"name":"Nutrición"},{"name":"Running"}]}' \
  '{"name":"Felipe Ortiz","age":38,"communities":[{"name":"Meli"},{"name":"Surf"},{"name":"Música"}]}' \
  '{"name":"Natalia Gómez","age":25,"communities":[{"name":"Crossfit"},{"name":"Nike"},{"name":"Podcasts"}]}' \
  '{"name":"Tomás Vega","age":42,"communities":[{"name":"Meli"},{"name":"Escalada"},{"name":"Viajes"}]}'
do
  curl -s -X POST http://localhost:8080/v1/persons/create \
    -H "Content-Type: application/json" -d "$payload"
  echo
done
```

---

## Autenticación

La API usa JWT firmados con RSA (algoritmo `RS256`). El flujo es:

1. Generar el par de claves RSA (una sola vez por entorno).
2. Crear un usuario admin con el binario de seed (passwords se almacenan como hash bcrypt).
3. Hacer `POST /v1/login` para obtener un token.
4. Mandar el token en el header `Authorization: Bearer <token>` para acceder a rutas protegidas.

### Generación de claves RSA

`cmd/main.go` carga `certificates/private.pem` y `certificates/public.pem` al arrancar. Si no existen, el server falla con `error loading certificates`. Generalas con OpenSSL:

```bash
mkdir -p certificates
openssl genrsa -out certificates/private.pem 2048
openssl rsa -in certificates/private.pem -pubout -out certificates/public.pem
```

> Estas claves no deben commitearse. Asegurate de tener `certificates/` en `.gitignore`.

### Crear el usuario admin (seed)

`cmd/seed/main.go` es un binario one-shot que reusa la capa `storage` para insertar un usuario con password ya hasheado con bcrypt:

```bash
go run ./cmd/seed -email=camilo@test.com -password=secreto123
```

Salida esperada:

```
Database PostgreSQL connected successfully
usuario creado: id=1 email=camilo@test.com
```

Verificá la inserción contra la BD:

```bash
docker exec -it <container-postgres> psql -U postgres -d golang_api \
  -c "SELECT id, email, length(password) FROM users;"
```

`length(password)` debe ser **60** — el largo fijo de un hash bcrypt. Si te da menos, el password se guardó en claro y hay un bug.

Comandos útiles para inspección manual:

```bash
# listar tablas
docker exec -it <container-postgres> psql -U postgres -d golang_api -c '\dt'

# entrar al psql interactivo
docker exec -it <container-postgres> psql -U postgres -d golang_api
```

---

### `POST /v1/login`

Valida credenciales y retorna un JWT firmado con RS256, válido por 5 minutos.

**Request body:**

```json
{
  "email": "camilo@test.com",
  "password": "secreto123"
}
```

**Login válido (200 OK):**

```bash
curl -i -X POST http://localhost:8080/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"camilo@test.com","password":"secreto123"}'
```

```json
{
  "message_type": "message",
  "message": "Login OK",
  "data": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Credenciales inválidas (401 Unauthorized):**

```bash
# password incorrecto
curl -i -X POST http://localhost:8080/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"camilo@test.com","password":"wrongpass"}'

# email inexistente
curl -i -X POST http://localhost:8080/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"noexiste@test.com","password":"secreto123"}'
```

Ambos casos retornan el mismo mensaje genérico `"email or password is incorrect"` a propósito, para evitar **user enumeration** (un atacante no puede inferir qué emails están registrados midiendo respuestas distintas).

**Validación de formato (400 Bad Request):**

```bash
curl -i -X POST http://localhost:8080/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"no-es-email","password":"secreto123"}'
```

Dispara el validator (`required,email` / `required,min=8`) antes de tocar la BD.

---

### Usar el token en rutas protegidas

`POST /v1/persons/create` está cubierto por `middleware.Authentication`. Pasá el JWT como bearer:

```bash
TOKEN="<pegá_acá_el_jwt_devuelto_por_login>"

curl -i -X POST http://localhost:8080/v1/persons/create \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Juan","age":30,"communities":[{"name":"Devs"}]}'
```

> El token expira a los **5 minutos** (definido en `authorization/token.go`). Si la prueba demora, repetí el login para obtener uno nuevo.

---

## Dependencias

| Paquete                    | Versión | Uso                      |
| -------------------------- | ------- | ------------------------ |
| `gorm.io/gorm`             | v1.31.1 | ORM principal            |
| `gorm.io/driver/postgres`  | v1.6.0  | Driver PostgreSQL        |
| `gorm.io/driver/mysql`     | v1.6.0  | Driver MySQL             |
| `github.com/jackc/pgx/v5`  | v5.6.0  | Driver nativo PostgreSQL |
| `github.com/golang-jwt/jwt/v5` | v5.x | Firma y verificación de JWT |
| `golang.org/x/crypto/bcrypt`   | latest | Hashing de passwords     |
