package model

import "errors"

var (
	ErrNoUser             = errors.New("no such user")
	ErrIncorrectPass      = errors.New("incorrect password")
	ErrUnauthorizedAccess = errors.New("no access to this content")
	ErrNoContent          = errors.New("no content")
	ErrNoComments         = errors.New("no comments")
	ErrOccupiedLogin      = errors.New("login is taken by another user")
)
