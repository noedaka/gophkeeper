package service

import (
	"context"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/internal/domain"
)

type CardServ struct {
	repo repository.CardRepository
}

func NewCardService(repo repository.CardRepository) *CardServ {
	return &CardServ{repo: repo}
}

func (s *CardServ) Create(ctx context.Context, card *domain.Card) (int, error) {
	return s.repo.Create(ctx, card)
}

func (s *CardServ) Get(ctx context.Context, cardID int, userID int) (*domain.Card, error) {
	return s.repo.Get(ctx, cardID, userID)
}

func (s *CardServ) ListIDs(ctx context.Context, userID int) ([]int, error) {
	return s.repo.ListIDs(ctx, userID)
}

func (s *CardServ) Delete(ctx context.Context, cardID int, userID int) error {
	return s.repo.Delete(ctx, cardID, userID)
}
