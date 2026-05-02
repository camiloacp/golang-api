package model

import "gorm.io/gorm"

// Community estructura de una comunidad.
//
// Tags:
//   - json:      nombre del campo en el body/respuesta JSON.
//   - gorm:      reglas del ORM (tipo de columna, nullabilidad).
//   - validate:  reglas de negocio que corre go-playground/validator.
//                `required` rechaza strings vacíos; `max=100` limita el largo.
type Community struct {
	gorm.Model
	PersonID uint   `json:"-"`
	Name     string `json:"name" gorm:"type:varchar(100);not null" validate:"required,max=100"`
}

// Communities slice de comunidades
type Communities []Community

// Person estructura de una persona.
//
// El tag `validate:"dive"` en Communities le dice al validator que baje al slice
// y aplique las reglas del struct Community a cada elemento. Sin `dive`,
// comunidades con nombre vacío pasarían silenciosas.
type Person struct {
	gorm.Model
	Name        string      `json:"name" gorm:"type:varchar(100);not null" validate:"required,max=100"`
	Age         uint8       `json:"age"`
	Communities Communities `json:"communities" gorm:"foreignKey:PersonID" validate:"dive"`
}

// Persons slice de personas
type Persons []Person
