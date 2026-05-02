package storage

import (
	"fmt"
	"log"
	"os"

	"golang-api/model"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBEngine uint8

const (
	PostgreSQL DBEngine = iota + 1
	MySQL
)

// Store es el almacenamiento principal de la aplicación
type Store struct {
	db     *gorm.DB
	person *GormPerson
	login  *GormLogin
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// New crea una nueva instancia de Store conectada a la BD indicada
func New(engine DBEngine) *Store {
	var (
		dialect gorm.Dialector
		name    string
	)

	switch engine {
	case PostgreSQL:
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "secret"),
			getEnv("DB_NAME", "golang_api"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_SSLMODE", "disable"),
		)
		dialect = postgres.Open(dsn)
		name = "PostgreSQL"

	case MySQL:
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			getEnv("DB_USER", "root"),
			getEnv("DB_PASSWORD", "root"),
			getEnv("DB_HOST", "127.0.0.1"),
			getEnv("DB_PORT", "3306"),
			getEnv("DB_NAME", "golang_api"),
		)
		dialect = mysql.Open(dsn)
		name = "MySQL"

	default:
		log.Fatalf("storage.New: unsupported engine %d", engine)
	}

	db, err := gorm.Open(dialect, &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Info),
		TranslateError: true,
	})
	if err != nil {
		log.Fatalf("storage.New: connecting to %s: %v", name, err)
	}

	fmt.Printf("Database %s connected successfully\n", name)

	return &Store{
		db:     db,
		person: NewGormPerson(db),
		login:  NewGormLogin(db),
	}
}

func (s *Store) Migrate() error {
	return s.db.AutoMigrate(&model.Person{}, &model.Community{}, &model.User{})
}

func (s *Store) Create(person *model.Person) error {
	return s.person.Create(person)
}

func (s *Store) Update(ID int, person *model.Person) error {
	return s.person.Update(ID, person)
}

func (s *Store) Delete(ID int) error {
	return s.person.Delete(ID)
}

func (s *Store) GetByID(ID int) (model.Person, error) {
	return s.person.GetByID(ID)
}

func (s *Store) GetAll() (model.Persons, error) {
	return s.person.GetAll()
}

func (s *Store) IsLoginValid(login model.Login) error {
	return s.login.IsLoginValid(login)
}

// CreateUser persiste un usuario nuevo. Password debe venir hasheado.
func (s *Store) CreateUser(user *model.User) error {
	return s.login.Create(user)
}
