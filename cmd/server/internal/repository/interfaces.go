package repository

import (
	"context"
	"gophkeeper/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.UserCredentials) (int, error)
	GetIDByCreds(ctx context.Context, user *domain.UserCredentials) (int, error)
}

type RecordRepository interface {
	Create(ctx context.Context, record *domain.Record) (int, error)
	Get(ctx context.Context, recordID int, userID int) (*domain.Record, error)
	List(ctx context.Context, userID int, recordType string) ([]domain.RecordInfo, error)
	Delete(ctx context.Context, recordID int, userID int) error
}
