package service

import (
	"context"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/internal/domain"
)

type RecordServ struct {
	repo repository.RecordRepository
}

func NewRecordServ(repo repository.RecordRepository) *RecordServ {
	return &RecordServ{repo: repo}
}

func (s *RecordServ) Create(ctx context.Context, record *domain.Record) (int, error) {
	return s.repo.Create(ctx, record)
}

func (s *RecordServ) Get(ctx context.Context, recordID int, userID int) (*domain.Record, error) {
	return s.repo.Get(ctx, recordID, userID)
}

func (s *RecordServ) List(ctx context.Context, userID int, recordType string) ([]domain.RecordInfo, error) {
	return s.repo.List(ctx, userID, recordType)
}

func (s *RecordServ) Delete(ctx context.Context, recordID int, userID int) error {
	return s.repo.Delete(ctx, recordID, userID)
}
