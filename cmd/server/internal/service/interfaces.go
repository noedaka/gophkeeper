package service

import (
	"context"
	"gophkeeper/internal/domain"
	"io"
)

// UserService интерфейс для сервиса пользователя
type UserService interface {
	Register(ctx context.Context, user *domain.UserCredentials) (string, error)
	Login(ctx context.Context, user *domain.UserCredentials) (string, error)
}

// RecordService интерфейс для работы с записью
type RecordService interface {
	Create(ctx context.Context, record *domain.Record) (int, error)
	Get(ctx context.Context, recordID int, userID string) (*domain.Record, error)
	List(ctx context.Context, userID string, recordType string) ([]domain.RecordInfo, error)
	Delete(ctx context.Context, recordID int, userID string) error
}

// BinaryService интерфейс для работы с бинарной записью
type BinaryService interface {
	Create(ctx context.Context, userID string, metadata string) (int, string, error)
	Get(ctx context.Context, s3Key string) (io.ReadCloser, int64, error)
	List(ctx context.Context, userID string) ([]domain.BinaryRecord, error)
	Delete(ctx context.Context, recordID int, userID string) error
	Upload(ctx context.Context, s3Key string, reader io.Reader, size int64) error
	GetKey(ctx context.Context, ID int, userID string) (string, error)
}
