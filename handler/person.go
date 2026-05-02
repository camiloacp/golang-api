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
)

// validate es la instancia compartida del validator.
// Crear un *validator.Validate es caro (cachea reflection), así que se
// instancia una sola vez a nivel de paquete y se reutiliza. Es thread-safe.
var validate = validator.New(validator.WithRequiredStructEnabled())

// maxBodyBytes limita el tamaño del request body a 1 MiB.
// Protege contra clientes maliciosos que mandan bodies enormes para agotar RAM.
const maxBodyBytes = 1 << 20

// person agrupa los handlers HTTP del recurso /persons y mantiene la dependencia al storage.
type person struct {
	storage Storage
}

// newPerson construye un handler de personas inyectando la implementación de Storage.
func newPerson(s Storage) *person {
	return &person{storage: s}
}

// create atiende POST /v1/persons/create — decodifica el body, persiste la persona y responde 201.
func (p *person) create(w http.ResponseWriter, r *http.Request) {
	data, err := decodePerson(w, r)
	if err != nil {
		log.Printf("handler.person.create: decode/validate failed: %v", err)
		response := newResponse(Error, decodeErrorMessage(err), nil)
		responseJSON(w, http.StatusBadRequest, response)
		return
	}

	err = p.storage.Create(&data)
	if err != nil {
		log.Printf("handler.person.create: storage.Create failed for name=%q: %v", data.Name, err)
		response := newResponse(Error, "Failed to create person", nil)
		responseJSON(w, http.StatusInternalServerError, response)
		return
	}

	log.Printf("handler.person.create: person created id=%d name=%q", data.ID, data.Name)
	response := newResponse(Message, "Person created successfully", nil)
	responseJSON(w, http.StatusCreated, response)
}

// update atiende PUT /v1/persons/update?id={id} — reemplaza los campos de la persona indicada.
func (p *person) update(w http.ResponseWriter, r *http.Request) {
	rawID := r.URL.Query().Get("id")
	ID, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("handler.person.update: invalid id %q: %v", rawID, err)
		response := newResponse(Error, "Invalid ID", nil)
		responseJSON(w, http.StatusBadRequest, response)
		return
	}

	data, err := decodePerson(w, r)
	if err != nil {
		log.Printf("handler.person.update: decode/validate failed for id=%d: %v", ID, err)
		response := newResponse(Error, decodeErrorMessage(err), nil)
		responseJSON(w, http.StatusBadRequest, response)
		return
	}

	err = p.storage.Update(ID, &data)
	if errors.Is(err, model.ErrIDPersonDoesNotExists) {
		log.Printf("handler.person.update: person does not exist id=%d", ID)
		response := newResponse(Error, "Person does not exist", nil)
		responseJSON(w, http.StatusNotFound, response)
		return
	} else if err != nil {
		log.Printf("handler.person.update: storage.Update failed for id=%d: %v", ID, err)
		response := newResponse(Error, "Failed to update person", nil)
		responseJSON(w, http.StatusInternalServerError, response)
		return
	}

	log.Printf("handler.person.update: person updated id=%d", ID)
	response := newResponse(Message, "Person updated successfully", nil)
	responseJSON(w, http.StatusOK, response)
}

// delete atiende DELETE /v1/persons/delete?id={id} — distingue "no existe" (404) de fallos internos (500).
func (p *person) delete(w http.ResponseWriter, r *http.Request) {
	rawID := r.URL.Query().Get("id")
	ID, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("handler.person.delete: invalid id %q: %v", rawID, err)
		response := newResponse(Error, "Invalid ID", nil)
		responseJSON(w, http.StatusBadRequest, response)
		return
	}

	err = p.storage.Delete(ID)
	if errors.Is(err, model.ErrIDPersonDoesNotExists) {
		log.Printf("handler.person.delete: person does not exist id=%d", ID)
		response := newResponse(Error, "Person does not exist", nil)
		responseJSON(w, http.StatusNotFound, response)
		return
	} else if err != nil {
		log.Printf("handler.person.delete: storage.Delete failed for id=%d: %v", ID, err)
		response := newResponse(Error, "Failed to delete person", nil)
		responseJSON(w, http.StatusInternalServerError, response)
		return
	} else {
		log.Printf("handler.person.delete: person deleted id=%d", ID)
		response := newResponse(Message, "Person deleted successfully", nil)
		responseJSON(w, http.StatusOK, response)
	}
}

func (p *person) getByID(w http.ResponseWriter, r *http.Request) {
	rawID := r.URL.Query().Get("id")
	ID, err := strconv.Atoi(rawID)
	if err != nil {
		log.Printf("handler.person.getByID: invalid id %q: %v", rawID, err)
		response := newResponse(Error, "Invalid ID", nil)
		responseJSON(w, http.StatusBadRequest, response)
		return
	}

	resp, err := p.storage.GetByID(ID)
	if errors.Is(err, model.ErrIDPersonDoesNotExists) {
		log.Printf("handler.person.getByID: person does not exist id=%d", ID)
		response := newResponse(Error, "Person does not exist", nil)
		responseJSON(w, http.StatusNotFound, response)
		return
	} else if err != nil {
		log.Printf("handler.person.getByID: storage.GetByID failed for id=%d: %v", ID, err)
		response := newResponse(Error, "Failed to get person by ID", nil)
		responseJSON(w, http.StatusInternalServerError, response)
		return
	}

	log.Printf("handler.person.getByID: person fetched id=%d", ID)
	response := newResponse(Message, "Person fetched successfully", resp)
	responseJSON(w, http.StatusOK, response)
}

// getAll atiende GET /v1/persons/get-all — retorna el listado completo envuelto en el response estándar.
func (p *person) getAll(w http.ResponseWriter, r *http.Request) {
	resp, err := p.storage.GetAll()
	if err != nil {
		log.Printf("handler.person.getAll: storage.GetAll failed: %v", err)
		response := newResponse(Error, "Failed to get all persons", nil)
		responseJSON(w, http.StatusInternalServerError, response)
		return
	}

	log.Printf("handler.person.getAll: fetched %d persons", len(resp))
	response := newResponse(Message, "Persons fetched successfully", resp)
	responseJSON(w, http.StatusOK, response)
}

// decodePerson lee el body, lo decodifica a model.Person y corre las
// validaciones de los tags `validate:"..."` definidos en el struct.
// Centraliza 3 protecciones que antes no existían:
//  1. MaxBytesReader: limita el tamaño del body (DoS por body gigante).
//  2. DisallowUnknownFields: rechaza JSON con campos que no existen en el
//     struct. Evita typos silenciosos tipo "naem" que dejarían Name vacío.
//  3. validator.Struct: aplica reglas de negocio declaradas en los tags
//     (required, max, dive, etc.).
func decodePerson(w http.ResponseWriter, r *http.Request) (model.Person, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	decoder := json.NewDecoder(r.Body)
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
// error de decode/validación. Distinguimos dos fuentes:
//   - validator.ValidationErrors: seguro exponer (es copy controlado por nosotros).
//   - cualquier otro (json.Decoder, MaxBytesReader): mensaje genérico para no
//     filtrar detalles internos del parser.
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
// Centraliza el "copy" de cada tag; si agregás tags nuevos (ej. email, uuid),
// extendé este switch.
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
