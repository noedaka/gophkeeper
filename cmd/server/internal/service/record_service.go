package service

import (
	"context"
	"gophkeeper/cmd/server/internal/repository"
	"gophkeeper/internal/domain"
)

// RecordServ структура для сервиса записей
type RecordServ struct {
	repo repository.RecordRepository
}

// NewRecordServ создает новый экземпляр RecordServ
func NewRecordServ(repo repository.RecordRepository) *RecordServ {
	return &RecordServ{repo: repo}
}

// Create сохраняет новую зашифрованную запись и возвращает её ID
func (s *RecordServ) Create(ctx context.Context, record *domain.Record) (int, error) {
	return s.repo.Create(ctx, record)
}

// Get получает полную зашифрованную запись по ID с проверкой владения
func (s *RecordServ) Get(ctx context.Context, recordID int, userID string) (*domain.Record, error) {
	return s.repo.Get(ctx, recordID, userID)
}

// List возвращает список упрощённой информации о записях пользователя
func (s *RecordServ) List(ctx context.Context, userID string, recordType string) ([]domain.RecordInfo, error) {
	return s.repo.List(ctx, userID, recordType)
}

// Delete удаляет запись по ID с проверкой владения
func (s *RecordServ) Delete(ctx context.Context, recordID int, userID string) error {
	return s.repo.Delete(ctx, recordID, userID)
}
