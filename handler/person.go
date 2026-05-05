package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang-api/model"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// validate es la instancia compartida del validator.
// Crear un *validator.Validate es caro (cachea reflection), así que se
// instancia una sola vez a nivel de paquete y se reutiliza. Es thread-safe.
var validate = validator.New(validator.WithRequiredStructEnabled())

// person agrupa los handlers HTTP del recurso /persons y mantiene la dependencia al storage.
type person struct {
	storage Storage
}

// newPerson construye un handler de personas inyectando la implementación de Storage.
func newPerson(s Storage) *person {
	return &person{storage: s}
}

// create atiende POST /v1/persons/create — decodifica el body, persiste la persona y responde 201.
func (p *person) create(c echo.Context) error {
	data, err := decodePerson(c)
	if err != nil {
		log.Printf("handler.person.create: decode/validate failed: %v", err)
		return c.JSON(http.StatusBadRequest, newResponse(Error, decodeErrorMessage(err), nil))
	}

	if err := p.storage.Create(&data); err != nil {
		log.Printf("handler.person.create: storage.Create failed for name=%q: %v", data.Name, err)
		return c.JSON(http.StatusInternalServerError, newResponse(Error, "Failed to create person", nil))
	}

	log.Printf("handler.person.create: person created id=%d name=%q", data.ID, data.Name)
	return c.JSON(http.StatusCreated, newResponse(Message, "Person created successfully", nil))
}

// update atiende PUT /v1/persons/:id — reemplaza los campos de la persona indicada.
func (p *person) update(c echo.Context) error {
	rawID := c.Param("id")
	ID, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("handler.person.update: invalid id %q: %v", rawID, err)
		return c.JSON(http.StatusBadRequest, newResponse(Error, "Invalid ID", nil))
	}

	data, err := decodePerson(c)
	if err != nil {
		log.Printf("handler.person.update: decode/validate failed for id=%d: %v", ID, err)
		return c.JSON(http.StatusBadRequest, newResponse(Error, decodeErrorMessage(err), nil))
	}

	err = p.storage.Update(ID, &data)
	if errors.Is(err, model.ErrIDPersonDoesNotExists) {
		log.Printf("handler.person.update: person does not exist id=%d", ID)
		return c.JSON(http.StatusNotFound, newResponse(Error, "Person does not exist", nil))
	}
	if err != nil {
		log.Printf("handler.person.update: storage.Update failed for id=%d: %v", ID, err)
		return c.JSON(http.StatusInternalServerError, newResponse(Error, "Failed to update person", nil))
	}

	log.Printf("handler.person.update: person updated id=%d", ID)
	return c.JSON(http.StatusOK, newResponse(Message, "Person updated successfully", nil))
}

// delete atiende DELETE /v1/persons/:id — distingue "no existe" (404) de fallos internos (500).
func (p *person) delete(c echo.Context) error {
	rawID := c.Param("id")
	ID, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("handler.person.delete: invalid id %q: %v", rawID, err)
		return c.JSON(http.StatusBadRequest, newResponse(Error, "Invalid ID", nil))
	}

	err = p.storage.Delete(ID)
	if errors.Is(err, model.ErrIDPersonDoesNotExists) {
		log.Printf("handler.person.delete: person does not exist id=%d", ID)
		return c.JSON(http.StatusNotFound, newResponse(Error, "Person does not exist", nil))
	}
	if err != nil {
		log.Printf("handler.person.delete: storage.Delete failed for id=%d: %v", ID, err)
		return c.JSON(http.StatusInternalServerError, newResponse(Error, "Failed to delete person", nil))
	}

	log.Printf("handler.person.delete: person deleted id=%d", ID)
	return c.JSON(http.StatusOK, newResponse(Message, "Person deleted successfully", nil))
}

// getByID atiende GET /v1/persons/:id.
func (p *person) getByID(c echo.Context) error {
	rawID := c.Param("id")
	ID, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("handler.person.getByID: invalid id %q: %v", rawID, err)
		return c.JSON(http.StatusBadRequest, newResponse(Error, "Invalid ID", nil))
	}

	resp, err := p.storage.GetByID(ID)
	if errors.Is(err, model.ErrIDPersonDoesNotExists) {
		log.Printf("handler.person.getByID: person does not exist id=%d", ID)
		return c.JSON(http.StatusNotFound, newResponse(Error, "Person does not exist", nil))
	}
	if err != nil {
		log.Printf("handler.person.getByID: storage.GetByID failed for id=%d: %v", ID, err)
		return c.JSON(http.StatusInternalServerError, newResponse(Error, "Failed to get person by ID", nil))
	}

	log.Printf("handler.person.getByID: person fetched id=%d", ID)
	return c.JSON(http.StatusOK, newResponse(Message, "Person fetched successfully", resp))
}

// getAll atiende GET /v1/persons/get-all — retorna el listado completo envuelto en el response estándar.
func (p *person) getAll(c echo.Context) error {
	resp, err := p.storage.GetAll()
	if err != nil {
		log.Printf("handler.person.getAll: storage.GetAll failed: %v", err)
		return c.JSON(http.StatusInternalServerError, newResponse(Error, "Failed to get all persons", nil))
	}

	log.Printf("handler.person.getAll: fetched %d persons", len(resp))
	return c.JSON(http.StatusOK, newResponse(Message, "Persons fetched successfully", resp))
}

// decodePerson lee el body, lo decodifica a model.Person y corre las
// validaciones de los tags `validate:"..."` definidos en el struct.
//
// El límite de tamaño del body se aplica globalmente vía echomw.BodyLimit en main.go,
// así que no se duplica acá. DisallowUnknownFields rechaza JSON con campos que
// no existen en el struct (evita typos silenciosos tipo "naem" → Name vacío).
func decodePerson(c echo.Context) (model.Person, error) {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()

	var p model.Person
	if err := decoder.Decode(&p); err != nil {
		return p, err
	}
	if err := validate.Struct(p); err != nil {
		return p, err
	}
	return p, nil
}

// decodeErrorMessage produce el mensaje que se devuelve al cliente ante un
// error de decode/validación. Solo expone detalles si el error viene del
// validator; para errores de JSON parsing devuelve un mensaje genérico para
// no filtrar internals del decoder.
func decodeErrorMessage(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		msgs := make([]string, 0, len(ve))
		for _, fe := range ve {
			msgs = append(msgs, fieldErrorMessage(fe))
		}
		return strings.Join(msgs, ", ")
	}
	return "Invalid request body"
}

// fieldErrorMessage traduce un FieldError del validator a un mensaje amigable.
// Si agregás tags nuevos (ej. email, uuid), extendé este switch.
func fieldErrorMessage(fe validator.FieldError) string {
	field := jsonPath(fe)
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "max":
		return fmt.Sprintf("%s must be at most %s chars", field, fe.Param())
	case "min":
		return fmt.Sprintf("%s must be at least %s chars", field, fe.Param())
	default:
		return fmt.Sprintf("%s is invalid (%s)", field, fe.Tag())
	}
}

// jsonPath convierte el namespace del validator ("Person.Communities[0].Name")
// a un path estilo JSON ("communities[0].name"). Cosmético, pero así el
// mensaje de error habla el mismo "idioma" que el body que mandó el cliente.
func jsonPath(fe validator.FieldError) string {
	ns := fe.StructNamespace()
	if i := strings.Index(ns, "."); i >= 0 {
		ns = ns[i+1:]
	}
	return strings.ToLower(ns)
}
