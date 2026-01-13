package service

import (
	"context"
	"gophkeeper/internal/domain"
)

type UserService interface {
	Register(ctx context.Context, user *domain.UserCredentials) (int, error)
	Login(ctx context.Context, user *domain.UserCredentials) (int, error)
}

type CardService interface {
	Create(ctx context.Context, card *domain.Card) (int, error)
	Get(ctx context.Context, cardID int, userID int) (*domain.Card, error)
	ListIDs(ctx context.Context, userID int) ([]int, error)
	Delete(ctx context.Context, cardID int, userID int) error
}

type CredsService interface {
	Create(ctx context.Context, creds *domain.Creds) (int, error)
	Get(ctx context.Context, credsID int, userID int) (*domain.Creds, error)
	ListIDs(ctx context.Context, userID int) ([]int, error)
	Delete(ctx context.Context, credsID int, userID int) error
}
