package storage

import (
	"errors"
	"fmt"
	"golang-api/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// GormLogin es el repositorio para autenticación de usuarios.
type GormLogin struct {
	db *gorm.DB
}

// NewGormLogin crea una nueva instancia de GormLogin.
func NewGormLogin(db *gorm.DB) *GormLogin {
	return &GormLogin{db: db}
}

func (g *GormLogin) IsLoginValid(login model.Login) error {
	var u model.User
	err := g.db.Where("email = ?", login.Email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ErrInvalidCredentials
	}
	if err != nil {
		return fmt.Errorf("checking login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(login.Password)); err != nil {
		return model.ErrInvalidCredentials
	}
	return nil
}

func (g *GormLogin) Create(user *model.User) error {
	if user == nil {
		return errors.New("user can not be nil")
	}
	if err := g.db.Create(user).Error; err != nil {
		// GORM v1.25+ traduce errores nativos del driver a sentinels propios.
		// ErrDuplicatedKey cubre violaciones de unique constraint (uniqueIndex
		// en User.Email). Lo mapeamos al sentinel del dominio para que el
		// handler responda 409 sin importarle el motor de BD subyacente.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}
