package service

import (
	"context"
	"gophkeeper/internal/domain"
)

type UserService interface {
	Register(ctx context.Context, user *domain.UserCredentials) (int, error)
	Login(ctx context.Context, user *domain.UserCredentials) (int, error)
}
