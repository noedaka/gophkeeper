package service

import (
	"context"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/internal/domain"
)

// UserServ структура для сервиса пользователя
type UserServ struct {
	repo repository.UserRepository
}

// NewUserService создает новый экземпляр UserServ
func NewUserService(repo repository.UserRepository) *UserServ {
	return &UserServ{repo: repo}
}

// Register региструет нового пользователя
func (s *UserServ) Register(ctx context.Context, user *domain.UserCredentials) (string, error) {
	return s.repo.Create(ctx, user)
}

// Login авторизует пользователя
func (s *UserServ) Login(ctx context.Context, user *domain.UserCredentials) (string, error) {
	return s.repo.GetIDByCreds(ctx, user)
}
