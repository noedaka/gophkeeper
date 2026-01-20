package service

import (
	"context"
	"gophkeeper/internal/domain"
	"io"
)

type UserService interface {
	Register(ctx context.Context, user *domain.UserCredentials) (int, error)
	Login(ctx context.Context, user *domain.UserCredentials) (int, error)
}

type RecordService interface {
	Create(ctx context.Context, record *domain.Record) (int, error)
	Get(ctx context.Context, recordID int, userID int) (*domain.Record, error)
	List(ctx context.Context, userID int, recordType string) ([]domain.RecordInfo, error)
	Delete(ctx context.Context, recordID int, userID int) error
}

type BinaryService interface {
	Create(ctx context.Context, userID int, metadata string) (int, string, error)
	Get(ctx context.Context, s3Key string) (io.ReadCloser, int64, error)
	List(ctx context.Context, userID int) ([]domain.BinaryRecord, error)
	Delete(ctx context.Context, recordID int, userID int) error
	Upload(ctx context.Context, s3Key string, reader io.Reader, size int64) error
	GetKey(ctx context.Context, ID, userID int) (string, error) 
}
