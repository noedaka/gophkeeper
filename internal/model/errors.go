package model

import "errors"

type ContextKey string

const UserIDKey ContextKey = "user_id"

var JWTSecret = []byte("my-super-secret-key-for-testing")

var (
	ErrNoUser        = errors.New("no such user")
	ErrIncorrectPass = errors.New("incorrect password")
	ErrNoPosts       = errors.New("no posts")
	ErrNoComments    = errors.New("no comments")
	ErrOccupiedLogin = errors.New("login is taken by another user")
)
