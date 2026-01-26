package repository

import (
	"context"
	"gophkeeper/internal/domain"
)

// UserRepository интерфейс для авторизации
type UserRepository interface {
	Create(ctx context.Context, user *domain.UserCredentials) (string, error)
	GetIDByCreds(ctx context.Context, user *domain.UserCredentials) (string, error)
}

// RecordRepository интерфейс записи
type RecordRepository interface {
	Create(ctx context.Context, record *domain.Record) (int, error)
	Get(ctx context.Context, recordID int, userID string) (*domain.Record, error)
	List(ctx context.Context, userID string, recordType string) ([]domain.RecordInfo, error)
	Delete(ctx context.Context, recordID int, userID string) error
}

// BinaryRepository интерфейс бинарной записи
type BinaryRepository interface {
	Create(ctx context.Context, userID string, metadata string) (int, string, error)
	GetKey(ctx context.Context, ID int, userID string) (string, error)
	List(ctx context.Context, userID string) ([]domain.BinaryRecord, error)
	Delete(ctx context.Context, recordID int, userID string) error
}
