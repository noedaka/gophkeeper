package repository

import (
	"context"
	"gophkeeper/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.UserCredentials) (int, error)
	GetIDByCreds(ctx context.Context, user *domain.UserCredentials) (int, error)
}

type CardRepository interface {
	Create(ctx context.Context, card *domain.Card) (int, error)
	Get(ctx context.Context, cardID int, userID int) (*domain.Card, error)
	ListIDs(ctx context.Context, userID int) ([]int, error)
	Delete(ctx context.Context, cardID int, userID int) error
}

type CredsRepository interface {
	Create(ctx context.Context, creds *domain.Creds) (int, error)
	Get(ctx context.Context, credsID int, userID int) (*domain.Creds, error)
	ListIDs(ctx context.Context, userID int) ([]int, error)
	Delete(ctx context.Context, credsID int, userID int) error
}
