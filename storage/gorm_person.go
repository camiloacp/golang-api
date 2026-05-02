package storage

import (
	"errors"
	"fmt"
	"golang-api/model"

	"gorm.io/gorm"
)

// GormPerson es el repositorio de personas en la base de datos
type GormPerson struct {
	db *gorm.DB
}

// NewGormPerson crea una nueva instancia de GormPerson
func NewGormPerson(db *gorm.DB) *GormPerson {
	return &GormPerson{db: db}
}

// Create crea una nueva persona en la base de datos
func (g *GormPerson) Create(person *model.Person) error {
	return g.db.Create(person).Error
}

// GetAll obtiene todas las personas en la base de datos
func (g *GormPerson) GetAll() (model.Persons, error) {
	var persons model.Persons
	err := g.db.Preload("Communities").Find(&persons).Error
	return persons, err
}

// GetByID obtiene una persona por su ID
func (g *GormPerson) GetByID(ID int) (model.Person, error) {
	var person model.Person
	err := g.db.Preload("Communities").First(&person, uint(ID)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return person, fmt.Errorf("ID: %d: %w", ID, model.ErrIDPersonDoesNotExists)
	}
	return person, err
}

// Update actualiza una persona en la base de datos
func (g *GormPerson) Update(ID int, person *model.Person) error {
	if person == nil {
		return model.ErrPersonCanNotBeNil
	}

	return g.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verificar que la persona existe (sin esto, Save haría upsert).
		var existing model.Person
		if err := tx.First(&existing, uint(ID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("ID: %d: %w", ID, model.ErrIDPersonDoesNotExists)
			}
			return err
		}

		// 2. Borrar comunidades viejas para evitar huérfanas.
		if err := tx.Where("person_id = ?", ID).Delete(&model.Community{}).Error; err != nil {
			return err
		}

		// 3. Asignar el ID y guardar (las nuevas comunidades vienen sin ID, se insertan limpio).
		person.ID = uint(ID)
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(person).Error
	})
}

// Delete elimina una persona en la base de datos
func (g *GormPerson) Delete(ID int) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verificar existencia (para devolver 404 correcto).
		var existing model.Person
		if err := tx.First(&existing, uint(ID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("ID: %d: %w", ID, model.ErrIDPersonDoesNotExists)
			}
			return err
		}

		// 2. Borrar comunidades (hard-delete porque no tienen sentido huérfanas).
		if err := tx.Unscoped().Where("person_id = ?", ID).Delete(&model.Community{}).Error; err != nil {
			return err
		}

		// 3. Borrar persona (soft-delete por gorm.Model).
		return tx.Delete(&model.Person{}, uint(ID)).Error
	})
}
