package handler

import "golang-api/model"

// Storage es la interfaz de almacenamiento de las personas
type Storage interface {
	Create(person *model.Person) error
	Update(ID int, person *model.Person) error
	Delete(ID int) error
	GetByID(ID int) (model.Person, error)
	GetAll() (model.Persons, error)

	IsLoginValid(login model.Login) error
	CreateUser(user *model.User) error
}
