package model

import "errors"

var (
	ErrPersonCanNotBeNil     = errors.New("person can not be nil")
	ErrIDPersonDoesNotExists = errors.New("id person does not exists")
	ErrInvalidCredentials    = errors.New("email or password is incorrect")
	ErrEmailAlreadyExists    = errors.New("email already registered")
	// ErrPersonAlreadyExists        = errors.New("person already exists")
	// ErrPersonNotFound             = errors.New("person not found")
	// ErrPersonInvalid              = errors.New("person invalid")
	// ErrPersonInvalidName          = errors.New("person invalid name")
	// ErrPersonInvalidAge           = errors.New("person invalid age")
	// ErrPersonInvalidCommunities   = errors.New("person invalid communities")
	// ErrPersonInvalidCommunity     = errors.New("person invalid community")
	// ErrPersonInvalidCommunityName = errors.New("person invalid community name")
)
