package service

import (
	"context"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/internal/domain"
)

type UserServ struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserServ {
	return &UserServ{repo: repo}
}

func (s *UserServ) Register(ctx context.Context, user *domain.UserCredentials) (string, error) {
	return s.repo.Create(ctx, user)
}

func (s *UserServ) Login(ctx context.Context, user *domain.UserCredentials) (string, error) {
	return s.repo.GetIDByCreds(ctx, user)
}
