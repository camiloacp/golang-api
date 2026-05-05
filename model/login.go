package model

import (
	"github.com/golang-jwt/jwt/v5"
)

type Login struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginToken struct {
	Token string `json:"token"`
}

type Claim struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}
