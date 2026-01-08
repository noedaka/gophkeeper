package repository

import (
	"context"
	"gophkeeper/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.UserCredentials) (int, error)
	GetIDByCreds(ctx context.Context, user *domain.UserCredentials) (int, error)
}
