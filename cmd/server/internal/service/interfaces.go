package service

import (
	"context"
	"gophkeeper/internal/domain"
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
