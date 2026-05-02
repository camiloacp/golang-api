package main

import (
	"flag"
	"golang-api/model"
	"golang-api/storage"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	email := flag.String("email", "", "email del usuario a crear")
	password := flag.String("password", "", "password en claro (se hashea con bcrypt)")
	flag.Parse()

	if *email == "" || *password == "" {
		log.Fatal("uso: seed -email=<email> -password=<password>")
	}
	if len(*password) < 8 {
		log.Fatal("password debe tener al menos 8 caracteres")
	}

	// Reutilizamos la misma inicialización que usa el server para garantizar
	// que las tablas existen antes de insertar.
	store := storage.New(storage.PostgreSQL)
	if err := store.Migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// bcrypt.DefaultCost (10) está bien para dev. En prod considerá 12+.
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}

	user := model.User{Email: *email, Password: string(hash)}
	if err := store.CreateUser(&user); err != nil {
		log.Fatalf("create user: %v", err)
	}

	log.Printf("usuario creado: id=%d email=%s", user.ID, user.Email)
}
