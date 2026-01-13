package service

import (
	"context"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/internal/domain"
)

type CredsServ struct {
	repo repository.CredsRepository
}

func NewCredsService(repo repository.CredsRepository) *CredsServ {
	return &CredsServ{repo: repo}
}

func (s *CredsServ) Create(ctx context.Context, creds *domain.Creds) (int, error) {
	return s.repo.Create(ctx, creds)
}

func (s *CredsServ) Get(ctx context.Context, credsID int, userID int) (*domain.Creds, error) {
	return s.repo.Get(ctx, credsID, userID)
}

func (s *CredsServ) ListIDs(ctx context.Context, userID int) ([]int, error) {
	return s.repo.ListIDs(ctx, userID)
}

func (s *CredsServ) Delete(ctx context.Context, credsID int, userID int) error {
	return s.repo.Delete(ctx, credsID, userID)
}
